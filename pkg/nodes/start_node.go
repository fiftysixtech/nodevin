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

package nodes

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/fiftysixcrypto/nodevin/internal/logger"
	"github.com/fiftysixcrypto/nodevin/internal/utils"
	"github.com/fiftysixcrypto/nodevin/pkg/docker"
	"github.com/fiftysixcrypto/nodevin/pkg/docker/compose"
	"github.com/fiftysixcrypto/nodevin/pkg/nodes/bitcoin"
	"github.com/fiftysixcrypto/nodevin/pkg/nodes/dogecoin"
	ethereum_classic "github.com/fiftysixcrypto/nodevin/pkg/nodes/ethereum-classic"
	"github.com/fiftysixcrypto/nodevin/pkg/nodes/ipfs"
	"github.com/fiftysixcrypto/nodevin/pkg/nodes/litecoin"
	"github.com/fiftysixcrypto/nodevin/pkg/nodes/ord"
	ord_litecoin "github.com/fiftysixcrypto/nodevin/pkg/nodes/ord-litecoin"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var startNodeCmd = &cobra.Command{
	Use:   "start [network]",
	Short: "Start a blockchain node",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		startNode(args)
	},
}

func startNode(args []string) {
	if len(args) == 0 {
		logger.LogError("No network provided. Nodevin supports any of the following: " + utils.GetCommandSupportedNetworks())
		logger.LogInfo(fmt.Sprintf("Example usage: `%s start <network>`", utils.GetNodevinExecutable()))

		return
	}

	network := args[0]

	containerName, exists := utils.GetFiftysixDockerhubContainerName(network)
	if !exists {
		logger.LogError("Unsupported blockchain network: " + network)
		return
	}

	logger.LogInfo("Starting blockchain node for network: " + network)

	// Initialize Docker client
	if err := docker.InitDockerClient(); err != nil {
		logger.LogError("Failed to initialize Docker client: " + err.Error())
		return
	}

	version := viper.GetString("version")
	if version == "" {
		version = "latest"
	}
	image := containerName + ":" + version

	if err := docker.PullImage(image); err != nil {
		logger.LogError("Failed to pull Docker image: " + err.Error())
		return
	}

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		logger.LogError("failed to get current working directory: " + err.Error())
		return
	}

	// Create env file for chain compose
	composeFilePath, err := createComposeFileForNetwork(network, cwd)
	if err != nil {
		logger.LogError("Failed to create node docker compose file: " + err.Error())
	}

	// Print out warning info for chain size and snapshot sync timing
	if viper.GetBool("snapshot-sync") {
		logger.LogInfo("INFO: Snapshot sync is currently disabled. Continuing ")

		/*
			snapshotSize, exists := utils.GetNetworkRequiredSnapshotSize(network)

			if !exists {
				logger.LogInfo("Cannot determine assumed size for network.")
				snapshotSize = 0
			}

			logger.LogInfo("--")
			logger.LogInfo("WARNING: Initial snapshot sync can take hours depending on your download speed and computer specs. Nodevin will automatically start up your node after the download completes.")
			logger.LogInfo(fmt.Sprintf("WARNING: Snapshot sync for this software requires %s amount of space. Ensure you have enough storage on disk.", utils.GetSizeDescription(int64(snapshotSize))))

			if viper.GetBool("ord") {
				dataSize, exists := utils.GetNetworkRequiredSnapshotSize("ord")

				if !exists {
					logger.LogInfo("Cannot determine assumed size for ord.")
					dataSize = 0
				}

				logger.LogInfo(fmt.Sprintf("WARNING: Ord software requires an additional %s amount of snapshot space. Ensure you have enough storage on disk for both.", utils.GetSizeDescription(int64(dataSize))))
			} else if viper.GetBool("ord-litecoin") {
				dataSize, exists := utils.GetNetworkRequiredSnapshotSize("ord-litecoin")

				if !exists {
					logger.LogInfo("Cannot determine assumed size for ord-litecoin.")
					dataSize = 0
				}

				logger.LogInfo(fmt.Sprintf("WARNING: Ord-litecoin software requires an additional %s amount of snapshot space. Ensure you have enough storage on disk for both.", utils.GetSizeDescription(int64(dataSize))))
			}
		*/

		logger.LogInfo("--")
	} else if network == "ipfs" {
		logger.LogInfo("No assumed size for network, depends on user input.")
	} else {
		dataSize, exists := utils.GetNetworkRequiredDataSize(network)

		if !exists {
			logger.LogInfo("Cannot determine assumed size for network.")
			dataSize = 0
		}

		logger.LogInfo("--")
		logger.LogInfo("WARNING: Initial chain sync can take hours or days depending on your computer specs.")
		logger.LogInfo(fmt.Sprintf("WARNING: This software requires %s amount of space. Ensure you have enough storage on disk.", utils.GetSizeDescription(int64(dataSize))))

		if viper.GetBool("ord") {
			dataSize, exists := utils.GetNetworkRequiredDataSize("ord")

			if !exists {
				logger.LogInfo("Cannot determine assumed size for ord.")
				dataSize = 0
			}

			logger.LogInfo(fmt.Sprintf("WARNING: Ord software requires an additional %s amount of space. Ensure you have enough storage on disk for both.", utils.GetSizeDescription(int64(dataSize))))
		} else if viper.GetBool("ord-litecoin") {
			dataSize, exists := utils.GetNetworkRequiredDataSize("ord-litecoin")

			if !exists {
				logger.LogInfo("Cannot determine assumed size for ord-litecoin.")
				dataSize = 0
			}

			logger.LogInfo(fmt.Sprintf("WARNING: Ord-litecoin software requires an additional %s amount of space. Ensure you have enough storage on disk for both.", utils.GetSizeDescription(int64(dataSize))))
		}

		logger.LogInfo("--")
	}

	// Start the node
	cmd := exec.Command("docker-compose", "-f", composeFilePath, "up", "-d")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		logger.LogError("Failed to start Docker Compose services: " + err.Error())
		return
	}

	logger.LogInfo("Cleaning up excess containers and volumes...")
	if err := compose.RemoveInitContainersAndVolumes(); err != nil {
		logger.LogError("Failed to clean up excess init containers and volumes: " + err.Error())
	} else {
		logger.LogInfo("Successfully cleaned up excess containers and volumes.")
	}

	logger.LogInfo("Successfully started blockchain node for network: " + network)

	startMessage, _ := utils.GetStartMessage(network)

	fmt.Printf("\n%s\n", startMessage)
}

func createComposeFileForNetwork(network string, cwd string) (string, error) {
	switch network {
	case "bitcoin":
		return bitcoin.CreateBitcoinComposeFile(cwd)
	case "ord":
		return ord.CreateOrdComposeFile(cwd)
	case "litecoin":
		return litecoin.CreateLitecoinComposeFile(cwd)
	case "ord-litecoin":
		return ord_litecoin.CreateOrdLitecoinComposeFile(cwd)
	case "ipfs":
		return ipfs.CreateKuboComposeFile(cwd)
	case "dogecoin":
		return dogecoin.CreateDogecoinComposeFile(cwd)
	case "ethereum-classic":
		return ethereum_classic.CreateEthereumClassicComposeFile(cwd)
	default:
		return "", fmt.Errorf("unsupported network: %s", network)
	}
}
