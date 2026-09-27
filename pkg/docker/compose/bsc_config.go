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

// bscPorts returns fiftysix/bsc's port mappings: loopback-only HTTP/WS-RPC,
// plus P2P (tcp+udp) published publicly. Mainnet and testnet get different
// P2P host ports so both can run at once, matching this repo's Arbitrum/Base
// convention rather than Ethereum's mutual-exclusion one.
func bscPorts(suffix string, rpcPort int) []string {
	p2pPort := 30311
	if suffix != "" {
		p2pPort = 30312
	}
	return []string{
		fmt.Sprintf("127.0.0.1:%d:8545", rpcPort),
		fmt.Sprintf("127.0.0.1:%d:8546", rpcPort+1),
		fmt.Sprintf("%d:30311", p2pPort),
		fmt.Sprintf("%d:30311/udp", p2pPort),
	}
}

// GetBscNetworkComposeConfig builds the compose config for fiftysix/bsc on
// "bsc" (mainnet) or "bsc-testnet" (Chapel). Unlike geth/core-geth, network
// selection is the NETWORK environment variable, not a CLI flag - BSC has no
// built-in chain config nodevin can select with a plain flag, so this image
// bundles both networks' genesis/config and its entrypoint picks between
// them (see node-images' docs/bnb-smart-chain.md).
func GetBscNetworkComposeConfig(network string) (NetworkConfig, error) {
	nodevinDataDir, err := utils.GetNodevinDataDir()
	if err != nil {
		return NetworkConfig{}, err
	}

	containerName, ok := utils.GetDefaultLocalMappedContainerName(network)
	if !ok {
		return NetworkConfig{}, fmt.Errorf("unknown network: %s", network)
	}

	suffix := stackSuffix(network)
	envNetwork := "mainnet"
	if suffix != "" {
		envNetwork = "testnet"
	}

	localPath := filepath.Join(nodevinDataDir, containerName)

	return NetworkConfig{
		Image:                       "fiftysix/bsc",
		Version:                     "latest",
		ContainerName:               containerName,
		CommandIsIntentionallyEmpty: true,
		Environment: map[string]string{
			"NETWORK": envNetwork,
		},
		Ports: bscPorts(suffix, utils.NetworkDefaultRPCPorts()[network]),
		// Mounted at /node/bsc/data specifically, not the whole /node/bsc -
		// same reasoning as Nitro's volume mount: the image's own entrypoint
		// script and bundled genesis/config live under /node/bsc/scripts and
		// /node/bsc/configs, and a broader mount would shadow them with an
		// empty host directory. Nothing under /node/bsc/data needs seeding
		// either - bsc creates its own chain database there via `geth init`
		// on first start - so SkipInitCopy is correct, not just convenient.
		Volumes:      []string{fmt.Sprintf("%s:/node/bsc/data", localPath)},
		SkipInitCopy: true,
		Networks:     []string{"bsc" + suffix + "-net"},
		NetworkDefs: map[string]NetworkDetails{
			"bsc" + suffix + "-net": {Driver: "bridge"},
		},
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
