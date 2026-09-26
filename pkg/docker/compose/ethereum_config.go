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
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/fiftysixcrypto/nodevin/internal/utils"
	"github.com/spf13/viper"
)

// supportedExecutionClients are the Ethereum execution clients this repo
// builds and publishes (see fiftysixtech/node-images, ethereum/execution-clients).
// Every one of them shares the same on-image layout by convention (ROOT_DIR
// /node/<client>, JWT secret at <ROOT_DIR>/data/jwt.hex, JSON-RPC on 8545,
// WS on 8546, Engine API on 8551, P2P on 30303) - see
// node-images' docs/ethereum-execution-consensus-pairing.md for the exact
// per-client verification this relies on.
var supportedExecutionClients = map[string]bool{
	"reth":       true,
	"geth":       true,
	"erigon":     true,
	"besu":       true,
	"nethermind": true,
}

// supportedConsensusClients are the Ethereum consensus (beacon) clients this
// repo builds and publishes (see fiftysixtech/node-images,
// ethereum/consensus-clients). "none" is not a real client - it is the
// explicit opt-out for running the execution client standalone.
var supportedConsensusClients = map[string]bool{
	"lighthouse": true,
	"prysm":      true,
	"teku":       true,
	"nimbus":     true,
	"lodestar":   true,
}

// SelectedExecutionClient resolves --execution-client, defaulting to reth
// (the broadest-compatibility pick per the pairing doc: no glibc issues,
// GPG-verified release, and the client Base's own image is built on).
func SelectedExecutionClient() (string, error) {
	client := viper.GetString("execution-client")
	if client == "" {
		client = "reth"
	}
	if !supportedExecutionClients[client] {
		return "", fmt.Errorf("unsupported --execution-client: %s (supported: reth, geth, erigon, besu, nethermind)", client)
	}
	return client, nil
}

// SelectedConsensusClient resolves --consensus-client, defaulting to
// lighthouse (currently the largest share of Ethereum mainnet consensus
// traffic). Unlike --ord/--ipfs-cluster, this is deliberately on by default:
// an execution client cannot sync at all post-Merge without a paired
// consensus client driving it over the Engine API, so leaving this off by
// default would produce a node that looks like it started but never does
// anything. Pass --consensus-client=none to run execution-only.
//
// Exported since pkg/nodes/ethereum needs it too, to decide whether to wire
// up the extra consensus-client service and which of the five builder
// functions below to call.
func SelectedConsensusClient() (string, error) {
	client := viper.GetString("consensus-client")
	if client == "" {
		client = "lighthouse"
	}
	if client == "none" {
		return "none", nil
	}
	if !supportedConsensusClients[client] {
		return "", fmt.Errorf("unsupported --consensus-client: %s (supported: lighthouse, prysm, teku, nimbus, lodestar, none)", client)
	}
	return client, nil
}

// ResolveCheckpointSyncURL returns the validated --checkpoint-sync-url for a
// consensus client. It is required for every consensus client: Lighthouse and
// Teku refuse to sync from genesis at all, and genesis sync is impractically
// slow and unsafe for the rest. nodevin deliberately has no default
// endpoint - which third party to trust for checkpoint data is the user's call.
// Returns "" for consensusClient "none".
func ResolveCheckpointSyncURL(consensusClient string) (string, error) {
	if consensusClient == "none" {
		return "", nil
	}

	raw := strings.TrimSpace(viper.GetString("checkpoint-sync-url"))
	if raw == "" {
		return "", fmt.Errorf("--checkpoint-sync-url is required to run %s: a consensus client cannot practically sync from genesis (Lighthouse and Teku refuse to try). "+
			"Pass the URL of a checkpoint sync provider you trust (public list: https://eth-clients.github.io/checkpoint-sync-endpoints/), or use --consensus-client=none to run the execution client alone", consensusClient)
	}

	parsed, err := url.Parse(raw)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || strings.ContainsAny(raw, " \t\r\n\"'`$\\;&|<>") {
		return "", fmt.Errorf("invalid --checkpoint-sync-url %q: expected an http(s) URL such as https://mainnet.checkpoint.sigp.io", raw)
	}
	return strings.TrimRight(raw, "/"), nil
}

// WithCheckpointSync returns command (a consensus client's base command) set up
// to start from the checkpoint provider at checkpointURL.
//
// Every client but Nimbus just takes a flag. Nimbus's own checkpoint flags
// (--external-beacon-api-url with --trusted-block-root) rely on the provider
// serving light-client data, which public providers often don't - Nimbus then
// silently falls back to syncing from genesis. Its supported route is the
// separate `trustedNodeSync` subcommand, run once against an empty database, so
// for Nimbus the command becomes a small wrapper that runs it first. It never
// deletes anything that was there before: it only runs when there is no
// database yet, and only cleans up the partial one its own failed run created.
func WithCheckpointSync(consensusClient, chain, checkpointURL, command string) (string, error) {
	switch consensusClient {
	case "lighthouse", "teku":
		return command + " --checkpoint-sync-url " + checkpointURL, nil
	case "prysm":
		return fmt.Sprintf("%s --checkpoint-sync-url %s --genesis-beacon-api-url %s", command, checkpointURL, checkpointURL), nil
	case "lodestar":
		return command + " --checkpointSyncUrl " + checkpointURL, nil
	case "nimbus":
		const db = "/node/nimbus/data/db"
		script := fmt.Sprintf("if [ ! -d %[1]s ]; then "+
			"gosu nodeuser /usr/bin/nimbus_beacon_node trustedNodeSync --network=%[4]s --data-dir=/node/nimbus/data --trusted-node-url=%[2]s --backfill=false "+
			"|| { rm -rf %[1]s; exit 1; }; fi; "+
			"exec /node/nimbus/scripts/nimbus-entrypoint.sh %[3]s", db, checkpointURL, command, chain)
		return "/bin/sh -c '" + script + "'", nil
	default:
		return "", fmt.Errorf("unsupported --consensus-client: %s", consensusClient)
	}
}

