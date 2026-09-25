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
		if got, err := WithCheckpointSync(client, u, base); err != nil || got != expected {
			t.Errorf("WithCheckpointSync(%s) = (%q, %v), want %q", client, got, err, expected)
		}
	}
	if _, err := WithCheckpointSync("grandine", u, base); err == nil {
		t.Error("an unknown client must be an error")
	}
}

// Nimbus must sync via trustedNodeSync before starting, only when there is no
// database yet, must fail rather than fall through to a genesis start, and must
// keep the entrypoint's default flags by starting through it.
func TestWithCheckpointSync_Nimbus(t *testing.T) {
	got, err := WithCheckpointSync("nimbus", "https://cp.example.org", "nimbus_beacon_node --el=http://geth:8551 --jwt-secret=/node/geth/data/jwt.hex")
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
