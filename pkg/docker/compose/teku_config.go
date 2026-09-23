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

// GetTekuNetworkComposeConfig builds the compose config for Teku paired with
// whichever execution client --execution-client selects. See
// GetLighthouseNetworkComposeConfig and node-images'
// docs/ethereum-execution-consensus-pairing.md for the shared reasoning.
//
// Teku's own CLI has no subcommand (unlike lighthouse's "bn" or prysm's
// "beacon-chain") - the client name alone is the beacon node command.
func GetTekuNetworkComposeConfig(network string) (NetworkConfig, error) {
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

	localPath := filepath.Join(nodevinDataDir, "teku")
	localChainDataPath := filepath.Join(localPath, "teku")

	return NetworkConfig{
		Image:         "fiftysix/teku",
		Version:       "latest",
		ContainerName: "teku",
		Command: fmt.Sprintf(
			"teku --ee-endpoint %s --ee-jwt-secret-file %s",
			executionEngineEndpoint(executionClient),
			executionJWTPath(executionClient),
		),
		Ports: []string{"127.0.0.1:5051:5051", "9000:9000", "9000:9000/udp"},
		Volumes: []string{
			fmt.Sprintf("%s:/node/teku", localChainDataPath),
			execMountVolume,
		},
		Networks: []string{"ethereum-net"},
		NetworkDefs: map[string]NetworkDetails{
			"ethereum-net": {
				Driver: "bridge",
			},
		},
		VolumeDefs: map[string]VolumeDetails{
			"teku-data": {
				Labels: map[string]string{
					"nodevin.blockchain.software": "teku",
				},
			},
		},
		LocalPath: localPath,
	}, nil
}