// ethereumTestnetChain is the one Ethereum testnet nodevin runs.
const ethereumTestnetChain = "sepolia"

// stackSuffix is "-testnet" when network names the Sepolia stack
// ("ethereum-testnet", "lighthouse-testnet", ...), "" for mainnet. Every
// container, data directory, volume and Docker network of that stack carries
// it, so a testnet stack never touches mainnet data.
func stackSuffix(network string) string {
	if strings.HasSuffix(network, "-testnet") {
		return "-testnet"
	}
	return ""
}

// EthereumChain returns the chain ("mainnet" or "sepolia") the Ethereum stack
// named by network runs on.
func EthereumChain(network string) string {
	return ethereumChainName(stackSuffix(network))
}

func ethereumChainName(suffix string) string {
	if suffix != "" {
		return ethereumTestnetChain
	}
	return "mainnet"
}

func ethereumDockerNetwork(suffix string) string {
	if suffix != "" {
		return "ethereum-testnet-net"
	}
	return "ethereum-net"
}

// executionClientMountVolume returns the docker-compose volume entry that
// mounts the selected execution client's entire host data directory,
// read-only, into a consensus client's container at the identical absolute
// path the execution client itself uses (matching this repo's existing
// convention: ord_config.go mounts bitcoin-core's data at /node/bitcoin-core
// inside ord's own container too, rather than translating the path). The
// execution client's jwt.hex is then always reachable, from the consensus
// container's side, at /node/<executionClient>/data/jwt.hex. On the testnet
// stack the host directory carries the "-testnet" suffix; the path inside the
// container is the image's own and does not.
func executionClientMountVolume(executionClient, suffix string) (string, error) {
	nodevinDataDir, err := utils.GetNodevinDataDir()
	if err != nil {
		return "", err
	}

	// The init container copies the image's /node tree into the stack's
	// directory, so the client's files sit in a subdirectory named after the
	// image ("reth"), not after the suffixed container.
	localPath := filepath.Join(nodevinDataDir, executionClient+suffix)
	localChainDataPath := filepath.Join(localPath, executionClient)

	return fmt.Sprintf("%s:/node/%s:ro", localChainDataPath, executionClient), nil
}

// executionEngineEndpoint returns the Engine API URL a paired consensus
// client should use to reach the selected execution client, relying on
// Docker Compose's service-name DNS resolution over the stack's shared
// network (the execution client's container name, e.g. "reth" or
// "reth-testnet").
func executionEngineEndpoint(executionClient, suffix string) string {
	return fmt.Sprintf("http://%s%s:8551", executionClient, suffix)
}

// executionJWTPath returns where a paired consensus client finds the shared
// JWT secret once executionClientMountVolume's mount is in place.
func executionJWTPath(executionClient string) string {
	return fmt.Sprintf("/node/%s/data/jwt.hex", executionClient)
}

// executionTestnetCommand returns the container command that starts an
// execution client on Sepolia; mainnet uses each image's own default command.
// Where the image's entrypoint takes flags it is used as designed; two clients
// need the entrypoint bypassed, by starting the binary through gosu (which the
// entrypoint execs unchanged when the first word is not the client):
//   - nethermind takes a single --config, and the entrypoint injects its own
//     mainnet-oriented one before ours ("expects a single argument but 2 were
//     provided"), so Sepolia's config is named directly and the RPC settings
//     the injected config would have set are passed as options.
//   - besu's published image still ships an entrypoint that only injects its
//     config when started with no arguments and cannot take flags at all.
func executionTestnetCommand(executionClient string) string {
	switch executionClient {
	case "reth":
		return "--chain " + ethereumTestnetChain
	case "geth":
		return "--" + ethereumTestnetChain
	case "erigon":
		return "--chain=" + ethereumTestnetChain
	case "besu":
		return "gosu nodeuser besu --config-file=/node/besu/configs/config.toml --network=" + ethereumTestnetChain
	case "nethermind":
		return "gosu nodeuser nethermind --config " + ethereumTestnetChain +
			" --datadir /node/nethermind/data --JsonRpc.JwtSecretFile /node/nethermind/data/jwt.hex" +
			" --JsonRpc.Enabled true --JsonRpc.Host 0.0.0.0 --JsonRpc.EngineHost 0.0.0.0 --JsonRpc.EnginePort 8551" +
			" --HealthChecks.Enabled true --Merge.Enabled true"
	default:
		return ""
	}
}

