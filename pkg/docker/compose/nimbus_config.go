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

// GetNimbusNetworkComposeConfig builds the compose config for Nimbus paired
// with whichever execution client --execution-client selects. See
// GetLighthouseNetworkComposeConfig and node-images'
// docs/ethereum-execution-consensus-pairing.md for the shared reasoning.
//
// Nimbus's (Nim/confutils) CLI only accepts --flag=value - a space-separated
// "--flag value" is silently misparsed as two separate tokens. The "="
// below is required, not stylistic.
func GetNimbusNetworkComposeConfig(network string) (NetworkConfig, error) {
	executionClient, err := SelectedExecutionClient()
	if err != nil {
		return NetworkConfig{}, err
	}

	execMountVolume, err := executionClientMountVolume(executionClient)
	if err != nil {
		return NetworkConfig{}, err
	}

	nodevinDataDir, err := utils.GetNodevinDataDir()
	if err != nil {
		return NetworkConfig{}, err
	}

	localPath := filepath.Join(nodevinDataDir, "nimbus")
	localChainDataPath := filepath.Join(localPath, "nimbus")

	return NetworkConfig{
		Image:         "fiftysix/nimbus",
		Version:       "latest",
		ContainerName: "nimbus",
		Command: fmt.Sprintf(
			"nimbus_beacon_node --el=%s --jwt-secret=%s",
			executionEngineEndpoint(executionClient),
			executionJWTPath(executionClient),
		),
		Ports: []string{"127.0.0.1:5052:5052", "9000:9000", "9000:9000/udp", "9001:9001/udp"},
		Volumes: []string{
			fmt.Sprintf("%s:/node/nimbus", localChainDataPath),
			execMountVolume,
		},
		Networks: []string{"ethereum-net"},
		NetworkDefs: map[string]NetworkDetails{
			"ethereum-net": {
				Driver: "bridge",
			},
		},
		VolumeDefs: map[string]VolumeDetails{
			"nimbus-data": {
				Labels: map[string]string{
					"nodevin.blockchain.software": "nimbus",
				},
			},
		},
		LocalPath: localPath,
	}, nil
}
