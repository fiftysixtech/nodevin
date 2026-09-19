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

func GetOrdNetworkComposeConfig(network string) (NetworkConfig, error) {
	// Get base nodevin data directory
	nodevinDataDir, err := utils.GetNodevinDataDir()
	if err != nil {
		return NetworkConfig{}, err
	}

	// Define the base configuration for ord
	baseConfig := NetworkConfig{
		Image:    "fiftysix/ord",
		Version:  "latest",
		Restart:  "always",
		Ports:    []string{"80:80"},
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
	case "ord":
		localPath := filepath.Join(nodevinDataDir, "ord")     // nodevin data dir, software type
		localChainDataPath := filepath.Join(localPath, "ord") // on-image data dir
		baseConfig.ContainerName = "ord"
		baseConfig.Command = "ord --bitcoin-rpc-url http://bitcoin-core:8332"
		baseConfig.Volumes = []string{
			fmt.Sprintf("%s:/node/bitcoin-core", filepath.Join(nodevinDataDir, "bitcoin-core", "bitcoin-core")),
			fmt.Sprintf("%s:/node/ord", localChainDataPath),
		}
		baseConfig.VolumeDefs = map[string]VolumeDetails{
			"ord-data": {
				Labels: map[string]string{
					"nodevin.blockchain.software": "ord",
				},
			},
		}
		baseConfig.LocalPath = localPath
		baseConfig.SnapshotSyncCID = networkCID
		baseConfig.SnapshotDataFilename = "ord-bitcoin-mainnet-chain-data.tar.gz"
		baseConfig.LocalChainDataPath = "/nodevin-volume-ord/ord/data"

	case "ord-testnet":
		localPath := filepath.Join(nodevinDataDir, "ord-testnet") // nodevin data dir, software type
		localChainDataPath := filepath.Join(localPath, "ord")     // on-image data dir
		baseConfig.ContainerName = "ord-testnet"
		baseConfig.Command = "ord --testnet --bitcoin-rpc-url http://bitcoin-core-testnet:18332"
		baseConfig.Volumes = []string{
			fmt.Sprintf("%s:/node/bitcoin-core", filepath.Join(nodevinDataDir, "bitcoin-core-testnet", "bitcoin-core")),
			fmt.Sprintf("%s:/node/ord", localChainDataPath),
		}
		baseConfig.Networks = []string{"bitcoin-testnet-net"}
		baseConfig.Ports = []string{"8081:80"}
		baseConfig.NetworkDefs = map[string]NetworkDetails{
			"bitcoin-testnet-net": {
				Driver: "bridge",
			},
		}
		baseConfig.VolumeDefs = map[string]VolumeDetails{
			"ord-testnet-data": {
				Labels: map[string]string{
					"nodevin.blockchain.software": "ord",
				},
			},
		}
		baseConfig.LocalPath = localPath
		baseConfig.SnapshotSyncCID = networkCID
		baseConfig.SnapshotDataFilename = "ord-bitcoin-testnet-chain-data.tar.gz"
		baseConfig.LocalChainDataPath = "/nodevin-volume-ord/bitcoin-core/data/testnet3"

	default:
		return NetworkConfig{}, fmt.Errorf("unknown network: %s", network)
	}

	// Optionally add RPC authentication to the command
	cookieAuth := viper.GetBool("ord-cookie-auth")
	if !cookieAuth {
		rpcUsername := viper.GetString("ord-rpc-user")
		rpcPassword := viper.GetString("ord-rpc-pass")

		if rpcUsername == "" {
			rpcUsername = "user"
		}

		if rpcPassword == "" {
			rpcPassword = "fiftysix"
		}

		baseConfig.Command = fmt.Sprintf("%s --bitcoin-rpc-username %s --bitcoin-rpc-password %s", baseConfig.Command, rpcUsername, rpcPassword)
	}

	baseConfig.Command = baseConfig.Command + " server"

	return baseConfig, nil
}
