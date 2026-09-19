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
	"os"
	"testing"

	"github.com/spf13/viper"
)

// TestGenerateComposeFilesForCI writes the docker-compose files nodevin
// really generates for every registered network, plus the companion
// combinations (`start bitcoin --ord`, `start litecoin --ord-litecoin`,
// `start ipfs --ipfs-cluster`), into $NODEVIN_CI_COMPOSE_DIR. The CI workflow
// then runs `docker compose config` on each one and starts `ipfs` for real.
// It is skipped unless that variable is set, so it never runs in a normal
// `go test ./...`.
//
// Files land in $NODEVIN_CI_COMPOSE_DIR/.nodevin/data/, exactly where the CLI
// writes them for a given --data-dir.
func TestGenerateComposeFilesForCI(t *testing.T) {
	outDir := os.Getenv("NODEVIN_CI_COMPOSE_DIR")
	if outDir == "" {
		t.Skip("NODEVIN_CI_COMPOSE_DIR not set; this test only generates files for the CI workflow")
	}

	viper.Set("data-dir", outDir)
	t.Cleanup(func() { viper.Set("data-dir", "") })

	for _, row := range networkBuilders {
		cfg, err := row.builder(row.network)
		if err != nil {
			t.Fatalf("builder(%q): %v", row.network, err)
		}
		if _, err := CreateComposeFile(cfg.ContainerName, cfg, nil, nil, outDir); err != nil {
			t.Fatalf("CreateComposeFile(%q): %v", cfg.ContainerName, err)
		}
	}

	combos := []struct {
		main      func(string) (NetworkConfig, error)
		mainNet   string
		extra     func(string) (NetworkConfig, error)
		extraNet  string
		extraName string
	}{
		{GetBitcoinNetworkComposeConfig, "bitcoin", GetOrdNetworkComposeConfig, "ord", "ord"},
		{GetLitecoinNetworkComposeConfig, "litecoin", GetOrdLitecoinNetworkComposeConfig, "ord-litecoin", "ord-litecoin"},
		{GetKuboNetworkComposeConfig, "ipfs", GetIpfsClusterNetworkComposeConfig, "ipfs-cluster", "ipfs-cluster"},
	}
	for _, c := range combos {
		mainCfg, err := c.main(c.mainNet)
		if err != nil {
			t.Fatal(err)
		}
		extraCfg, err := c.extra(c.extraNet)
		if err != nil {
			t.Fatal(err)
		}
		// Same naming the CLI uses for combined files: the main node's name,
		// which the combined file overwrites. Keep both variants on disk by
		// generating the combination into a sibling directory.
		comboDir := outDir + "/combo-" + c.mainNet + "-" + c.extraName
		viper.Set("data-dir", comboDir)
		if _, err := CreateComposeFile(mainCfg.ContainerName, mainCfg, []string{c.extraName}, []NetworkConfig{extraCfg}, comboDir); err != nil {
			t.Fatalf("combo %s+%s: %v", c.mainNet, c.extraName, err)
		}
	}
}
