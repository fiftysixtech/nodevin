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

package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/spf13/viper"
)

type NetworkInfo struct {
	ContainerName string
	// AlternateContainerNames are other containers the network's primary
	// service may run as (e.g. Ethereum's execution client is chosen with
	// --execution-client). ContainerName is the default. See ResolveContainerName.
	AlternateContainerNames []string
	// ClientFlag names the CLI flag that selects between ContainerName and
	// AlternateContainerNames.
	ClientFlag string
	// PartOf marks a network that only ever runs inside another network's
	// compose stack (e.g. lighthouse inside ethereum): it has no compose file
	// of its own and cannot be stopped independently.
	PartOf           string
	DockerHubImage   string
	SnapshotCID      string
	RPCPort          int
	DataSize         int
	SnapshotSize     int
	StartMessage     string
	CommandSupported bool
}

var networkInfoMap = map[string]NetworkInfo{
	"bitcoin": {
		ContainerName:    "bitcoin-core",
		DockerHubImage:   "bitcoin-core",
		RPCPort:          8332,
		SnapshotCID:      "QmbTy7qCfPJYengA8zew1Ng2vzJFdLtFAx3fxyYFUptoHR",
		DataSize:         708669603840,  // 660 GB
		SnapshotSize:     1319413953331, // 1.2TB (~660 GB + ~540 GB)
		StartMessage:     "\"A system for electronic transactions without relying on trust.\" -- Satoshi Nakamoto",
		CommandSupported: true,
	},
	"bitcoin-testnet": {
		ContainerName:    "bitcoin-core-testnet",
		DockerHubImage:   "bitcoin-core",
		RPCPort:          18332,
		SnapshotCID:      "",
		DataSize:         0,
		SnapshotSize:     0,
		StartMessage:     "\"Testing is the lifeblood of innovation and security.\"",
		CommandSupported: false,
	},
	"ord": {
		ContainerName:    "ord",
		DockerHubImage:   "ord",
		RPCPort:          80,
		SnapshotCID:      "QmYt17T4GXF3Hvv2tX8M1H6hq7zQXmDM77EGp2c7Q5wM63",
		DataSize:         182536110080, // 170 GB
		SnapshotSize:     279172874240, // (~170GB + ~90GB)
		StartMessage:     "\"Ordinal theory imbues satoshis with numismatic value, allowing them to be collected and traded as curios.\"",
		CommandSupported: true,
	},
	"ord-testnet": {
		ContainerName:    "ord-testnet",
		DockerHubImage:   "ord",
		RPCPort:          8081,
		SnapshotCID:      "",
		DataSize:         0,
		SnapshotSize:     0,
		StartMessage:     "\"Ordinal theory imbues satoshis with numismatic value, allowing them to be collected and traded as curios.\"",
		CommandSupported: false,
	},
	"litecoin": {
		ContainerName:    "litecoin-core",
		DockerHubImage:   "litecoin-core",
		RPCPort:          9332,
		SnapshotCID:      "QmPovkiCovehHEDKCe4YqyHAArVohZUVZiLMzrut8JLXn9",
		DataSize:         268435456000, // 250 GB
		SnapshotSize:     429496729600, // 400 GB (~240GB + ~160GB)
		StartMessage:     "\"Litecoin is the silver to Bitcoin's gold.\" -- Charlie Lee",
		CommandSupported: true,
	},
	"litecoin-testnet": {
		ContainerName:    "litecoin-core-testnet",
		DockerHubImage:   "litecoin-core",
		RPCPort:          19332,
		SnapshotCID:      "",
		DataSize:         0,
		SnapshotSize:     0,
		StartMessage:     "\"Testing is the lifeblood of innovation and security.\"",
		CommandSupported: false,
	},
	"ord-litecoin": {
		ContainerName:    "ord-litecoin",
		DockerHubImage:   "ord-litecoin",
		RPCPort:          8082,
		SnapshotCID:      "",
		DataSize:         107387498496, // 100 GB
		SnapshotSize:     0,            // (~100GB + ~100GB)
		StartMessage:     "\"Ordinal theory imbues satoshis with numismatic value, allowing them to be collected and traded as curios.\"",
		CommandSupported: true,
	},
	"ord-litecoin-testnet": {
		ContainerName:    "ord-litecoin-testnet",
		DockerHubImage:   "ord-litecoin",
		RPCPort:          8083,
		SnapshotCID:      "",
		DataSize:         0,
		SnapshotSize:     0,
		StartMessage:     "\"Ordinal theory imbues satoshis with numismatic value, allowing them to be collected and traded as curios.\"",
		CommandSupported: false,
	},
	"ipfs": {
		ContainerName:    "ipfs",
		DockerHubImage:   "kubo",
		RPCPort:          5001,
		SnapshotCID:      "",
		DataSize:         0, // needs review
		SnapshotSize:     0, // needs review
		StartMessage:     "\"A peer-to-peer media protocol to make the web safer, faster, and more open.\" -- IPFS",
		CommandSupported: true,
	},
	"ipfs-cluster": {
		ContainerName:    "ipfs-cluster",
		DockerHubImage:   "ipfs-cluster",
		RPCPort:          9094,
		SnapshotCID:      "",
		DataSize:         0, // needs review
		SnapshotSize:     0, // needs review
		StartMessage:     "\"A network of nodes working together to preserve and share data reliably.\"",
		CommandSupported: false,
	},
	"dogecoin": {
		ContainerName:    "dogecoin-core",
		DockerHubImage:   "dogecoin-core",
		RPCPort:          22555,
		SnapshotCID:      "",
		DataSize:         0,
		SnapshotSize:     0,
		StartMessage:     "\"Dogecoin to the moon.\" -- Dogecoin Community",
		CommandSupported: true,
	},
	"dogecoin-testnet": {
		ContainerName:    "dogecoin-core-testnet",
		DockerHubImage:   "dogecoin-core",
		RPCPort:          44555,
		SnapshotCID:      "",
		DataSize:         0,
		SnapshotSize:     0,
		StartMessage:     "\"Testing is the lifeblood of innovation and security.\"",
		CommandSupported: false,
	},
	"ethereum-classic": {
		ContainerName:    "core-geth",
		DockerHubImage:   "core-geth",
		RPCPort:          8545,
		SnapshotCID:      "",
		DataSize:         0,
		SnapshotSize:     0,
		StartMessage:     "\"Code is Law.\" -- Ethereum Classic Community",
		CommandSupported: true,
	},
	"ethereum-classic-testnet": {
		ContainerName:    "core-geth-testnet",
		DockerHubImage:   "core-geth",
		RPCPort:          8546,
		SnapshotCID:      "",
		DataSize:         0,
		SnapshotSize:     0,
		StartMessage:     "\"Testing is the lifeblood of innovation and security.\"",
		CommandSupported: false,
	},
	// "ethereum" registers the DEFAULT execution client ("reth") as its
	// ContainerName; --execution-client can select any of the alternates. Use
	// ResolveContainerName, not this static name, to find what is running.
	"ethereum": {
		ContainerName:           "reth",
		AlternateContainerNames: []string{"geth", "erigon", "besu", "nethermind"},
		ClientFlag:              "execution-client",
		DockerHubImage:          "reth",
		RPCPort:                 8547,
		SnapshotCID:             "",
		DataSize:                0,
		SnapshotSize:            0,
		StartMessage:            "\"Not your keys, not your coins.\" -- Ethereum Community",
		CommandSupported:        true,
	},
	// The five consensus clients each get their own registry entry (mirroring
	// how "ord" is registered despite being bundled with bitcoin) so logs,
	// shell and delete can target one directly. They run inside the ethereum
	// stack (PartOf), so stop goes through "ethereum".
	"lighthouse": {
		PartOf:           "ethereum",
		ContainerName:    "lighthouse",
		DockerHubImage:   "lighthouse",
		RPCPort:          5052,
		SnapshotCID:      "",
		DataSize:         0,
		SnapshotSize:     0,
		StartMessage:     "\"Proof of stake is coming.\" -- Ethereum Foundation",
		CommandSupported: false,
	},
	"prysm": {
		PartOf:           "ethereum",
		ContainerName:    "prysm",
		DockerHubImage:   "prysm",
		RPCPort:          3500,
		SnapshotCID:      "",
		DataSize:         0,
		SnapshotSize:     0,
		StartMessage:     "\"Proof of stake is coming.\" -- Ethereum Foundation",
		CommandSupported: false,
	},
	"teku": {
		PartOf:           "ethereum",
		ContainerName:    "teku",
		DockerHubImage:   "teku",
		RPCPort:          5051,
		SnapshotCID:      "",
		DataSize:         0,
		SnapshotSize:     0,
		StartMessage:     "\"Proof of stake is coming.\" -- Ethereum Foundation",
		CommandSupported: false,
	},
	"nimbus": {
		PartOf:           "ethereum",
		ContainerName:    "nimbus",
		DockerHubImage:   "nimbus",
		RPCPort:          5052,
		SnapshotCID:      "",
		DataSize:         0,
		SnapshotSize:     0,
		StartMessage:     "\"Proof of stake is coming.\" -- Ethereum Foundation",
		CommandSupported: false,
	},
	"lodestar": {
		PartOf:           "ethereum",
		ContainerName:    "lodestar",
		DockerHubImage:   "lodestar",
		RPCPort:          9596,
		SnapshotCID:      "",
		DataSize:         0,
		SnapshotSize:     0,
		StartMessage:     "\"Proof of stake is coming.\" -- Ethereum Foundation",
		CommandSupported: false,
	},
	// Sepolia runs as its own stack: every container and data directory carries
	// the "-testnet" suffix, so it never touches mainnet data. Only one
	// Ethereum stack (mainnet or testnet) runs at a time, since the consensus
	// clients publish the same host ports in both.
	"ethereum-testnet": {
		ContainerName:           "reth-testnet",
		AlternateContainerNames: []string{"geth-testnet", "erigon-testnet", "besu-testnet", "nethermind-testnet"},
		ClientFlag:              "execution-client",
		DockerHubImage:          "reth",
		RPCPort:                 8549,
		SnapshotCID:             "",
		DataSize:                0,
		SnapshotSize:            0,
		StartMessage:            "\"Testing is the lifeblood of innovation and security.\"",
		CommandSupported:        false,
	},
	"lighthouse-testnet": {
		PartOf:           "ethereum-testnet",
		ContainerName:    "lighthouse-testnet",
		DockerHubImage:   "lighthouse",
		RPCPort:          5052,
		SnapshotCID:      "",
		DataSize:         0,
		SnapshotSize:     0,
		StartMessage:     "\"Testing is the lifeblood of innovation and security.\"",
		CommandSupported: false,
	},
	"prysm-testnet": {
		PartOf:           "ethereum-testnet",
		ContainerName:    "prysm-testnet",
		DockerHubImage:   "prysm",
		RPCPort:          3500,
		SnapshotCID:      "",
		DataSize:         0,
		SnapshotSize:     0,
		StartMessage:     "\"Testing is the lifeblood of innovation and security.\"",
		CommandSupported: false,
	},
	"teku-testnet": {
		PartOf:           "ethereum-testnet",
		ContainerName:    "teku-testnet",
		DockerHubImage:   "teku",
		RPCPort:          5051,
		SnapshotCID:      "",
		DataSize:         0,
		SnapshotSize:     0,
		StartMessage:     "\"Testing is the lifeblood of innovation and security.\"",
		CommandSupported: false,
	},
	"nimbus-testnet": {
		PartOf:           "ethereum-testnet",
		ContainerName:    "nimbus-testnet",
		DockerHubImage:   "nimbus",
		RPCPort:          5052,
		SnapshotCID:      "",
		DataSize:         0,
		SnapshotSize:     0,
		StartMessage:     "\"Testing is the lifeblood of innovation and security.\"",
		CommandSupported: false,
	},
	"lodestar-testnet": {
		PartOf:           "ethereum-testnet",
		ContainerName:    "lodestar-testnet",
		DockerHubImage:   "lodestar",
		RPCPort:          9596,
		SnapshotCID:      "",
		DataSize:         0,
		SnapshotSize:     0,
		StartMessage:     "\"Testing is the lifeblood of innovation and security.\"",
		CommandSupported: false,
	},
}

