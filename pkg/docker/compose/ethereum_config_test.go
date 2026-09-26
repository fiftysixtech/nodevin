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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

var consensusBuilders = map[string]func(string) (NetworkConfig, error){
	"lighthouse": GetLighthouseNetworkComposeConfig,
	"prysm":      GetPrysmNetworkComposeConfig,
	"teku":       GetTekuNetworkComposeConfig,
	"nimbus":     GetNimbusNetworkComposeConfig,
	"lodestar":   GetLodestarNetworkComposeConfig,
}

// Every execution/consensus pairing must wire the consensus client to the
// selected execution client's Engine API, JWT secret and (read-only) data.
func TestEthereumPairings(t *testing.T) {
	for execClient := range supportedExecutionClients {
		for consClient, build := range consensusBuilders {
			t.Run(execClient+"+"+consClient, func(t *testing.T) {
				dir := t.TempDir()
				viper.Set("data-dir", dir)
				viper.Set("execution-client", execClient)
				t.Cleanup(func() {
					viper.Set("data-dir", "")
					viper.Set("execution-client", "")
				})

				execCfg, err := GetEthereumNetworkComposeConfig("ethereum")
				if err != nil {
					t.Fatal(err)
				}
				consCfg, err := build(consClient)
				if err != nil {
					t.Fatal(err)
				}

				if execCfg.ContainerName != execClient || consCfg.ContainerName != consClient {
					t.Errorf("container names = (%s, %s), want (%s, %s)", execCfg.ContainerName, consCfg.ContainerName, execClient, consClient)
				}
				for _, want := range []string{
					fmt.Sprintf("http://%s:8551", execClient),
					fmt.Sprintf("/node/%s/data/jwt.hex", execClient),
				} {
					if !strings.Contains(consCfg.Command, want) {
						t.Errorf("command %q does not contain %q", consCfg.Command, want)
					}
				}

				var readOnly []string
				for _, v := range consCfg.Volumes {
					if strings.HasSuffix(v, fmt.Sprintf(":/node/%s:ro", execClient)) {
						readOnly = append(readOnly, v)
					}
				}
				if len(readOnly) != 1 {
					t.Errorf("want exactly one read-only mount of the %s data dir, got volumes %v", execClient, consCfg.Volumes)
				}
				if !hasWritableMount(consCfg.Volumes, consClient) {
					t.Errorf("consensus client has no writable data volume: %v", consCfg.Volumes)
				}

				path, err := CreateComposeFile(execClient, execCfg, []string{consClient}, []NetworkConfig{consCfg}, dir)
				if err != nil {
					t.Fatal(err)
				}
				raw, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				var file ComposeFile
				if err := yaml.Unmarshal(raw, &file); err != nil {
					t.Fatal(err)
				}
				for _, service := range []string{execClient, consClient} {
					if _, ok := file.Services[service]; !ok {
						t.Errorf("compose file has no %q service", service)
					}
				}
			})
		}
	}
}

func hasWritableMount(volumes []string, client string) bool {
	for _, v := range volumes {
		if strings.HasSuffix(v, ":/node/"+client) {
			return true
		}
	}
	return false
}

func TestEthereumNeverPublishesTheEngineAPI(t *testing.T) {
	viper.Set("data-dir", t.TempDir())
	t.Cleanup(func() { viper.Set("data-dir", "") })

	cfg, err := GetEthereumNetworkComposeConfig("ethereum")
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range cfg.Ports {
		if strings.Contains(p, "8551") {
			t.Errorf("Engine API must stay on the compose network, but %q publishes it", p)
		}
	}
}

