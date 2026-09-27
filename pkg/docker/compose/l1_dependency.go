/*
// SPDX-License-Identifier: Apache-2.0
//
// Copyright 2024 The Nodevin Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
*/

package compose

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/fiftysixcrypto/nodevin/internal/utils"
	"github.com/spf13/viper"
)

// L1Endpoints holds what an L2 node (Arbitrum, Base) needs to reach an
// Ethereum L1: an execution JSON-RPC endpoint and a consensus (beacon) REST
// endpoint that can serve blob sidecars.
type L1Endpoints struct {
	ExecutionRPCURL string
	BeaconURL       string
	// DockerNetwork is the Docker network the L2 service must join to reach
	// the L1 by container name. Empty when the L1 is external (a plain URL,
	// reachable without any Docker network membership).
	DockerNetwork string
}

func validateHTTPURL(raw, flagName string) error {
	parsed, err := url.Parse(raw)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || strings.ContainsAny(raw, " \t\r\n\"'`$\\;&|<>") {
		return fmt.Errorf("invalid --%s %q: expected an http(s) URL", flagName, raw)
	}
	return nil
}

// ResolveL1Endpoints returns the L1 endpoints an L2 node (Arbitrum, Base)
// needs, for its mainnet stack or its Sepolia stack (testnet).
//
// If --l1-execution-rpc-url and --l1-beacon-url are both set, they are used
// directly and DockerNetwork is left empty - the L2 node needs no Docker
// network membership to reach a plain external URL. Setting only one of the
// two is an error.
//
// If neither is set, nodevin auto-attaches to its own Ethereum node instead:
// it resolves the same execution/consensus client --execution-client and
// --consensus-client would pick for `nodevin start ethereum`, requires both
// of those containers to actually be running, and points at them by
// container name over their stack's own Docker network (confirmed to work
// across separate compose files - see EthereumDockerNetworkName). This is
// deliberately not automatic-and-silent about starting that node: an L2 node
// pointed at a fresh L1 fails to find the blobs it needs until the L1 has run
// long enough (see nodevin's --blob-serving documentation), so nodevin
// requires it to already be running rather than starting one just-in-time.
func ResolveL1Endpoints(testnet bool) (L1Endpoints, error) {
	execURL := strings.TrimSpace(viper.GetString("l1-execution-rpc-url"))
	beaconURL := strings.TrimSpace(viper.GetString("l1-beacon-url"))

	if execURL != "" && beaconURL != "" {
		if err := validateHTTPURL(execURL, "l1-execution-rpc-url"); err != nil {
			return L1Endpoints{}, err
		}
		if err := validateHTTPURL(beaconURL, "l1-beacon-url"); err != nil {
			return L1Endpoints{}, err
		}
		return L1Endpoints{
			ExecutionRPCURL: strings.TrimRight(execURL, "/"),
			BeaconURL:       strings.TrimRight(beaconURL, "/"),
		}, nil
	}
	if execURL != "" || beaconURL != "" {
		return L1Endpoints{}, fmt.Errorf("--l1-execution-rpc-url and --l1-beacon-url must both be set to use an external L1, or both left unset to auto-attach to nodevin's own running Ethereum node")
	}

	executionClient, err := SelectedExecutionClient()
	if err != nil {
		return L1Endpoints{}, err
	}
	consensusClient, err := SelectedConsensusClient()
	if err != nil {
		return L1Endpoints{}, err
	}
	if consensusClient == "none" {
		return L1Endpoints{}, fmt.Errorf("auto-attaching to a local L1 needs a consensus client to read blobs from; pass --consensus-client (not \"none\"), or set --l1-execution-rpc-url and --l1-beacon-url to use an external L1")
	}

	suffix := ""
	exeCmd := "nodevin start ethereum"
	if testnet {
		suffix = "-testnet"
		exeCmd += " --testnet"
	}
	exeCmd += fmt.Sprintf(" --execution-client=%s --consensus-client=%s --blob-serving", executionClient, consensusClient)

	execContainer := executionClient + suffix
	consensusContainer := consensusClient + suffix

	running, err := utils.RunningContainerNames()
	if err != nil {
		return L1Endpoints{}, fmt.Errorf("failed to check for a running Ethereum node: %w", err)
	}
	runningSet := make(map[string]bool, len(running))
	for _, name := range running {
		runningSet[name] = true
	}

	var missing []string
	if !runningSet[execContainer] {
		missing = append(missing, execContainer)
	}
	if !runningSet[consensusContainer] {
		missing = append(missing, consensusContainer)
	}
	if len(missing) > 0 {
		return L1Endpoints{}, fmt.Errorf("no external L1 given (--l1-execution-rpc-url/--l1-beacon-url) and nodevin's own Ethereum node is not fully running (missing: %s); start one first (e.g. `%s`), or point at an external L1 with --l1-execution-rpc-url/--l1-beacon-url",
			strings.Join(missing, ", "), exeCmd)
	}

	consensusPort, ok := utils.NetworkDefaultRPCPorts()[consensusClient]
	if !ok {
		return L1Endpoints{}, fmt.Errorf("no known default RPC port for consensus client: %s", consensusClient)
	}

	return L1Endpoints{
		ExecutionRPCURL: fmt.Sprintf("http://%s:8545", execContainer),
		BeaconURL:       fmt.Sprintf("http://%s:%d", consensusContainer, consensusPort),
		DockerNetwork:   EthereumDockerNetworkName(suffix),
	}, nil
}
