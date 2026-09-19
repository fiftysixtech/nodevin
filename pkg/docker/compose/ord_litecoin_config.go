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

func GetOrdLitecoinNetworkComposeConfig(network string) (NetworkConfig, error) {
	// Get base nodevin data directory
	nodevinDataDir, err := utils.GetNodevinDataDir()
	if err != nil {
		return NetworkConfig{}, err
	}

	// Define the base configuration for ord-litecoin
	baseConfig := NetworkConfig{
		Image:    "fiftysix/ord-litecoin",
		Version:  "latest",
		Restart:  "always",
		Ports:    []string{"80:80"},
		Volumes:  []string{},
		Networks: []string{"litecoin-net"},
		NetworkDefs: map[string]NetworkDetails{
			"litecoin-net": {
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
	case "ord-litecoin":
		localPath := filepath.Join(nodevinDataDir, "ord-litecoin")     // nodevin data dir, software type
		localChainDataPath := filepath.Join(localPath, "ord-litecoin") // on-image data dir
		baseConfig.ContainerName = "ord-litecoin"
		baseConfig.Ports = []string{"8082:80"}
		baseConfig.Command = "ord --litecoin-rpc-url http://litecoin-core:9332"
		baseConfig.Volumes = []string{
			fmt.Sprintf("%s:/node/litecoin-core", filepath.Join(nodevinDataDir, "litecoin-core", "litecoin-core")),
			fmt.Sprintf("%s:/node/ord-litecoin", localChainDataPath),
		}
		baseConfig.VolumeDefs = map[string]VolumeDetails{
			"ord-litecoin-data": {
				Labels: map[string]string{
					"nodevin.blockchain.software": "ord-litecoin",
				},
			},
		}
		baseConfig.LocalPath = localPath
		baseConfig.SnapshotSyncCID = networkCID
		baseConfig.SnapshotDataFilename = "ord-litecoin-mainnet-chain-data.tar.gz"
		baseConfig.LocalChainDataPath = "/nodevin-volume-ord-litecoin/ord-litecoin/data"

	case "ord-litecoin-testnet":
		localPath := filepath.Join(nodevinDataDir, "ord-litecoin-testnet") // nodevin data dir, software type
		localChainDataPath := filepath.Join(localPath, "ord-litecoin")     // on-image data dir
		baseConfig.ContainerName = "ord-litecoin-testnet"
		baseConfig.Command = "ord --testnet --litecoin-rpc-url http://litecoin-core-testnet:19332"
		baseConfig.Volumes = []string{
			fmt.Sprintf("%s:/node/litecoin-core", filepath.Join(nodevinDataDir, "litecoin-core-testnet", "litecoin-core")),
			fmt.Sprintf("%s:/node/ord-litecoin", localChainDataPath),
		}
		baseConfig.Networks = []string{"litecoin-testnet-net"}
		baseConfig.Ports = []string{"8083:80"}
		baseConfig.NetworkDefs = map[string]NetworkDetails{
			"litecoin-testnet-net": {
				Driver: "bridge",
			},
		}
		baseConfig.VolumeDefs = map[string]VolumeDetails{
			"ord-litecoin-testnet-data": {
				Labels: map[string]string{
					"nodevin.blockchain.software": "ord-litecoin",
				},
			},
		}
		baseConfig.LocalPath = localPath
		baseConfig.SnapshotSyncCID = networkCID
		baseConfig.SnapshotDataFilename = "ord-litecoin-testnet-chain-data.tar.gz"
		baseConfig.LocalChainDataPath = "/nodevin-volume-ord-litecoin/ord-litecoin/data/testnet3" // forked from ord, so doesn't move to testnet4

	default:
		return NetworkConfig{}, fmt.Errorf("unknown network: %s", network)
	}

	// Optionally add RPC authentication to the command
	cookieAuth := viper.GetBool("ord-litecoin-cookie-auth") || viper.GetBool("ord-cookie-auth")
	if !cookieAuth {
		rpcUsername := viper.GetString("ord-litecoin-rpc-user")
		rpcPassword := viper.GetString("ord-litecoin-rpc-pass")

		if rpcUsername == "" {
			rpcUsername = viper.GetString("ord-rpc-user")
			if rpcUsername == "" {
				rpcUsername = "user"
			}
		}

		if rpcPassword == "" {
			rpcPassword = viper.GetString("ord-rpc-pass")
			if rpcPassword == "" {
				rpcPassword = "fiftysix"
			}
		}

		baseConfig.Command = fmt.Sprintf("%s --litecoin-rpc-username %s --litecoin-rpc-password %s", baseConfig.Command, rpcUsername, rpcPassword)
	}

	baseConfig.Command = baseConfig.Command + " server"

	return baseConfig, nil
}
