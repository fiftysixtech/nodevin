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
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func TestGetBscNetworkComposeConfig(t *testing.T) {
	viper.Set("data-dir", t.TempDir())
	t.Cleanup(func() { viper.Set("data-dir", "") })

	cfg, err := GetBscNetworkComposeConfig("bsc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ContainerName != "bsc" {
		t.Errorf("ContainerName = %q, want bsc", cfg.ContainerName)
	}
	if !cfg.CommandIsIntentionallyEmpty || cfg.Command != "" {
		t.Errorf("Command = %q, CommandIsIntentionallyEmpty = %v; bsc selects its network via NETWORK env var, not a CLI flag", cfg.Command, cfg.CommandIsIntentionallyEmpty)
	}
	if cfg.Environment["NETWORK"] != "mainnet" {
		t.Errorf("NETWORK = %q, want mainnet", cfg.Environment["NETWORK"])
	}
	if !cfg.SkipInitCopy {
		t.Error("SkipInitCopy = false; bsc's entrypoint/config live outside the mounted /node/bsc/data, so nothing needs seeding")
	}
	if len(cfg.Volumes) != 1 || !strings.HasSuffix(cfg.Volumes[0], ":/node/bsc/data") {
		t.Errorf("Volumes = %v, want a single mount at /node/bsc/data (not the whole /node/bsc)", cfg.Volumes)
	}

	testnetCfg, err := GetBscNetworkComposeConfig("bsc-testnet")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if testnetCfg.ContainerName != "bsc-testnet" {
		t.Errorf("ContainerName = %q, want bsc-testnet", testnetCfg.ContainerName)
	}
	if testnetCfg.Environment["NETWORK"] != "testnet" {
		t.Errorf("testnet NETWORK = %q, want testnet", testnetCfg.Environment["NETWORK"])
	}
}

func TestBscPorts_MainnetAndTestnetDoNotCollide(t *testing.T) {
	viper.Set("data-dir", t.TempDir())
	t.Cleanup(func() { viper.Set("data-dir", "") })

	mainnet, err := GetBscNetworkComposeConfig("bsc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	testnet, err := GetBscNetworkComposeConfig("bsc-testnet")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	seen := map[string]bool{}
	for _, p := range mainnet.Ports {
		seen[p] = true
	}
	for _, p := range testnet.Ports {
		if seen[p] {
			t.Errorf("port mapping %q published by both mainnet and testnet", p)
		}
	}
}
