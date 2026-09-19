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

func GetKuboNetworkComposeConfig(network string) (NetworkConfig, error) {
	// Get base nodevin data directory
	nodevinDataDir, err := utils.GetNodevinDataDir()
	if err != nil {
		return NetworkConfig{}, err
	}

	// Define the base configuration for Kubo (IPFS)
	baseConfig := NetworkConfig{
		Image:    "fiftysix/kubo",
		Version:  "latest",
		Ports:    []string{"4001:4001", "5001:5001", "8080:8080"},
		Volumes:  []string{},
		Networks: []string{"ipfs-net"},
		NetworkDefs: map[string]NetworkDetails{
			"ipfs-net": {
				Driver: "bridge",
			},
		},
		VolumeDefs: map[string]VolumeDetails{},
	}

	// Set the container name and command based on the network
	switch network {
	case "ipfs":
		localPath := filepath.Join(nodevinDataDir, "ipfs")     // nodevin data dir, software type
		localChainDataPath := filepath.Join(localPath, "ipfs") // on-image data dir
		baseConfig.ContainerName = "ipfs"
		baseConfig.Command = ""
		baseConfig.CommandIsIntentionallyEmpty = true
		baseConfig.Volumes = []string{fmt.Sprintf("%s:/node/ipfs", localChainDataPath)}
		baseConfig.VolumeDefs = map[string]VolumeDetails{
			"ipfs-data": {
				Labels: map[string]string{
					"nodevin.blockchain.software": "kubo",
				},
			},
		}
		baseConfig.LocalPath = localPath
		baseConfig.SnapshotSyncCID = ""
		baseConfig.SnapshotDataFilename = ""
		baseConfig.LocalChainDataPath = "/nodevin-volume/ipfs/data"

	default:
		return NetworkConfig{}, fmt.Errorf("unknown network: %s", network)
	}

	return baseConfig, nil
}
