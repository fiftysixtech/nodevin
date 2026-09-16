package nodes

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/fiftysixcrypto/nodevin/internal/logger"
	"github.com/fiftysixcrypto/nodevin/internal/utils"
	"github.com/spf13/cobra"
)

var viewCmd = &cobra.Command{
	Use:   "view",
	Short: "View your Nodevin in a fun and artistic way!",
	Run: func(cmd *cobra.Command, args []string) {
		nodeSizes := fetchNodeSizes()
		nodeStats := fetchNodeStats()
		displayNodevinArt(nodeSizes, nodeStats)
	},
}

type NodeData struct {
	Network     string `json:"network"`
	Name        string `json:"name"`
	Uptime      int    `json:"uptime"`
	Peers       int    `json:"peers"`
	LatestBlock int    `json:"latest_block"`
}

func fetchNodeSizes() map[string]int64 {
	nodevinDataDir, err := utils.GetNodevinDataDir()
	if err != nil {
		logger.LogError("Failed to find Nodevin data directory: " + err.Error())
		return nil
	}

	networks := utils.GetAllSupportedNetworks()
	if networks == "" {
		fmt.Println("No data found.")
		return nil
	}

	nodeSizes := make(map[string]int64)
	for _, network := range strings.Split(networks, ", ") {
		containerName, exists := utils.GetDefaultLocalMappedContainerName(network)
		if !exists {
			logger.LogError("Unsupported blockchain network: " + network)
			continue
		}

		networkDir := filepath.Join(nodevinDataDir, containerName)
		size, err := getDirectorySize(networkDir)
		if err == nil {
			nodeSizes[network] = size
		}
	}
	return nodeSizes
}

func fetchNodeStats() []NodeData {
	args := []string{"ps", "--format", "{{json .}}"}
	cmd := exec.Command("docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.LogError("Failed to fetch Docker container information: " + err.Error())
		return nil
	}

	containers := strings.Split(string(output), "\n")
	var nodes []NodeData
	for _, containerJSON := range containers {
		if strings.TrimSpace(containerJSON) == "" {
			continue
		}
		var container ContainerInfo
		if err := json.Unmarshal([]byte(containerJSON), &container); err != nil {
			logger.LogError("Failed to parse container JSON: " + err.Error())
			continue
		}
		if !utils.IsSupportedExtendedInfoSoftware(container.Names) {
			continue
		}

		peers := getPeers(container.Names)
		localLatestBlock, _ := getLatestBlocks(container.Names)
		nodes = append(nodes, NodeData{
			Network:     getSoftwareNetworkName(container.Names),
			Name:        container.Names,
			Uptime:      extractUptime(container.Status),
			Peers:       peers,
			LatestBlock: localLatestBlock,
		})
	}
	return nodes
}

func getNodevinName(network string) string {
	switch strings.ToLower(network) {
	case "bitcoin-core":
		return "Bitvin"
	case "litecoin-core":
		return "Litevin"
	case "dogecoin-core":
		return "Dogevin"
	case "core-geth":
		return "Classicvin"
	default:
		return "Nodevin"
	}
}

func getSoftwareNetworkName(softwareName string) string {
	switch softwareName {
	case "bitcoin-core":
		return "bitcoin"
	case "litecoin-core":
		return "litecoin"
	case "dogecoin-core":
		return "dogecoin"
	case "core-geth":
		return "ethereum-classic"
	default:
		return ""
	}
}

func extractUptime(status string) int {
	if strings.Contains(status, "days") {
		var days int
		fmt.Sscanf(status, "%d days", &days)
		return days
	} else if strings.Contains(status, "hours") {
		return 1
	}
	return 0
}

func displayNodevinArt(nodeSizes map[string]int64, nodes []NodeData) {
	if len(nodes) == 0 {
		fmt.Printf("\nVin Status:\n\n")
		fmt.Printf("   (v_v)  \n  /[   ]\\   \n   [   ]   \n Feeling Lonely... \n\n")
		return
	}

	for _, node := range nodes {
		nodeSize := nodeSizes[node.Network]
		vinName := getNodevinName(node.Name)
		fmt.Printf("\n%s Status:\n\n", vinName)

		var art string
		switch {
		case node.Uptime >= 50 && node.Peers >= 5 && node.LatestBlock >= 2000000 && nodeSize > 1000000000000:
			art = "   (n_v)  \n  \\[&&&]/  \n   [!!!]   \nDecentralized Giant!"
		case node.Uptime >= 40 && node.Peers >= 5 && node.LatestBlock >= 1500000 && nodeSize > 1000000000000:
			art = "   (n_v)  \n  \\[+++]/  \n   [===]   \nBlockchain Beacon!"
		case node.Uptime >= 25 && node.Peers >= 3 && node.LatestBlock >= 1000000 && nodeSize > 750000000000:
			art = "   (n_v)  \n  \\[###]/  \n   [***]   \nLeading the Charge!"
		case node.Peers >= 3 && node.LatestBlock >= 500000 && nodeSize > 600000000000:
			art = "   (n_v)  \n  \\[ooo]/  \n   [~~~]   \nConnectivity Master!"
		case nodeSize > 1000000000000:
			art = "   (n_v)  \n  \\[@@@]/  \n   [$$$]   \nOverflowing Power!"
		case nodeSize > 500000000000:
			art = "   (n_v)  \n  \\[>>>]/  \n   [:::]   \nNodeimus Prime!"
		case node.Uptime >= 20 && node.Peers >= 8:
			art = "   (n_v)  \n  \\[ ##]/  \n   [ **]   \nGrowing Network!"
		case node.Uptime >= 10 && node.Peers >= 5:
			art = "   (n_v)  \n  \\[ **]/  \n   [ --]   \nExpanding Presence!"
		case node.Peers >= 5 && node.LatestBlock >= 100000:
			art = "   (n_v)  \n  \\[ **]/  \n   [ ==]   \nStepping Forward!"
		case node.LatestBlock >= 5000:
			art = "   (n_v)  \n  \\[ --]/  \n   [ **]   \nSynchronizing Chain!"
		case node.Uptime >= 1 && node.Peers >= 2:
			art = "   (n_v)  \n  \\[ .-]/  \n   [ ..]   \nEarly Adopter!"
		default:
			art = "   (n_v)  \n  /[   ]\\   \n   [   ]   \nAwaiting Activity..."
		}

		fmt.Printf("%s\n\n", art)
		displayNodeDetails(node, nodeSize)
	}
}

func displayNodeDetails(node NodeData, size int64) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', tabwriter.Debug)
	fmt.Fprintln(w, "| METRIC\t VALUE")
	fmt.Fprintf(w, "| Network\t %s\n", node.Network)
	fmt.Fprintf(w, "| Uptime\t %d days\n", node.Uptime)
	fmt.Fprintf(w, "| Peers\t %d\n", node.Peers)
	fmt.Fprintf(w, "| Latest Block\t %d\n", node.LatestBlock)
	fmt.Fprintf(w, "| Size\t %s\n", utils.GetSizeDescription(size))
	w.Flush()
}
