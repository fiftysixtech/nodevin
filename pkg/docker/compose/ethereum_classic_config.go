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

func GetEthereumClassicNetworkComposeConfig(network string) (NetworkConfig, error) {
	// Get base nodevin data directory
	nodevinDataDir, err := utils.GetNodevinDataDir()
	if err != nil {
		return NetworkConfig{}, err
	}

	// Define the base configuration for the Ethereum Classic network
	baseConfig := NetworkConfig{
		Image:    "fiftysix/core-geth",
		Version:  "latest",
		Ports:    []string{"8545:8545", "30303:30303", "30303:30303/udp"},
		Volumes:  []string{},
		Networks: []string{"ethereum-classic-net"},
		NetworkDefs: map[string]NetworkDetails{
			"ethereum-classic-net": {
				Driver: "bridge",
			},
		},
		VolumeDefs: map[string]VolumeDetails{},
	}

	networkCID, exists := utils.GetSnapshotCIDByNetwork(network)
	if !exists {
		networkCID = ""
		fmt.Printf("Unable to find CID for network. Skipping...")
	}

	// core-geth (like upstream go-ethereum) has no built-in HTTP basic-auth
	// flags (no -rpcuser/-rpcpassword, unlike the Bitcoin-derived daemons),
	// so --rpc-user/--rpc-pass/--cookie-auth are not applicable here. Instead,
	// the JSON-RPC HTTP server is restricted with --http.vhosts/--http.corsdomain.

	// Set the container name and command based on the network
	switch network {
	case "ethereum-classic":
		localPath := filepath.Join(nodevinDataDir, "core-geth")     // nodevin data dir, software type
		localChainDataPath := filepath.Join(localPath, "core-geth") // on-image data dir
		baseConfig.ContainerName = "core-geth"
		baseConfig.Command = "geth --classic --http --http.addr 0.0.0.0 --http.port 8545 --http.api eth,net,web3,txpool --http.vhosts * --http.corsdomain * --port 30303"
		baseConfig.Volumes = []string{fmt.Sprintf("%s:/node/core-geth", localChainDataPath)}
		baseConfig.VolumeDefs = map[string]VolumeDetails{
			"core-geth-data": {
				Labels: map[string]string{
					"nodevin.blockchain.software": "core-geth",
				},
			},
		}
		baseConfig.LocalPath = localPath
		baseConfig.SnapshotSyncCID = networkCID
		baseConfig.SnapshotDataFilename = "ethereum-classic-mainnet-chain-data.tar.gz"
		baseConfig.LocalChainDataPath = "/nodevin-volume/core-geth/data"

	case "ethereum-classic-testnet":
		localPath := filepath.Join(nodevinDataDir, "core-geth-testnet") // nodevin data dir, software type
		localChainDataPath := filepath.Join(localPath, "core-geth")     // on-image data dir
		baseConfig.ContainerName = "core-geth-testnet"
		baseConfig.Command = "geth --mordor --http --http.addr 0.0.0.0 --http.port 8546 --http.api eth,net,web3,txpool --http.vhosts * --http.corsdomain * --port 30304"
		baseConfig.Networks = []string{"core-geth-testnet-net"}
		baseConfig.NetworkDefs = map[string]NetworkDetails{
			"core-geth-testnet-net": {
				Driver: "bridge",
			},
		}
		baseConfig.Ports = []string{"8546:8546", "30304:30304", "30304:30304/udp"}
		baseConfig.Volumes = []string{fmt.Sprintf("%s:/node/core-geth", localChainDataPath)}
		baseConfig.VolumeDefs = map[string]VolumeDetails{
			"core-geth-testnet-data": {
				Labels: map[string]string{
					"nodevin.blockchain.software": "core-geth",
				},
			},
		}
		baseConfig.LocalPath = localPath
		baseConfig.SnapshotSyncCID = networkCID
		baseConfig.SnapshotDataFilename = "ethereum-classic-testnet-chain-data.tar.gz"
		baseConfig.LocalChainDataPath = "/nodevin-volume/core-geth/data/mordor"

	default:
		return NetworkConfig{}, fmt.Errorf("unknown network: %s", network)
	}

	return baseConfig, nil
}