func NetworkContainerMap() map[string]string {
	networkContainerMap := make(map[string]string)
	for key, value := range networkInfoMap {
		networkContainerMap[key] = value.ContainerName
	}
	return networkContainerMap
}

func NetworkDefaultRPCPorts() map[string]int {
	networkDefaultRPCPorts := make(map[string]int)
	for key, value := range networkInfoMap {
		networkDefaultRPCPorts[key] = value.RPCPort
	}
	return networkDefaultRPCPorts
}

func GetStartMessage(network string) (string, bool) {
	networkInfo, exists := networkInfoMap[network]
	return networkInfo.StartMessage, exists
}

func GetDefaultLocalMappedContainerName(network string) (string, bool) {
	networkInfo, exists := networkInfoMap[network]
	return networkInfo.ContainerName, exists
}

func GetNetworkRequiredDataSize(network string) (int, bool) {
	networkInfo, exists := networkInfoMap[network]
	return networkInfo.DataSize, exists
}

func GetNetworkRequiredSnapshotSize(network string) (int, bool) {
	networkInfo, exists := networkInfoMap[network]
	return networkInfo.SnapshotSize, exists
}

func GetFiftysixDockerhubContainerName(network string) (string, bool) {
	networkInfo, exists := networkInfoMap[network]
	return "fiftysix/" + networkInfo.DockerHubImage, exists
}

