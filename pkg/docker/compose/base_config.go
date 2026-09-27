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
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fiftysixcrypto/nodevin/internal/utils"
)

// baseDockerNetwork is the Docker network fiftysix/base-reth and
// fiftysix/base-consensus share, per stack suffix ("" mainnet, "-testnet"
// Sepolia).
func baseDockerNetwork(suffix string) string {
	return "base" + suffix + "-net"
}

// baseChain returns Base's own --chain/RETH_CHAIN and BASE_NODE_NETWORK
// value, and the L1 chain (for logging/validation only - unlike Arbitrum,
// Base's images resolve their own rollup config from this name alone, no
// separate parent-chain ID flag is needed).
func baseChain(suffix string) string {
	if suffix != "" {
		return "base-sepolia"
	}
	return "base"
}

// BaseJWTSecret returns the shared Engine API JWT secret for a Base stack,
// generating and persisting a new one on first use. Unlike every Ethereum
// execution client image, fiftysix/base-reth and fiftysix/base-consensus
// never generate this themselves - they each independently write whatever
// raw value they are given (BASE_NODE_L2_ENGINE_AUTH_RAW) to their own local
// file on every start (see docs/base.md in node-images). nodevin is
// therefore the one responsible for minting it once and reusing the same
// value on every subsequent compose regeneration, the same way geth/besu's
// own entrypoints only generate jwt.hex when it does not already exist.
func BaseJWTSecret(suffix string) (string, error) {
	nodevinDataDir, err := utils.GetNodevinDataDir()
	if err != nil {
		return "", err
	}

	dir := filepath.Join(nodevinDataDir, "base"+suffix)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory for the Base JWT secret: %w", err)
	}

	path := filepath.Join(dir, "jwt.hex")
	if data, err := os.ReadFile(path); err == nil {
		secret := strings.TrimSpace(string(data))
		if len(secret) == 64 {
			return secret, nil
		}
	}

	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("failed to generate a JWT secret: %w", err)
	}
	secret := hex.EncodeToString(raw)

	if err := os.WriteFile(path, []byte(secret), 0600); err != nil {
		return "", fmt.Errorf("failed to persist the Base JWT secret: %w", err)
	}
	return secret, nil
}

// baseExecutionPorts returns fiftysix/base-reth's port mappings: loopback-only
// HTTP-RPC on rpcPort, plus P2P (tcp+udp) and discv5 (udp) published
// publicly. Mainnet and testnet get different P2P/discv5 host ports (unlike
// the RPC port, reusing the same ones would collide between the two stacks),
// chosen to avoid 30303/30304, already used by ethereum-classic/-testnet.
func baseExecutionPorts(suffix string, rpcPort int) []string {
	p2pPort, discv5Port := 30307, 9200
	if suffix != "" {
		p2pPort, discv5Port = 30308, 9201
	}
	return []string{
		fmt.Sprintf("127.0.0.1:%d:8545", rpcPort),
		fmt.Sprintf("127.0.0.1:%d:8546", rpcPort+1),
		fmt.Sprintf("%d:%d", p2pPort, p2pPort),
		fmt.Sprintf("%d:%d/udp", p2pPort, p2pPort),
		fmt.Sprintf("%d:%d/udp", discv5Port, discv5Port),
	}
}

// baseConsensusPorts returns fiftysix/base-consensus's port mappings:
// loopback-only RPC on rpcPort, plus P2P (libp2p tcp, discv5 udp) published
// publicly. Mainnet and testnet get different P2P host ports for the same
// reason as baseExecutionPorts.
func baseConsensusPorts(suffix string, rpcPort int) []string {
	p2pTCPPort, p2pUDPPort := 9222, 9223
	if suffix != "" {
		p2pTCPPort, p2pUDPPort = 9224, 9225
	}
	return []string{
		fmt.Sprintf("127.0.0.1:%d:9545", rpcPort),
		fmt.Sprintf("%d:%d", p2pTCPPort, p2pTCPPort),
		fmt.Sprintf("%d:%d/udp", p2pUDPPort, p2pUDPPort),
	}
}

