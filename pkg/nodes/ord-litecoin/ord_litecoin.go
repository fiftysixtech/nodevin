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

package ord_litecoin

import (
	"errors"
	"fmt"
	"runtime"

	"github.com/fiftysixcrypto/nodevin/internal/logger"
	"github.com/fiftysixcrypto/nodevin/internal/utils"
	"github.com/fiftysixcrypto/nodevin/pkg/docker"
	"github.com/fiftysixcrypto/nodevin/pkg/docker/compose"
	"github.com/spf13/viper"
)

// resolveOrdLitecoinNetwork picks the network name to pass to
// compose.GetOrdLitecoinNetworkComposeConfig based on the --testnet/--network
// flags. Kept as its own pure function so this decision is unit-testable
// without going through CreateOrdLitecoinComposeFile's Docker-dependent tail
// (docker.PullImage, which requires a real Docker daemon and a logger
// initialized via logger.Init(), neither of which are available in a plain
// unit test).
func resolveOrdLitecoinNetwork() string {
	if utils.CheckIfTestnetOrTestnetNetworkFlag() {
		return "ord-litecoin-testnet"
	}
	return "ord-litecoin"
}

func CreateOrdLitecoinComposeFile(cwd string) (string, error) {
	if runtime.GOARCH == "arm64" {
		err := errors.New("ord-litecoin functionality is not supported on ARM builds")
		logger.LogError("Running on ARM architecture: " + err.Error())
		return "", err
	}

	fmt.Printf("WARNING: It isn't reccomended to start ord-litecoin individually. Most cases would require starting ord-litecoin alongside Litecoin with command `%s start litecoin --ord-litecoin`. You may run into unintentional errors or require additional configuration.", utils.GetNodevinExecutable())

	network := resolveOrdLitecoinNetwork()

	ordLitecoinBaseComposeConfig, err := compose.GetOrdLitecoinNetworkComposeConfig(network)
	if err != nil {
		return "", err
	}

	// Pull the ord Docker image
	image := viper.GetString("ord-litecoin-image") + ":" + viper.GetString("ord-litecoin-version")
	if err := docker.PullImage(image); err != nil {
		logger.LogError("Failed to pull Docker image: " + err.Error())
		return "", err
	}

	composeFilePath, err := compose.CreateComposeFile(
		ordLitecoinBaseComposeConfig.ContainerName,
		ordLitecoinBaseComposeConfig,
		[]string{},
		[]compose.NetworkConfig{},
		cwd)

	if err != nil {
		return "", err
	}

	return composeFilePath, nil
}