func GetAllSupportedNetworks() string {
	var keys []string
	for key := range networkInfoMap {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	return strings.Join(keys, ", ")
}

func GetCommandSupportedNetworks() string {
	var commandSupportedNetworks []string
	for key, value := range networkInfoMap {
		if value.CommandSupported {
			commandSupportedNetworks = append(commandSupportedNetworks, key)
		}
	}
	sort.Strings(commandSupportedNetworks)

	return strings.Join(commandSupportedNetworks, ", ")
}

func CheckIfTestnetOrTestnetNetworkFlag() bool {
	networkFlag := viper.GetString("network")
	testnetFlag := viper.GetBool("testnet")

	return testnetFlag || networkFlag == "testnet"
}

func GetSnapshotCIDByNetwork(network string) (string, bool) {
	networkInfo, exists := networkInfoMap[network]
	return networkInfo.SnapshotCID, exists
}

func IsSupportedExtendedInfoSoftware(software string) bool {
	if software == "bitcoin-core" || software == "litecoin-core" || software == "dogecoin-core" || software == "core-geth" {
		return true
	}
	return IsEthereumExecutionClient(software)
}

// IsEthereumExecutionClient reports whether name is one of the container names
// an Ethereum stack (mainnet or testnet) can run its execution client as.
func IsEthereumExecutionClient(name string) bool {
	for _, network := range []string{"ethereum", "ethereum-testnet"} {
		for _, candidate := range CandidateContainerNames(network) {
			if candidate == name {
				return true
			}
		}
	}
	return false
}

// EthereumExecutionNetwork returns the registry key ("ethereum" or
// "ethereum-testnet") whose stack an execution client container belongs to, or
// "" if name is not an Ethereum execution client.
func EthereumExecutionNetwork(name string) string {
	if !IsEthereumExecutionClient(name) {
		return ""
	}
	if strings.HasSuffix(name, "-testnet") {
		return "ethereum-testnet"
	}
	return "ethereum"
}

// Expands a leading "~" or "~/" in path to the current user's home directory.
// filepath.Join and os.MkdirAll treat "~" as a literal directory name, not a
// shell shortcut, so a value like "~/Desktop" must be expanded manually before
// use or it silently creates a "~" directory relative to the current working
// directory instead of resolving to the user's home.
func ExpandHomeDir(path string) (string, error) {
	if path != "~" && !strings.HasPrefix(path, "~/") {
		return path, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to expand '~' in path: %v", err)
	}

	if path == "~" {
		return homeDir, nil
	}

	return filepath.Join(homeDir, path[2:]), nil
}

// Returns path to the user's nodevin data directory (~/.nodevin/data)
func GetNodevinDataDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %v", err)
	}

	if viper.IsSet("data-dir") {
		homeDir, err = ExpandHomeDir(viper.GetString("data-dir"))
		if err != nil {
			return "", err
		}
	}

	nodevinDataDir := filepath.Join(homeDir, ".nodevin", "data")

	// Create the directory if it doesn't exist
	if _, err := os.Stat(nodevinDataDir); os.IsNotExist(err) {
		err = os.MkdirAll(nodevinDataDir, os.ModePerm)
		if err != nil {
			return "", fmt.Errorf("failed to create nodevin data directory: %v", err)
		}
	}

	return nodevinDataDir, nil
}