// GetBaseExecutionNetworkComposeConfig builds the compose config for
// fiftysix/base-reth on "base" (mainnet) or "base-testnet" (Base Sepolia).
// Unlike every Ethereum execution client image, base-reth's entrypoint is
// entirely environment-variable driven and takes no CLI flags at all
// (confirmed by reading node-images' entrypoint.sh for it) - Command is
// intentionally empty.
func GetBaseExecutionNetworkComposeConfig(network, jwtSecret string) (NetworkConfig, error) {
	nodevinDataDir, err := utils.GetNodevinDataDir()
	if err != nil {
		return NetworkConfig{}, err
	}

	suffix := stackSuffix(network)
	containerName, ok := utils.GetDefaultLocalMappedContainerName(network)
	if !ok {
		return NetworkConfig{}, fmt.Errorf("unknown network: %s", network)
	}

	localPath := filepath.Join(nodevinDataDir, containerName)

	return NetworkConfig{
		Image:                       "fiftysix/base-reth",
		Version:                     "latest",
		ContainerName:               containerName,
		CommandIsIntentionallyEmpty: true,
		// /app (base-reth's WORKDIR) holds the upstream binaries and
		// entrypoint scripts themselves, not a default config to seed a
		// fresh data directory with.
		SkipInitCopy: true,
		Environment: map[string]string{
			"RETH_CHAIN":                   baseChain(suffix),
			"BASE_NODE_L2_ENGINE_AUTH_RAW": jwtSecret,
		},
		// Loopback-only HTTP/WS RPC (admin-level API, same reasoning as every
		// other chain here); P2P and discv5 stay public so peers can connect.
		// The authrpc port (8551) is deliberately NOT published to the host -
		// only fiftysix/base-consensus, over the shared Docker network below,
		// is meant to reach it. P2P/discv5 host ports differ between mainnet
		// and testnet (unlike the RPC port, both would otherwise collide) and
		// avoid 30303/30304, already used by ethereum-classic/-testnet.
		Ports:    baseExecutionPorts(suffix, utils.NetworkDefaultRPCPorts()[network]),
		Volumes:  []string{fmt.Sprintf("%s:/data", localPath)},
		Networks: []string{baseDockerNetwork(suffix)},
		NetworkDefs: map[string]NetworkDetails{
			baseDockerNetwork(suffix): {Driver: "bridge"},
		},
		VolumeDefs: map[string]VolumeDetails{
			fmt.Sprintf("%s-data", containerName): {
				Labels: map[string]string{
					"nodevin.blockchain.software": containerName,
				},
			},
		},
		LocalPath: localPath,
	}, nil
}

// GetBaseConsensusNetworkComposeConfig builds the compose config for
// fiftysix/base-consensus on "base-consensus" (mainnet) or
// "base-consensus-testnet" (Base Sepolia), paired with the fiftysix/base-reth
// container in the same stack. l1 comes from ResolveL1Endpoints.
func GetBaseConsensusNetworkComposeConfig(network string, jwtSecret string, l1 L1Endpoints) (NetworkConfig, error) {
	nodevinDataDir, err := utils.GetNodevinDataDir()
	if err != nil {
		return NetworkConfig{}, err
	}

	suffix := stackSuffix(network)
	containerName, ok := utils.GetDefaultLocalMappedContainerName(network)
	if !ok {
		return NetworkConfig{}, fmt.Errorf("unknown network: %s", network)
	}

	executionContainerName, ok := utils.GetDefaultLocalMappedContainerName("base" + suffix)
	if !ok {
		return NetworkConfig{}, fmt.Errorf("unknown network: base%s", suffix)
	}

	localPath := filepath.Join(nodevinDataDir, containerName)

	dockerNetworks := []string{baseDockerNetwork(suffix)}
	networkDefs := map[string]NetworkDetails{
		baseDockerNetwork(suffix): {Driver: "bridge"},
	}
	if l1.DockerNetwork != "" {
		dockerNetworks = append(dockerNetworks, l1.DockerNetwork)
		networkDefs[l1.DockerNetwork] = NetworkDetails{Driver: "bridge"}
	}

	return NetworkConfig{
		Image:                       "fiftysix/base-consensus",
		Version:                     "latest",
		ContainerName:               containerName,
		CommandIsIntentionallyEmpty: true,
		// Same reasoning as fiftysix/base-reth above.
		SkipInitCopy: true,
		Environment: map[string]string{
			"BASE_NODE_NETWORK":            baseChain(suffix),
			"BASE_NODE_L1_ETH_RPC":         l1.ExecutionRPCURL,
			"BASE_NODE_L1_BEACON":          l1.BeaconURL,
			"BASE_NODE_L2_ENGINE_RPC":      fmt.Sprintf("http://%s:8551", executionContainerName),
			"BASE_NODE_L2_ENGINE_AUTH_RAW": jwtSecret,
		},
		Ports:       baseConsensusPorts(suffix, utils.NetworkDefaultRPCPorts()[network]),
		Volumes:     []string{fmt.Sprintf("%s:/data", localPath)},
		Networks:    dockerNetworks,
		NetworkDefs: networkDefs,
		VolumeDefs: map[string]VolumeDetails{
			fmt.Sprintf("%s-data", containerName): {
				Labels: map[string]string{
					"nodevin.blockchain.software": containerName,
				},
			},
		},
		LocalPath: localPath,
	}, nil
}
