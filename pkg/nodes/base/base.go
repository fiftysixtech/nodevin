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

package base

import (
	"fmt"

	"github.com/fiftysixcrypto/nodevin/internal/logger"
	"github.com/fiftysixcrypto/nodevin/internal/utils"
	"github.com/fiftysixcrypto/nodevin/pkg/docker"
	"github.com/fiftysixcrypto/nodevin/pkg/docker/compose"
)

// pullConsensusImage is a variable so tests can substitute it: unlike the
// `docker` CLI, pkg/docker.PullImage talks to the Docker Engine API directly
// and panics without a real daemon connection, which a test binary never has
// (mirrors internal/utils.listRunningContainers' own reason for being a var).
var pullConsensusImage = docker.PullImage

// CreateBaseComposeFile builds the compose file for the fiftysix/base-reth +
// fiftysix/base-consensus pair on "base" (mainnet) or "base-testnet"
// (--testnet). See compose.ResolveL1Endpoints for how the Ethereum L1 both
// depend on is found, and compose.BaseJWTSecret for the shared Engine API
// secret between the two containers.
func CreateBaseComposeFile(cwd string) (string, error) {
	network := "base"
	consensusNetwork := "base-consensus"
	testnet := utils.CheckIfTestnetOrTestnetNetworkFlag()
	if testnet {
		network = "base-testnet"
		consensusNetwork = "base-consensus-testnet"
	}

	l1, err := compose.ResolveL1Endpoints(testnet)
	if err != nil {
		return "", err
	}
	if l1.DockerNetwork != "" {
		logger.LogInfo("No --l1-execution-rpc-url/--l1-beacon-url given: attaching to nodevin's own running Ethereum node as this node's L1.")
	}
	logger.LogInfo("NOTE: a freshly-started Ethereum consensus client only has blobs from around when it started. If your L1 was started recently, or without --blob-serving, this node may not find the blobs its batches need until the L1 has run long enough - see nodevin's --blob-serving documentation.")

	jwtSecret, err := compose.BaseJWTSecret(stackSuffix(testnet))
	if err != nil {
		return "", fmt.Errorf("failed to prepare the shared Engine API secret: %w", err)
	}

	executionComposeConfig, err := compose.GetBaseExecutionNetworkComposeConfig(network, jwtSecret)
	if err != nil {
		return "", err
	}

	// Pull the consensus image up front, same pattern as Ethereum's own
	// consensus client wiring: the main service's image is pulled implicitly
	// by `docker-compose up -d` later, but the extra service is pulled
	// explicitly first so any warnings/errors surface before that point.
	consensusImage := "fiftysix/base-consensus"
	consensusVersion := "latest"
	if err := pullConsensusImage(fmt.Sprintf("%s:%s", consensusImage, consensusVersion)); err != nil {
		logger.LogError("Failed to pull Docker image: " + err.Error())
		return "", err
	}

	consensusComposeConfig, err := compose.GetBaseConsensusNetworkComposeConfig(consensusNetwork, jwtSecret, l1)
	if err != nil {
		return "", err
	}

	composeFilePath, err := compose.CreateComposeFile(
		executionComposeConfig.ContainerName,
		executionComposeConfig,
		[]string{consensusComposeConfig.ContainerName},
		[]compose.NetworkConfig{consensusComposeConfig},
		cwd)

	if err != nil {
		return "", err
	}

	return composeFilePath, nil
}

func stackSuffix(testnet bool) string {
	if testnet {
		return "-testnet"
	}
	return ""
}