// FindComposeFile locates the docker-compose file nodevin generated for a
// container ("docker-compose_<container>.yml"). It looks in the nodevin data
// directory first, which honors --data-dir, then in the directory of the
// nodevin executable and the current directory, where older versions wrote
// them.
func FindComposeFile(containerName string) (string, error) {
	fileName := fmt.Sprintf("docker-compose_%s.yml", containerName)

	var dirs []string
	if dataDir, err := GetNodevinDataDir(); err == nil {
		dirs = append(dirs, dataDir)
	}
	if exe, err := os.Executable(); err == nil {
		dirs = append(dirs, filepath.Dir(exe))
	}
	if cwd, err := os.Getwd(); err == nil {
		dirs = append(dirs, cwd)
	}

	for _, dir := range dirs {
		path := filepath.Join(dir, fileName)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("could not find %s in %s (if you started the node with --data-dir, pass the same --data-dir here)", fileName, strings.Join(dirs, ", "))
}

func GetSizeDescription(size int64) string {
	if size <= 0 {
		return "unknown (do you have proper permissions?)"
	}

	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
		PB = TB * 1024
	)

	switch {
	case size >= PB:
		return fmt.Sprintf("%.2f PB", float64(size)/PB)
	case size >= TB:
		return fmt.Sprintf("%.2f TB", float64(size)/TB)
	case size >= GB:
		return fmt.Sprintf("%.2f GB", float64(size)/GB)
	case size >= MB:
		return fmt.Sprintf("%.2f MB", float64(size)/MB)
	case size >= KB:
		return fmt.Sprintf("%.2f KB", float64(size)/KB)
	default:
		return fmt.Sprintf("%d B", size)
	}
}

func GetNodevinExecutable() string {
	if runtime.GOOS == "windows" {
		return "nodevin.exe"
	}
	return "nodevin"
}