// GetEthereumNetworkComposeConfig builds the execution client's compose config
// for "ethereum" (mainnet) or "ethereum-testnet" (Sepolia).
func GetEthereumNetworkComposeConfig(network string) (NetworkConfig, error) {
	executionClient, err := SelectedExecutionClient()
	if err != nil {
		return NetworkConfig{}, err
	}

	// Get base nodevin data directory
	nodevinDataDir, err := utils.GetNodevinDataDir()
	if err != nil {
		return NetworkConfig{}, err
	}

	suffix := stackSuffix(network)
	name := executionClient + suffix
	localPath := filepath.Join(nodevinDataDir, name)                // nodevin data dir, software type
	localChainDataPath := filepath.Join(localPath, executionClient) // on-image data dir (see executionClientMountVolume)

	// JSON-RPC/WS are admin-level APIs, published loopback-only (same
	// reasoning as every other chain in this repo). The Engine API (8551) is
	// deliberately NOT published to the host at all - it is only ever meant to
	// be reached by a paired consensus client over the shared Docker network,
	// never from outside the host. Ports 8545/30303 are already used by
	// ethereum-classic, and 8546/30304 by ethereum-classic-testnet, so
	// Ethereum uses the next free ports instead - override with --ports if you
	// need the canonical ports specifically. The Sepolia stack gets its own.
	ports := []string{"127.0.0.1:8547:8545", "127.0.0.1:8548:8546", "30305:30303", "30305:30303/udp"}
	if suffix != "" {
		ports = []string{"127.0.0.1:8549:8545", "127.0.0.1:8550:8546", "30306:30303", "30306:30303/udp"}
	}

	// Every execution client image in this repo already brings up
	// JSON-RPC/WS/Engine API on 0.0.0.0 with sensible mainnet defaults out of
	// the box (confirmed for all five by actually running each with no
	// arguments), so mainnet needs no command - matching kubo/ipfs-cluster's
	// existing CommandIsIntentionallyEmpty pattern. Sepolia only has to name
	// the chain.
	command := ""
	if suffix != "" {
		command = executionTestnetCommand(executionClient)
	}

	baseConfig := NetworkConfig{
		Image:                       fmt.Sprintf("fiftysix/%s", executionClient),
		Version:                     "latest",
		ContainerName:               name,
		Command:                     command,
		CommandIsIntentionallyEmpty: suffix == "",
		Ports:                       ports,
		Volumes:                     []string{fmt.Sprintf("%s:/node/%s", localChainDataPath, executionClient)},
		Networks:                    []string{ethereumDockerNetwork(suffix)},
		NetworkDefs: map[string]NetworkDetails{
			ethereumDockerNetwork(suffix): {
				Driver: "bridge",
			},
		},
		VolumeDefs: map[string]VolumeDetails{
			fmt.Sprintf("%s-data", name): {
				Labels: map[string]string{
					"nodevin.blockchain.software": name,
				},
			},
		},
		LocalPath: localPath,
	}

	return baseConfig, nil
}

// consensusConfig assembles what every consensus client shares: its image,
// its own writable data directory, a read-only view of the paired execution
// client's data (for the JWT secret), and the stack's Docker network. Each
// client's file supplies only what genuinely differs - its ports and its
// command, given the Engine API endpoint, JWT path and chain to run on.
func consensusConfig(client, network string, ports []string, command func(engineEndpoint, jwtPath, chain string) string) (NetworkConfig, error) {
	executionClient, err := SelectedExecutionClient()
	if err != nil {
		return NetworkConfig{}, err
	}

	suffix := stackSuffix(network)
	execMountVolume, err := executionClientMountVolume(executionClient, suffix)
	if err != nil {
		return NetworkConfig{}, err
	}

	nodevinDataDir, err := utils.GetNodevinDataDir()
	if err != nil {
		return NetworkConfig{}, err
	}

	name := client + suffix
	localPath := filepath.Join(nodevinDataDir, name)
	localChainDataPath := filepath.Join(localPath, client) // on-image data dir (see executionClientMountVolume)

	return NetworkConfig{
		Image:         "fiftysix/" + client,
		Version:       "latest",
		ContainerName: name,
		Command: command(
			executionEngineEndpoint(executionClient, suffix),
			executionJWTPath(executionClient),
			ethereumChainName(suffix),
		),
		Ports: ports,
		Volumes: []string{
			fmt.Sprintf("%s:/node/%s", localChainDataPath, client),
			execMountVolume,
		},
		Networks: []string{ethereumDockerNetwork(suffix)},
		NetworkDefs: map[string]NetworkDetails{
			ethereumDockerNetwork(suffix): {
				Driver: "bridge",
			},
		},
		VolumeDefs: map[string]VolumeDetails{
			name + "-data": {
				Labels: map[string]string{
					"nodevin.blockchain.software": name,
				},
			},
		},
		LocalPath: localPath,
	}, nil
}
