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

// GetLighthouseNetworkComposeConfig builds the compose config for Lighthouse
// paired with whichever execution client --execution-client selects (reth by
// default). Lighthouse's own entrypoint has_flag-defaults everything else
// (network, ports) - the only things it cannot default correctly are the
// paired execution client's Engine API endpoint and shared JWT secret, since
// those depend entirely on which execution client this node is paired with.
// See node-images' docs/ethereum-execution-consensus-pairing.md.
func GetLighthouseNetworkComposeConfig(network string) (NetworkConfig, error) {
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

	localPath := filepath.Join(nodevinDataDir, "lighthouse")
	localChainDataPath := filepath.Join(localPath, "lighthouse")

	return NetworkConfig{
		Image:         "fiftysix/lighthouse",
		Version:       "latest",
		ContainerName: "lighthouse",
		Command: fmt.Sprintf(
			"bn --execution-endpoint %s --execution-jwt %s",
			executionEngineEndpoint(executionClient),
			executionJWTPath(executionClient),
		),
		// The Beacon (REST) API is an admin-level surface, published
		// loopback-only (same reasoning as every other chain's RPC port in
		// this repo). P2P stays public.
		Ports: []string{"127.0.0.1:5052:5052", "9000:9000", "9000:9000/udp"},
		Volumes: []string{
			fmt.Sprintf("%s:/node/lighthouse", localChainDataPath),
			execMountVolume,
		},
		Networks: []string{"ethereum-net"},
		NetworkDefs: map[string]NetworkDetails{
			"ethereum-net": {
				Driver: "bridge",
			},
		},
		VolumeDefs: map[string]VolumeDetails{
			"lighthouse-data": {
				Labels: map[string]string{
					"nodevin.blockchain.software": "lighthouse",
				},
			},
		},
		LocalPath: localPath,
	}, nil
}
