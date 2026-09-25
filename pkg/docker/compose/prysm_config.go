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

// GetPrysmNetworkComposeConfig builds the compose config for Prysm paired
// with whichever execution client --execution-client selects. See
// GetLighthouseNetworkComposeConfig and node-images'
// docs/ethereum-execution-consensus-pairing.md for the shared reasoning.
func GetPrysmNetworkComposeConfig(network string) (NetworkConfig, error) {
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

	localPath := filepath.Join(nodevinDataDir, "prysm")
	localChainDataPath := filepath.Join(localPath, "prysm")

	return NetworkConfig{
		Image:         "fiftysix/prysm",
		Version:       "latest",
		ContainerName: "prysm",
		Command: fmt.Sprintf(
			"beacon-chain --execution-endpoint %s --jwt-secret %s",
			executionEngineEndpoint(executionClient),
			executionJWTPath(executionClient),
		),
		Ports: []string{"127.0.0.1:3500:3500", "13000:13000", "12000:12000/udp"},
		Volumes: []string{
			fmt.Sprintf("%s:/node/prysm", localChainDataPath),
			execMountVolume,
		},
		Networks: []string{"ethereum-net"},
		NetworkDefs: map[string]NetworkDetails{
			"ethereum-net": {
				Driver: "bridge",
			},
		},
		VolumeDefs: map[string]VolumeDetails{
			"prysm-data": {
				Labels: map[string]string{
					"nodevin.blockchain.software": "prysm",
				},
			},
		},
		LocalPath: localPath,
	}, nil
}
