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
	"path/filepath"

	"github.com/fiftysixcrypto/nodevin/internal/utils"
)

// arbitrumChainIDs are Nitro's --chain.id and the parent (L1) chain's
// --parent-chain.id it must be paired with. Confirmed against
// fiftysix/nitro: --chain.id alone resolves Arbitrum's built-in chain
// config, no genesis file needed.
type arbitrumChainIDs struct {
	chainID       string
	parentChainID string
}

func arbitrumChain(network string) arbitrumChainIDs {
	if network == "arbitrum-testnet" {
		return arbitrumChainIDs{chainID: "421614", parentChainID: "11155111"} // Arbitrum Sepolia / Ethereum Sepolia
	}
	return arbitrumChainIDs{chainID: "42161", parentChainID: "1"} // Arbitrum One / Ethereum mainnet
}

// ArbitrumChainIDs exports arbitrumChain's result for the snapshot-init flow
// (pkg/nodes/arbitrum), which needs the same --chain.id/--parent-chain.id
// Nitro's normal run command uses - confirmed empirically that Nitro reads
// the parent chain during init, before any snapshot download begins, so
// phase 1 cannot skip these.
type ArbitrumChainIDs struct {
	ChainID       string
	ParentChainID string
}

func ResolveArbitrumChainIDs(network string) ArbitrumChainIDs {
	ids := arbitrumChain(network)
	return ArbitrumChainIDs{ChainID: ids.chainID, ParentChainID: ids.parentChainID}
}

// ArbitrumSnapshotChainName returns the on-disk/snapshot chain name Nitro and
// snapshot.arbitrum.foundation both use - confirmed by reading a real running
// node's own log line (database=.../sepolia-rollup/nitro/l2chaindata) and
// cross-checked against the real metadata.json of both networks' published
// snapshots ("chain_name": "arb1" / "sepolia-rollup"). This is NOT the same
// string as --chain.id (a number) or the registry network key.
func ArbitrumSnapshotChainName(network string) string {
	if network == "arbitrum-testnet" {
		return "sepolia-rollup"
	}
	return "arb1"
}

// ArbitrumLocalChainDataPath returns the host directory mounted at
// /node/nitro/data for network - the single source of truth both
// GetArbitrumNetworkComposeConfig (phase 2, the normal run) and the
// snapshot-init flow (phase 1, pkg/nodes/arbitrum) use, so the two always
// agree on exactly where Nitro's data lives.
func ArbitrumLocalChainDataPath(network string) (string, error) {
	nodevinDataDir, err := utils.GetNodevinDataDir()
	if err != nil {
		return "", err
	}
	containerName, ok := utils.GetDefaultLocalMappedContainerName(network)
	if !ok {
		return "", fmt.Errorf("unknown network: %s", network)
	}
	return filepath.Join(nodevinDataDir, containerName, "nitro"), nil
}

// GetArbitrumNetworkComposeConfig builds the compose config for fiftysix/nitro
// on "arbitrum" (mainnet) or "arbitrum-testnet" (Arbitrum Sepolia). Unlike
// every Ethereum client, Nitro's entrypoint execs the binary directly with
// the container's command as its arguments (no "nitro" prefix, unlike
// geth/besu/etc's "geth ..." convention) - confirmed by reading
// node-images' entrypoint.sh for it.
func GetArbitrumNetworkComposeConfig(network string, l1 L1Endpoints) (NetworkConfig, error) {
	nodevinDataDir, err := utils.GetNodevinDataDir()
	if err != nil {
		return NetworkConfig{}, err
	}

	containerName, ok := utils.GetDefaultLocalMappedContainerName(network)
	if !ok {
		return NetworkConfig{}, fmt.Errorf("unknown network: %s", network)
	}

	localPath := filepath.Join(nodevinDataDir, containerName)
	localChainDataPath, err := ArbitrumLocalChainDataPath(network)
	if err != nil {
		return NetworkConfig{}, err
	}

	chain := arbitrumChain(network)
	command := fmt.Sprintf(
		"--chain.id %s --parent-chain.id %s --parent-chain.connection.url %s --parent-chain.blob-client.beacon-url %s",
		chain.chainID, chain.parentChainID, l1.ExecutionRPCURL, l1.BeaconURL,
	)

	// Loopback-only, admin-level JSON-RPC/WS, same reasoning as every other
	// chain in this repo. Nitro has no P2P port of its own (it reads execution
	// data from the L1, not from a P2P network) - 9642 (the sequencer feed) is
	// published on all interfaces instead, since it is what other nodes read
	// from this one, functionally taking P2P's place for the loopback-only
	// exemption TestRPCPortsArePublishedOnLoopbackOnly requires.
	sequencerFeedPort := 9642
	if network == "arbitrum-testnet" {
		sequencerFeedPort = 9643
	}
	ports := []string{
		fmt.Sprintf("127.0.0.1:%d:8547", utils.NetworkDefaultRPCPorts()[network]),
		fmt.Sprintf("127.0.0.1:%d:8548", utils.NetworkDefaultRPCPorts()[network]+1),
		fmt.Sprintf("%d:9642", sequencerFeedPort),
	}

	dockerNetworks := []string{fmt.Sprintf("%s-net", containerName)}
	networkDefs := map[string]NetworkDetails{
		fmt.Sprintf("%s-net", containerName): {Driver: "bridge"},
	}
	// Auto-attached to nodevin's own Ethereum L1: join its Docker network too,
	// so "http://<execution client>:8545"/"...:beacon-port" resolve by
	// container name (see ResolveL1Endpoints and EthereumDockerNetworkName).
	if l1.DockerNetwork != "" {
		dockerNetworks = append(dockerNetworks, l1.DockerNetwork)
		networkDefs[l1.DockerNetwork] = NetworkDetails{Driver: "bridge"}
	}

	return NetworkConfig{
		Image:         "fiftysix/nitro",
		Version:       "latest",
		ContainerName: containerName,
		Command:       command,
		// Nitro's WORKDIR is /home/user, which holds large prover/machine
		// files the entrypoint needs in place, not a default config to seed
		// a fresh data directory with - and copying them fails outright
		// anyway, since Nitro runs as a non-root user that cannot set
		// permissions on the copies (confirmed empirically).
		SkipInitCopy: true,
		Ports:        ports,
		// Mounted at /node/nitro/data specifically, not the whole /node/nitro
		// (unlike geth/reth/etc's own broad /node/<client> mount) - Nitro's
		// own entrypoint script lives at /node/nitro/scripts inside the
		// image, and a broader mount would shadow it with an empty host
		// directory (confirmed: the container then fails to even start,
		// "no such file or directory" for the entrypoint). Nothing under
		// /node/nitro/data needs seeding from the image either - Nitro
		// creates its own chain database files there at runtime - so this
		// also makes SkipInitCopy above correct rather than just convenient.
		Volumes:     []string{fmt.Sprintf("%s:/node/nitro/data", localChainDataPath)},
		Networks:    dockerNetworks,
		NetworkDefs: networkDefs,
		VolumeDefs: map[string]VolumeDetails{
			fmt.Sprintf("%s-data", containerName): {
				Labels: map[string]string{
					"nodevin.blockchain.software": containerName,
				},
			},
		},
		LocalPath: localPath,
	}, nil
}
