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

package bsc

import (
	"github.com/fiftysixcrypto/nodevin/internal/utils"
	"github.com/fiftysixcrypto/nodevin/pkg/docker/compose"
)

// CreateBscComposeFile builds the compose file for fiftysix/bsc on "bsc"
// (mainnet) or "bsc-testnet" (--testnet). Unlike Arbitrum and Base, BSC has
// no L1 dependency - Parlia consensus is built into the same process - so
// there is no L1 resolution step here.
func CreateBscComposeFile(cwd string) (string, error) {
	network := "bsc"
	if utils.CheckIfTestnetOrTestnetNetworkFlag() {
		network = "bsc-testnet"
	}

	bscComposeConfig, err := compose.GetBscNetworkComposeConfig(network)
	if err != nil {
		return "", err
	}

	composeFilePath, err := compose.CreateComposeFile(
		bscComposeConfig.ContainerName,
		bscComposeConfig,
		[]string{},
		[]compose.NetworkConfig{},
		cwd)

	if err != nil {
		return "", err
	}

	return composeFilePath, nil
}
