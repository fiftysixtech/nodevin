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

package arbitrum

import (
	"github.com/fiftysixcrypto/nodevin/internal/logger"
	"github.com/fiftysixcrypto/nodevin/internal/utils"
	"github.com/fiftysixcrypto/nodevin/pkg/docker/compose"
)

// CreateArbitrumComposeFile builds the compose file for fiftysix/nitro on
// "arbitrum" (mainnet) or "arbitrum-testnet" (--testnet). See
// compose.ResolveL1Endpoints for how the Ethereum L1 it depends on is found.
func CreateArbitrumComposeFile(cwd string) (string, error) {
	network := "arbitrum"
	testnet := utils.CheckIfTestnetOrTestnetNetworkFlag()
	if testnet {
		network = "arbitrum-testnet"
	}

	l1, err := compose.ResolveL1Endpoints(testnet)
	if err != nil {
		return "", err
	}
	if l1.DockerNetwork != "" {
		logger.LogInfo("No --l1-execution-rpc-url/--l1-beacon-url given: attaching to nodevin's own running Ethereum node as this node's L1.")
	}
	logger.LogInfo("NOTE: a freshly-started Ethereum consensus client only has blobs from around when it started. If your L1 was started recently, or without --blob-serving, this node may not find the blobs its batches need until the L1 has run long enough - see nodevin's --blob-serving documentation.")

	arbitrumComposeConfig, err := compose.GetArbitrumNetworkComposeConfig(network, l1)
	if err != nil {
		return "", err
	}

	composeFilePath, err := compose.CreateComposeFile(
		arbitrumComposeConfig.ContainerName,
		arbitrumComposeConfig,
		[]string{},
		[]compose.NetworkConfig{},
		cwd)

	if err != nil {
		return "", err
	}

	return composeFilePath, nil
}
