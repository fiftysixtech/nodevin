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