func TestResolveCheckpointSyncURL(t *testing.T) {
	cases := []struct {
		name    string
		client  string
		url     string
		want    string
		wantErr string
	}{
		{"none needs no URL", "none", "", "", ""},
		{"none ignores a bad URL", "none", "not a url", "", ""},
		{"required for a real client", "lighthouse", "", "", "--checkpoint-sync-url is required"},
		{"points at the public endpoint list", "teku", "", "", "eth-clients.github.io/checkpoint-sync-endpoints"},
		{"accepts https", "lighthouse", "https://mainnet.checkpoint.sigp.io", "https://mainnet.checkpoint.sigp.io", ""},
		{"trims a trailing slash", "prysm", "https://example.org/", "https://example.org", ""},
		{"accepts http (own node on a LAN)", "lodestar", "http://192.168.1.5:5052", "http://192.168.1.5:5052", ""},
		{"rejects a non-http scheme", "lighthouse", "file:///etc/passwd", "", "invalid --checkpoint-sync-url"},
		{"rejects no host", "lighthouse", "https://", "", "invalid --checkpoint-sync-url"},
		{"rejects whitespace that would inject a second flag", "lighthouse", "https://a.org --allow-insecure-genesis-sync", "", "invalid --checkpoint-sync-url"},
		{"rejects shell metacharacters", "lighthouse", "https://a.org/$(id)", "", "invalid --checkpoint-sync-url"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			viper.Set("checkpoint-sync-url", c.url)
			t.Cleanup(func() { viper.Set("checkpoint-sync-url", "") })

			got, err := ResolveCheckpointSyncURL(c.client)
			if c.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), c.wantErr) {
					t.Fatalf("error = %v, want one containing %q", err, c.wantErr)
				}
				return
			}
			if err != nil || got != c.want {
				t.Errorf("got (%q, %v), want (%q, nil)", got, err, c.want)
			}
		})
	}
}

func TestWithCheckpointSync(t *testing.T) {
	const u = "https://cp.example.org"
	const base = "BASE"

	want := map[string]string{
		"lighthouse": "BASE --checkpoint-sync-url https://cp.example.org",
		"prysm":      "BASE --checkpoint-sync-url https://cp.example.org --genesis-beacon-api-url https://cp.example.org",
		"teku":       "BASE --checkpoint-sync-url https://cp.example.org",
		"lodestar":   "BASE --checkpointSyncUrl https://cp.example.org",
	}
	for client, expected := range want {
		if got, err := WithCheckpointSync(client, "mainnet", u, base); err != nil || got != expected {
			t.Errorf("WithCheckpointSync(%s) = (%q, %v), want %q", client, got, err, expected)
		}
	}
	if _, err := WithCheckpointSync("grandine", "mainnet", u, base); err == nil {
		t.Error("an unknown client must be an error")
	}
}

// Nimbus must sync via trustedNodeSync before starting, only when there is no
// database yet, must fail rather than fall through to a genesis start, and must
// keep the entrypoint's default flags by starting through it.
func TestWithCheckpointSync_Nimbus(t *testing.T) {
	got, err := WithCheckpointSync("nimbus", "mainnet", "https://cp.example.org", "nimbus_beacon_node --el=http://geth:8551 --jwt-secret=/node/geth/data/jwt.hex")
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{
		"/bin/sh -c '",
		"if [ ! -d /node/nimbus/data/db ]; then",
		"trustedNodeSync --network=mainnet --data-dir=/node/nimbus/data --trusted-node-url=https://cp.example.org --backfill=false",
		"|| { rm -rf /node/nimbus/data/db; exit 1; }",
		"exec /node/nimbus/scripts/nimbus-entrypoint.sh nimbus_beacon_node --el=http://geth:8551 --jwt-secret=/node/geth/data/jwt.hex'",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("command %q does not contain %q", got, want)
		}
	}
	if strings.Contains(got, "$") {
		t.Errorf("command %q contains a $, which docker compose would try to interpolate", got)
	}
	if strings.Index(got, "trustedNodeSync") > strings.Index(got, "exec ") {
		t.Error("trustedNodeSync must run before nimbus starts")
	}
}

