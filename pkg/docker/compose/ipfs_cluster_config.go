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

func GetIpfsClusterNetworkComposeConfig(network string) (NetworkConfig, error) {
	// Get base nodevin data directory
	nodevinDataDir, err := utils.GetNodevinDataDir()
	if err != nil {
		return NetworkConfig{}, err
	}

	// Define the base configuration for ipfs-cluster (used alongside IPFS)
	baseConfig := NetworkConfig{
		Image:    "fiftysix/ipfs-cluster",
		Version:  "latest",
		Restart:  "always",
		Ports:    []string{"9094:9094", "9096:9096"},
		Volumes:  []string{},
		Networks: []string{"ipfs-net"},
		NetworkDefs: map[string]NetworkDetails{
			"ipfs-net": {
				Driver: "bridge",
			},
		},
		VolumeDefs: map[string]VolumeDetails{},
	}

	// Conditionally set environment variables if Viper flags are set
	env := make(map[string]string)

	if peername := viper.GetString("ipfs-cluster-peername"); peername != "" {
		env["CLUSTER_PEERNAME"] = peername
	}
	if secret := viper.GetString("ipfs-cluster-secret"); secret != "" {
		env["CLUSTER_SECRET"] = secret
	}
	if bootstrap := viper.GetString("ipfs-cluster-bootstrap"); bootstrap != "" {
		env["CLUSTER_BOOTSTRAP"] = bootstrap
	}

	// Only assign the environment map if it has entries
	if len(env) > 0 {
		baseConfig.Environment = env
	}

	// Set the container name and command based on the network
	switch network {
	case "ipfs-cluster":
		localPath := filepath.Join(nodevinDataDir, "ipfs-cluster")     // nodevin data dir, software type
		localChainDataPath := filepath.Join(localPath, "ipfs-cluster") // on-image data dir
		baseConfig.ContainerName = "ipfs-cluster"
		baseConfig.Command = ""
		baseConfig.CommandIsIntentionallyEmpty = true
		baseConfig.Volumes = []string{
			fmt.Sprintf("%s:/node/ipfs", filepath.Join(nodevinDataDir, "ipfs", "ipfs")),
			fmt.Sprintf("%s:/node/ipfs-cluster", localChainDataPath),
		}
		baseConfig.VolumeDefs = map[string]VolumeDetails{
			"ipfs-cluster-data": {
				Labels: map[string]string{
					"nodevin.blockchain.software": "ipfs-cluster",
				},
			},
		}
		baseConfig.LocalPath = localPath
		baseConfig.SnapshotSyncCID = ""
		baseConfig.SnapshotDataFilename = ""
		baseConfig.LocalChainDataPath = "/nodevin-volume-ipfs-cluster/ipfs-cluster/data"

	default:
		return NetworkConfig{}, fmt.Errorf("unknown network: %s", network)
	}

	return baseConfig, nil
}
