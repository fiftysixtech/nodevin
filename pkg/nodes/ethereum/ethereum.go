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

package ethereum

import (
	"fmt"
	"strings"

	"github.com/fiftysixcrypto/nodevin/internal/logger"
	"github.com/fiftysixcrypto/nodevin/internal/utils"
	"github.com/fiftysixcrypto/nodevin/pkg/docker"
	"github.com/fiftysixcrypto/nodevin/pkg/docker/compose"
	"github.com/spf13/viper"
)

// consensusClientComposeConfig returns the right one of the five per-client
// builder functions for consensusClient. A single shared function per client
// (rather than one generic function switching internally) mirrors
// docs/ethereum-execution-consensus-pairing.md's own reasoning: the
// per-client differences (flag names, "--flag=value" requirement, hard-fail
// vs. graceful startup) are large enough that a shared abstraction would
// just be a switch statement wearing a trench coat.
//
// network is the consensus client's own registry name: "lighthouse" on
// mainnet, "lighthouse-testnet" on Sepolia.
func consensusClientComposeConfig(consensusClient, network string) (compose.NetworkConfig, error) {
	switch consensusClient {
	case "lighthouse":
		return compose.GetLighthouseNetworkComposeConfig(network)
	case "prysm":
		return compose.GetPrysmNetworkComposeConfig(network)
	case "teku":
		return compose.GetTekuNetworkComposeConfig(network)
	case "nimbus":
		return compose.GetNimbusNetworkComposeConfig(network)
	case "lodestar":
		return compose.GetLodestarNetworkComposeConfig(network)
	default:
		return compose.NetworkConfig{}, fmt.Errorf("unsupported --consensus-client: %s", consensusClient)
	}
}

// conflictingContainers returns the running Ethereum stack containers, from
// either the mainnet or the Sepolia stack, that are not part of the stack about
// to be started. Two stacks would fight over host ports and the shared
// network, and silently replacing a running node is not something `start`
// should do, so the caller refuses instead.
func conflictingContainers(running []string, executionClient, consensusClient string) []string {
	var members []string
	for _, network := range []string{"ethereum", "ethereum-testnet"} {
		members = append(members, utils.CandidateContainerNames(network)...)
		for _, component := range utils.ComponentNetworks(network) {
			name, _ := utils.GetDefaultLocalMappedContainerName(component)
			members = append(members, name)
		}
	}

	isMember := make(map[string]bool, len(members))
	for _, name := range members {
		isMember[name] = true
	}

	var conflicts []string
	for _, name := range running {
		if isMember[name] && name != executionClient && name != consensusClient {
			conflicts = append(conflicts, name)
		}
	}
	return conflicts
}

// testnetConsensusWarning returns a warning for consensus clients known not to
// work on Sepolia, or "" if there is none. Lodestar 1.48.0 found no peers in
// four 15-minute runs on Linux (including with --nat), while the same
// command finds them on mainnet and the other four clients find them on
// Sepolia; the cause was not found.
func testnetConsensusWarning(consensusClient string) string {
	if consensusClient == "lodestar" {
		return "WARNING: Lodestar found no peers on the Sepolia testnet in testing, so this node is unlikely to sync. Use another --consensus-client (lighthouse, prysm, teku or nimbus) for Sepolia."
	}
	return ""
}

// blobServingNotice returns a heads-up about a client/mode combination that did
// not work in testing, or "" if there is none. Nimbus in semi mode served no
// blobs in a 30-minute Linux run on Sepolia, but it had only 0-2 peers, so it
// is unresolved whether Nimbus cannot serve them or simply lacked peers.
func blobServingNotice(consensusClient, mode string) string {
	if consensusClient == "nimbus" && mode == "semi" {
		return "NOTE: Nimbus in semi mode did not serve blobs in testing (it had very few peers, so it is unresolved why). For blob serving, prefer lighthouse or prysm, or teku with --blob-serving=full."
	}
	return ""
}

// blobServingWarning describes what --blob-serving costs; "" when it is off.
// The mainnet figure is ethPandaOps' estimate for a semi-supernode after
// Fusaka (roughly half of a full supernode's 50-100 Mb/s sustained), not
// something measured here.
func blobServingWarning(mode, network string) string {
	if mode == "" {
		return ""
	}
	if network == "ethereum-testnet" {
		return "Blob serving is on: the consensus client keeps extra data columns so L2 nodes can read blobs from it."
	}
	cost := "roughly 8-16 TB/month of bandwidth (an estimate)"
	if mode == "full" {
		cost = "roughly twice that of a semi-supernode: over 16 TB/month of bandwidth (an estimate)"
	}
	return "WARNING: blob serving is on: on mainnet this consensus client will use " + cost + ", plus extra disk. Make sure your plan allows it."
}

