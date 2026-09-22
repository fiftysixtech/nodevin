package nodes

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/fiftysixcrypto/nodevin/internal/logger"
	"github.com/fiftysixcrypto/nodevin/internal/utils"
	"github.com/fiftysixcrypto/nodevin/pkg/docker"
	"github.com/spf13/cobra"
)

var stopNodeCmd = &cobra.Command{
	Use:   "stop [network]",
	Short: "Stop a blockchain node",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true

		if len(args) == 0 {
			logger.LogInfo("List of available networks: " + utils.GetCommandSupportedNetworks())
			logger.LogInfo(fmt.Sprintf("Example usage: `%s stop <network>`", utils.GetNodevinExecutable()))
			logger.LogInfo(fmt.Sprintf("Example usage: `%s stop <network> --testnet`", utils.GetNodevinExecutable()))
			logger.LogInfo(fmt.Sprintf("Example usage: `%s stop all`", utils.GetNodevinExecutable()))
			return errors.New("no network specified. To stop a node, specify the network explicitly")
		}

		if args[0] == "all" {
			return stopAllNodes()
		}
		return stopNode(args[0])
	},
}

// stopNode stops the node for network using the compose file nodevin generated
// for it, wherever --data-dir put it. Stopping a node that is not running is
// not an error.
func stopNode(network string) error {
	logger.LogInfo("Stopping blockchain node...")

	containerName, exists := utils.GetDefaultLocalMappedContainerName(network)
	if !exists {
		return fmt.Errorf("unsupported blockchain network: %s", network)
	}

	if utils.CheckIfTestnetOrTestnetNetworkFlag() {
		containerName = containerName + "-testnet"
	}

	composeFilePath, err := utils.FindComposeFile(containerName)
	if err != nil {
		return err
	}

	// Check if there are any running containers for this compose file
	psCmd, err := docker.ComposeCommand("-f", composeFilePath, "ps", "-q")
	if err != nil {
		return err
	}
	psOut, err := psCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to find Docker Compose services: %w", err)
	}

	if len(psOut) == 0 {
		logger.LogInfo("No running containers found for the specified network (did you mean to add --testnet?)")
		return nil
	}

	cmd, err := docker.ComposeCommand("-f", composeFilePath, "down")
	if err != nil {
		return err
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stop Docker Compose services: %w", err)
	}

	logger.LogInfo("Blockchain node stopped successfully.")
	return nil
}

// runningContainers returns which of names are running right now. It returns
// an error when docker cannot be queried.
func runningContainers(names []string) ([]string, error) {
	out, err := exec.Command("docker", "ps", "--format", "{{.Names}}").Output()
	if err != nil {
		return nil, err
	}

	wanted := make(map[string]bool, len(names))
	for _, name := range names {
		wanted[name] = true
	}

	var running []string
	for _, line := range strings.Split(string(out), "\n") {
		if name := strings.TrimSpace(line); wanted[name] {
			running = append(running, name)
		}
	}
	return running, nil
}

func stopAllNodes() error {
	logger.LogInfo("Stopping Docker Compose containers...")

	// Get the map of allowed network container names
	networkContainerMap := utils.NetworkContainerMap()
	allowedContainers := make(map[string]bool)
	for _, containerName := range networkContainerMap {
		allowedContainers[containerName] = true
	}

	// Include watchtower in container shutdowns
	allowedContainers["watchtower-nodevin"] = true

	// Get a list of running Docker container IDs with their names
	psCmd := exec.Command("docker", "ps", "--format", "{{.ID}} {{.Names}}")
	var psOut bytes.Buffer
	psCmd.Stdout = &psOut
	psCmd.Stderr = os.Stderr

	if err := psCmd.Run(); err != nil {
		return fmt.Errorf("failed to list running Docker containers: %w", err)
	}

	// Filter containers that match allowed container names
	var containerIDs []string
	lines := strings.Split(strings.TrimSpace(psOut.String()), "\n")
	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) != 2 {
			continue
		}
		containerID, containerName := parts[0], parts[1]
		if allowedContainers[containerName] {
			containerIDs = append(containerIDs, containerID)
		}
	}

	if len(containerIDs) == 0 {
		logger.LogInfo("No matching Docker Compose containers found.")
		return nil
	}

	// Stop the containers
	logger.LogInfo("Stopping containers: " + strings.Join(containerIDs, ", "))
	stopCmd := exec.Command("docker", append([]string{"stop"}, containerIDs...)...)
	stopCmd.Stdout = os.Stdout
	stopCmd.Stderr = os.Stderr

	if err := stopCmd.Run(); err != nil {
		return fmt.Errorf("failed to stop Docker containers: %w", err)
	}

	// Remove the containers
	logger.LogInfo("Removing containers: " + strings.Join(containerIDs, ", "))
	rmCmd := exec.Command("docker", append([]string{"rm"}, containerIDs...)...)
	rmCmd.Stdout = os.Stdout
	rmCmd.Stderr = os.Stderr

	if err := rmCmd.Run(); err != nil {
		return fmt.Errorf("failed to remove Docker containers: %w", err)
	}

	logger.LogInfo("Selected Docker Compose containers stopped and removed successfully.")
	return nil
}
