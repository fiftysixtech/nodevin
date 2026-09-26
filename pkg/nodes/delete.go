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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fiftysixcrypto/nodevin/internal/logger"
	"github.com/fiftysixcrypto/nodevin/internal/utils"
	"github.com/fiftysixcrypto/nodevin/pkg/docker"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete [network-name-or-all]",
	Short: "Delete a directory associated with a network or delete all data",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true

		// Get the nodevin data directory
		nodevinDataDir, err := utils.GetNodevinDataDir()
		if err != nil {
			return fmt.Errorf("failed to find Nodevin data directory: %w", err)
		}

		if len(args) == 0 {
			logger.LogInfo(fmt.Sprintf("Example usage: `%s delete <network>`", utils.GetNodevinExecutable()))
			logger.LogInfo(fmt.Sprintf("Example usage: `%s delete all`", utils.GetNodevinExecutable()))
			return errors.New("no network name provided. To delete network data, specify the name explicitly (for example: bitcoin, litecoin)")
		}

		if args[0] == "all" {
			return deleteAllDirectories(nodevinDataDir)
		}
		return deleteNetworkDirectory(nodevinDataDir, args[0])
	},
}

// deleteNetworkDirectory stops the node for networkName and removes its data.
// It refuses to remove the data of a node that is still running.
func deleteNetworkDirectory(baseDir, networkName string) error {
	// Networks whose primary service can run as several clients (Ethereum's
	// --execution-client) never have a client inferred here: deleting the
	// wrong one's data is not recoverable.
	containerName, err := utils.ExplicitContainerName(networkName)
	if err != nil {
		return err
	}

	// The Ethereum stack was already resolved to its "-testnet" names.
	if utils.CheckIfTestnetOrTestnetNetworkFlag() && !strings.HasSuffix(containerName, "-testnet") {
		containerName = containerName + "-testnet"
	}

	networkDir := filepath.Join(baseDir, containerName)
	if _, err := os.Stat(networkDir); os.IsNotExist(err) {
		return fmt.Errorf("data for network not found: %s", networkDir)
	}

	if owner, inStack := utils.StackOwner(utils.EffectiveNetwork(networkName)); inStack {
		// Runs inside another network's stack: there is nothing to stop here,
		// but its data must not be removed from under it.
		running, err := runningContainers([]string{containerName})
		if err == nil && len(running) > 0 {
			stopStack := fmt.Sprintf("%s stop %s", utils.GetNodevinExecutable(), strings.TrimSuffix(owner, "-testnet"))
			if strings.HasSuffix(owner, "-testnet") {
				stopStack += " --testnet"
			}
			return fmt.Errorf("refusing to delete data: %s is running as part of %s. Run `%s` first", containerName, owner, stopStack)
		}
	} else {
		// Stop network docker container, then check it really stopped: a failed
		// stop must never lead to deleting the data of a live node.
		if err := stopNode(networkName); err != nil {
			logger.LogError("Could not stop the node: " + err.Error())
		}
		if err := ensureNotRunning([]string{containerName}); err != nil {
			return err
		}
	}

	if err := docker.ForceRemoveAll(networkDir); err != nil {
		return fmt.Errorf("failed to remove data for network %s: %w", networkName, err)
	}

	// The generated stack file for a deleted execution client would otherwise
	// linger and make later commands think that client still has a footprint.
	if len(utils.CandidateContainerNames(networkName)) > 1 {
		os.Remove(filepath.Join(baseDir, "docker-compose_"+containerName+".yml"))
	}

	logger.LogInfo(fmt.Sprintf("Successfully removed %s data directory", networkName))

	// Components of this network's stack (Ethereum's consensus client) keep
	// their own data and are only deleted when named explicitly.
	for _, component := range utils.ComponentNetworks(utils.EffectiveNetwork(networkName)) {
		name, _ := utils.GetDefaultLocalMappedContainerName(component)
		if _, err := os.Stat(filepath.Join(baseDir, name)); err == nil {
			logger.LogInfo(fmt.Sprintf("Note: %s data was left in place. Remove it with `%s delete %s`.", component, utils.GetNodevinExecutable(), component))
		}
	}
	return nil
}

func deleteAllDirectories(baseDir string) error {
	// Stop all docker containers
	if err := stopAllNodes(); err != nil {
		logger.LogError("Could not stop all nodes: " + err.Error())
	}

	if err := ensureNotRunning(utils.AllContainerNames()); err != nil {
		return err
	}

	// Remove the entire nodevinDataDir directory
	if err := docker.ForceRemoveAll(baseDir); err != nil {
		return fmt.Errorf("failed to remove all directories: %w", err)
	}

	logger.LogInfo("Successfully removed all nodevin blockchain data")
	return nil
}

// ensureNotRunning returns an error if any of the named containers is running.
// If docker itself cannot be queried there is nothing running to protect, so
// that only produces a warning.
func ensureNotRunning(containers []string) error {
	running, err := runningContainers(containers)
	if err != nil {
		logger.LogInfo("Could not check whether the node is still running (is Docker available?): " + err.Error())
		return nil
	}
	if len(running) > 0 {
		return fmt.Errorf("refusing to delete data: %s still running. Stop it first (if you started it with --data-dir, pass the same --data-dir to stop)", strings.Join(running, ", "))
	}
	return nil
}
