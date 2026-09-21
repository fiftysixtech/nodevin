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
	"github.com/spf13/viper"
)

func GetBitcoinNetworkComposeConfig(network string) (NetworkConfig, error) {
	// Get base nodevin data directory
	nodevinDataDir, err := utils.GetNodevinDataDir()
	if err != nil {
		return NetworkConfig{}, err
	}

	// Define the base configuration for the Bitcoin network
	baseConfig := NetworkConfig{
		Image:   "fiftysix/bitcoin-core",
		Version: "latest",
		// The JSON-RPC port is published on the host's loopback interface only:
		// it is an admin-level API (and, for the Bitcoin family, protected by a
		// shared password over plain HTTP). Peer ports stay public so other
		// nodes can connect. Override with --ports.
		Ports:    []string{"127.0.0.1:8332:8332", "8333:8333"},
		Volumes:  []string{},
		Networks: []string{"bitcoin-net"},
		NetworkDefs: map[string]NetworkDetails{
			"bitcoin-net": {
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

	// Set the container name and command based on the network
	switch network {
	case "bitcoin":
		localPath := filepath.Join(nodevinDataDir, "bitcoin-core")     // nodevin data dir, software type
		localChainDataPath := filepath.Join(localPath, "bitcoin-core") // on-image data dir
		baseConfig.ContainerName = "bitcoin-core"
		baseConfig.Command = "bitcoind --server=1 --rpcbind=0.0.0.0 --rpcport=8332 --rpcallowip=0.0.0.0/0"
		baseConfig.Volumes = []string{fmt.Sprintf("%s:/node/bitcoin-core", localChainDataPath)}
		baseConfig.VolumeDefs = map[string]VolumeDetails{
			"bitcoin-core-data": {
				Labels: map[string]string{
					"nodevin.blockchain.software": "bitcoin-core",
				},
			},
		}
		baseConfig.LocalPath = localPath
		baseConfig.SnapshotSyncCID = networkCID
		baseConfig.SnapshotDataFilename = "bitcoin-mainnet-chain-data.tar.gz"
		baseConfig.LocalChainDataPath = "/nodevin-volume/bitcoin-core/data"

	case "bitcoin-testnet":
		localPath := filepath.Join(nodevinDataDir, "bitcoin-core-testnet") // nodevin data dir, software type
		localChainDataPath := filepath.Join(localPath, "bitcoin-core")     // on-image data dir
		baseConfig.ContainerName = "bitcoin-core-testnet"
		baseConfig.Command = "bitcoind --testnet --server=1 --rpcbind=0.0.0.0 --rpcport=18332 --rpcallowip=0.0.0.0/0"
		baseConfig.Networks = []string{"bitcoin-testnet-net"}
		baseConfig.NetworkDefs = map[string]NetworkDetails{
			"bitcoin-testnet-net": {
				Driver: "bridge",
			},
		}
		baseConfig.Ports = []string{"127.0.0.1:18332:18332", "18333:18333"}
		baseConfig.Volumes = []string{fmt.Sprintf("%s:/node/bitcoin-core", localChainDataPath)}
		baseConfig.VolumeDefs = map[string]VolumeDetails{
			"bitcoin-core-testnet-data": {
				Labels: map[string]string{
					"nodevin.blockchain.software": "bitcoin-core",
				},
			},
		}
		baseConfig.LocalPath = localPath
		baseConfig.SnapshotSyncCID = networkCID
		baseConfig.SnapshotDataFilename = "bitcoin-testnet-chain-data.tar.gz"
		baseConfig.LocalChainDataPath = "/nodevin-volume/bitcoin-core/data/testnet3"

	default:
		return NetworkConfig{}, fmt.Errorf("unknown network: %s", network)
	}

	// Optionally add RPC authentication to the command
	cookieAuth := viper.GetBool("cookie-auth")
	if !cookieAuth {
		rpcUsername := viper.GetString("rpc-user")
		rpcPassword := viper.GetString("rpc-pass")

		if rpcUsername == "" {
			rpcUsername = "user"
		}

		if rpcPassword == "" {
			rpcPassword = "fiftysix"
		}

		baseConfig.Command = fmt.Sprintf("%s -rpcuser=%s -rpcpassword=%s", baseConfig.Command, rpcUsername, rpcPassword)
	}

	return baseConfig, nil
}