// stopCommands names what stops the given conflicting containers: the mainnet
// stack and the Sepolia stack are stopped separately.
func stopCommands(conflicts []string) string {
	var mainnet, testnet bool
	for _, name := range conflicts {
		if strings.HasSuffix(name, "-testnet") {
			testnet = true
		} else {
			mainnet = true
		}
	}

	exe := utils.GetNodevinExecutable()
	var commands []string
	if mainnet {
		commands = append(commands, fmt.Sprintf("`%s stop ethereum`", exe))
	}
	if testnet {
		commands = append(commands, fmt.Sprintf("`%s stop ethereum --testnet`", exe))
	}
	return strings.Join(commands, " and ")
}

func CreateEthereumComposeFile(cwd string) (string, error) {
	network := "ethereum"
	consensusSuffix := ""
	if utils.CheckIfTestnetOrTestnetNetworkFlag() {
		network = "ethereum-testnet"
		consensusSuffix = "-testnet"
	}

	ethereumBaseComposeConfig, err := compose.GetEthereumNetworkComposeConfig(network)
	if err != nil {
		return "", err
	}

	consensusClient, err := compose.SelectedConsensusClient()
	if err != nil {
		return "", err
	}

	checkpointURL, err := compose.ResolveCheckpointSyncURL(consensusClient)
	if err != nil {
		return "", err
	}

	blobServing, err := compose.ResolveBlobServing(consensusClient)
	if err != nil {
		return "", err
	}
	if warning := blobServingWarning(blobServing, network); warning != "" {
		logger.LogInfo(warning)
	}
	if notice := blobServingNotice(consensusClient, blobServing); notice != "" {
		logger.LogInfo(notice)
	}

	if consensusSuffix != "" {
		if warning := testnetConsensusWarning(consensusClient); warning != "" {
			logger.LogInfo(warning)
		}
	}

	executionClient := ethereumBaseComposeConfig.ContainerName
	paired := consensusClient + consensusSuffix
	if consensusClient == "none" {
		paired = ""
	}
	if running, err := utils.RunningContainerNames(); err == nil {
		if conflicts := conflictingContainers(running, executionClient, paired); len(conflicts) > 0 {
			return "", fmt.Errorf("cannot start %s: %s already running as part of an ethereum stack. Run %s first",
				executionClient, strings.Join(conflicts, ", "), stopCommands(conflicts))
		}
	}

	if consensusClient == "none" {
		return compose.CreateComposeFile(
			ethereumBaseComposeConfig.ContainerName,
			ethereumBaseComposeConfig,
			[]string{},
			[]compose.NetworkConfig{},
			cwd)
	}

	// Pull the consensus client image (same pattern as bitcoin.go's --ord
	// wiring - the primary service's image is pulled implicitly by
	// `docker-compose up -d` later, but the extra service is pulled
	// explicitly up front so any warnings/errors surface before that point).
	consensusImage := viper.GetString("consensus-image")
	consensusVersion := viper.GetString("consensus-version")
	if consensusImage == "" {
		consensusImage = fmt.Sprintf("fiftysix/%s", consensusClient)
	}
	if consensusVersion == "" {
		consensusVersion = "latest"
	}
	if err := docker.PullImage(fmt.Sprintf("%s:%s", consensusImage, consensusVersion)); err != nil {
		logger.LogError("Failed to pull Docker image: " + err.Error())
		return "", err
	}

	consensusComposeConfig, err := consensusClientComposeConfig(consensusClient, consensusClient+consensusSuffix)
	if err != nil {
		return "", err
	}

	// --consensus-image/--consensus-version are a single pair of flags
	// (unlike --ord-image/--ord-version) since only one consensus client is
	// ever selected at a time - applied here, in the wiring layer, rather
	// than duplicated in each of the five per-client builders.
	consensusComposeConfig.Image = consensusImage
	consensusComposeConfig.Version = consensusVersion

	// Before the checkpoint wrapper: Nimbus's wrapper embeds this command.
	consensusComposeConfig.Command, err = compose.WithBlobServing(consensusClient, blobServing, consensusComposeConfig.Command)
	if err != nil {
		return "", err
	}

	consensusComposeConfig.Command, err = compose.WithCheckpointSync(consensusClient, compose.EthereumChain(network), checkpointURL, consensusComposeConfig.Command)
	if err != nil {
		return "", err
	}

	composeFilePath, err := compose.CreateComposeFile(
		ethereumBaseComposeConfig.ContainerName,
		ethereumBaseComposeConfig,
		[]string{consensusClient + consensusSuffix},
		[]compose.NetworkConfig{consensusComposeConfig},
		cwd)

	if err != nil {
		return "", err
	}

	return composeFilePath, nil
}