// The Sepolia stack must stay entirely apart from mainnet: its own containers,
// data directories, volumes and Docker network, wired to each other by the
// testnet names, and every client told to run on Sepolia.
func TestEthereumTestnetPairings(t *testing.T) {
	for execClient := range supportedExecutionClients {
		for consClient, build := range consensusBuilders {
			t.Run(execClient+"+"+consClient, func(t *testing.T) {
				dir := t.TempDir()
				viper.Set("data-dir", dir)
				viper.Set("execution-client", execClient)
				t.Cleanup(func() {
					viper.Set("data-dir", "")
					viper.Set("execution-client", "")
				})

				execCfg, err := GetEthereumNetworkComposeConfig("ethereum-testnet")
				if err != nil {
					t.Fatal(err)
				}
				consCfg, err := build(consClient + "-testnet")
				if err != nil {
					t.Fatal(err)
				}

				execName, consName := execClient+"-testnet", consClient+"-testnet"
				if execCfg.ContainerName != execName || consCfg.ContainerName != consName {
					t.Errorf("container names = (%s, %s), want (%s, %s)", execCfg.ContainerName, consCfg.ContainerName, execName, consName)
				}

				if !strings.Contains(execCfg.Command, "sepolia") {
					t.Errorf("execution command %q does not name sepolia", execCfg.Command)
				}
				if execCfg.CommandIsIntentionallyEmpty {
					t.Error("a Sepolia execution client must not run the image's mainnet default command")
				}
				if !strings.Contains(consCfg.Command, "sepolia") {
					t.Errorf("consensus command %q does not name sepolia", consCfg.Command)
				}
				if strings.Contains(consCfg.Command, "mainnet") {
					t.Errorf("consensus command %q mentions mainnet", consCfg.Command)
				}

				// wired by the testnet container name, JWT read from the image's own path
				for _, want := range []string{
					fmt.Sprintf("http://%s:8551", execName),
					fmt.Sprintf("/node/%s/data/jwt.hex", execClient),
				} {
					if !strings.Contains(consCfg.Command, want) {
						t.Errorf("command %q does not contain %q", consCfg.Command, want)
					}
				}

				// host data lives under testnet-suffixed directories, in the subdirectory the
				// init container fills with the image's own tree, mounted at the image's paths
				execData := filepath.Join(dir, ".nodevin", "data", execName, execClient)
				if want := fmt.Sprintf("%s:/node/%s", execData, execClient); execCfg.Volumes[0] != want {
					t.Errorf("execution volume = %q, want %q", execCfg.Volumes[0], want)
				}
				if want := fmt.Sprintf("%s:/node/%s:ro", execData, execClient); !contains(consCfg.Volumes, want) {
					t.Errorf("consensus volumes %v lack the read-only execution mount %q", consCfg.Volumes, want)
				}
				consData := filepath.Join(dir, ".nodevin", "data", consName, consClient)
				if want := fmt.Sprintf("%s:/node/%s", consData, consClient); !contains(consCfg.Volumes, want) {
					t.Errorf("consensus volumes %v lack its own data mount %q", consCfg.Volumes, want)
				}

				for _, cfg := range []NetworkConfig{execCfg, consCfg} {
					if len(cfg.Networks) != 1 || cfg.Networks[0] != "ethereum-testnet-net" {
						t.Errorf("%s networks = %v, want [ethereum-testnet-net]", cfg.ContainerName, cfg.Networks)
					}
					for name := range cfg.VolumeDefs {
						if !strings.HasSuffix(name, "-testnet-data") {
							t.Errorf("%s volume %q lacks the -testnet suffix", cfg.ContainerName, name)
						}
					}
				}

				path, err := CreateComposeFile(execName, execCfg, []string{consName}, []NetworkConfig{consCfg}, dir)
				if err != nil {
					t.Fatal(err)
				}
				raw, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				var file ComposeFile
				if err := yaml.Unmarshal(raw, &file); err != nil {
					t.Fatal(err)
				}
				for _, service := range []string{execName, consName} {
					if _, ok := file.Services[service]; !ok {
						t.Errorf("compose file has no %q service", service)
					}
				}
				if filepath.Base(path) != "docker-compose_"+execName+".yml" {
					t.Errorf("compose file = %s, want docker-compose_%s.yml", filepath.Base(path), execName)
				}
			})
		}
	}
}

func contains(list []string, want string) bool {
	for _, item := range list {
		if item == want {
			return true
		}
	}
	return false
}

// Mainnet keeps the image's own default command and never names a testnet.
func TestEthereumMainnetIsUnchangedByTestnetSupport(t *testing.T) {
	viper.Set("data-dir", t.TempDir())
	t.Cleanup(func() { viper.Set("data-dir", "") })

	for execClient := range supportedExecutionClients {
		viper.Set("execution-client", execClient)
		cfg, err := GetEthereumNetworkComposeConfig("ethereum")
		if err != nil {
			t.Fatal(err)
		}
		if cfg.Command != "" || !cfg.CommandIsIntentionallyEmpty || cfg.ContainerName != execClient {
			t.Errorf("%s mainnet = command %q, container %q; want the image default", execClient, cfg.Command, cfg.ContainerName)
		}
		if cfg.Networks[0] != "ethereum-net" {
			t.Errorf("%s mainnet network = %v", execClient, cfg.Networks)
		}
	}
	viper.Set("execution-client", "")

	for consClient, build := range consensusBuilders {
		cfg, err := build(consClient)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(cfg.Command, "sepolia") || strings.Contains(cfg.ContainerName, "testnet") {
			t.Errorf("%s mainnet config leaked testnet settings: %q / %q", consClient, cfg.Command, cfg.ContainerName)
		}
	}
}

// Each client's Sepolia command, exactly: these flags were each verified
// against the real published images.
func TestExecutionTestnetCommands(t *testing.T) {
	want := map[string]string{
		"reth":   "--chain sepolia",
		"geth":   "--sepolia",
		"erigon": "--chain=sepolia",
		"besu":   "gosu nodeuser besu --config-file=/node/besu/configs/config.toml --network=sepolia",
	}
	for client, expected := range want {
		if got := executionTestnetCommand(client); got != expected {
			t.Errorf("executionTestnetCommand(%s) = %q, want %q", client, got, expected)
		}
	}

	neth := executionTestnetCommand("nethermind")
	for _, part := range []string{"gosu nodeuser nethermind --config sepolia", "--JsonRpc.Host 0.0.0.0", "--JsonRpc.EngineHost 0.0.0.0", "--JsonRpc.JwtSecretFile /node/nethermind/data/jwt.hex"} {
		if !strings.Contains(neth, part) {
			t.Errorf("nethermind command %q lacks %q", neth, part)
		}
	}
	if strings.Count(neth, "--config") != 1 {
		t.Errorf("nethermind takes a single --config, got %q", neth)
	}
}

func TestPrysmTestnetBypassesTheEntrypointsMainnetFlag(t *testing.T) {
	viper.Set("data-dir", t.TempDir())
	t.Cleanup(func() { viper.Set("data-dir", "") })

	cfg, err := GetPrysmNetworkComposeConfig("prysm-testnet")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(cfg.Command, "gosu nodeuser beacon-chain ") || !strings.Contains(cfg.Command, "--sepolia") || strings.Contains(cfg.Command, "--mainnet") {
		t.Errorf("prysm testnet command = %q, want the entrypoint bypassed with --sepolia only", cfg.Command)
	}
	// still takes the checkpoint flags appended later
	full, err := WithCheckpointSync("prysm", "sepolia", "https://cp.example.org", cfg.Command)
	if err != nil || !strings.Contains(full, "--checkpoint-sync-url https://cp.example.org") {
		t.Errorf("WithCheckpointSync on the testnet command = (%q, %v)", full, err)
	}
}

func TestNimbusTestnetSyncsFromTheSepoliaCheckpoint(t *testing.T) {
	got, err := WithCheckpointSync("nimbus", "sepolia", "https://cp.example.org", "nimbus_beacon_node --el=http://geth-testnet:8551 --jwt-secret=/x --network=sepolia")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "trustedNodeSync --network=sepolia ") || strings.Contains(got, "--network=mainnet") {
		t.Errorf("command %q must run trustedNodeSync on sepolia, never mainnet", got)
	}
}
