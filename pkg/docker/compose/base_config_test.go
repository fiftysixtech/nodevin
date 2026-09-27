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
	"testing"

	"github.com/spf13/viper"
)

func TestBaseJWTSecret_GeneratesOnceAndPersists(t *testing.T) {
	viper.Set("data-dir", t.TempDir())
	t.Cleanup(func() { viper.Set("data-dir", "") })

	first, err := BaseJWTSecret("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(first) != 64 {
		t.Errorf("secret length = %d, want 64 hex characters", len(first))
	}

	second, err := BaseJWTSecret("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if first != second {
		t.Errorf("BaseJWTSecret regenerated a new secret on the second call: %q != %q", first, second)
	}
}

func TestBaseJWTSecret_MainnetAndTestnetAreIndependent(t *testing.T) {
	viper.Set("data-dir", t.TempDir())
	t.Cleanup(func() { viper.Set("data-dir", "") })

	mainnet, err := BaseJWTSecret("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	testnet, err := BaseJWTSecret("-testnet")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mainnet == testnet {
		t.Error("mainnet and testnet got the same JWT secret; they must be independent stacks")
	}
}

func TestGetBaseExecutionNetworkComposeConfig(t *testing.T) {
	viper.Set("data-dir", t.TempDir())
	t.Cleanup(func() { viper.Set("data-dir", "") })

	cfg, err := GetBaseExecutionNetworkComposeConfig("base", "abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ContainerName != "base-reth" {
		t.Errorf("ContainerName = %q, want base-reth", cfg.ContainerName)
	}
	if !cfg.CommandIsIntentionallyEmpty || cfg.Command != "" {
		t.Errorf("Command = %q, CommandIsIntentionallyEmpty = %v; base-reth's entrypoint takes no CLI flags at all", cfg.Command, cfg.CommandIsIntentionallyEmpty)
	}
	if cfg.Environment["RETH_CHAIN"] != "base" {
		t.Errorf("RETH_CHAIN = %q, want \"base\"", cfg.Environment["RETH_CHAIN"])
	}
	if cfg.Environment["BASE_NODE_L2_ENGINE_AUTH_RAW"] != "abc123" {
		t.Errorf("BASE_NODE_L2_ENGINE_AUTH_RAW = %q, want the given secret", cfg.Environment["BASE_NODE_L2_ENGINE_AUTH_RAW"])
	}

	testnetCfg, err := GetBaseExecutionNetworkComposeConfig("base-testnet", "abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if testnetCfg.Environment["RETH_CHAIN"] != "base-sepolia" {
		t.Errorf("testnet RETH_CHAIN = %q, want \"base-sepolia\"", testnetCfg.Environment["RETH_CHAIN"])
	}
}

func TestGetBaseConsensusNetworkComposeConfig(t *testing.T) {
	viper.Set("data-dir", t.TempDir())
	t.Cleanup(func() { viper.Set("data-dir", "") })

	l1 := L1Endpoints{ExecutionRPCURL: "http://reth:8545", BeaconURL: "http://lighthouse:5052", DockerNetwork: "ethereum-net"}
	cfg, err := GetBaseConsensusNetworkComposeConfig("base-consensus", "abc123", l1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ContainerName != "base-consensus" {
		t.Errorf("ContainerName = %q, want base-consensus", cfg.ContainerName)
	}
	if cfg.Environment["BASE_NODE_L2_ENGINE_RPC"] != "http://base-reth:8551" {
		t.Errorf("BASE_NODE_L2_ENGINE_RPC = %q, want http://base-reth:8551", cfg.Environment["BASE_NODE_L2_ENGINE_RPC"])
	}
	if cfg.Environment["BASE_NODE_L1_ETH_RPC"] != l1.ExecutionRPCURL {
		t.Errorf("BASE_NODE_L1_ETH_RPC = %q, want %q", cfg.Environment["BASE_NODE_L1_ETH_RPC"], l1.ExecutionRPCURL)
	}
	if cfg.Environment["BASE_NODE_L2_ENGINE_AUTH_RAW"] != "abc123" {
		t.Errorf("BASE_NODE_L2_ENGINE_AUTH_RAW = %q, want the same secret given to base-reth", cfg.Environment["BASE_NODE_L2_ENGINE_AUTH_RAW"])
	}

	found := false
	for _, n := range cfg.Networks {
		if n == "ethereum-net" {
			found = true
		}
	}
	if !found {
		t.Errorf("Networks = %v, want it to include the auto-attached L1's Docker network", cfg.Networks)
	}

	testnetCfg, err := GetBaseConsensusNetworkComposeConfig("base-consensus-testnet", "abc123", L1Endpoints{ExecutionRPCURL: "http://x", BeaconURL: "http://y"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if testnetCfg.Environment["BASE_NODE_L2_ENGINE_RPC"] != "http://base-reth-testnet:8551" {
		t.Errorf("testnet BASE_NODE_L2_ENGINE_RPC = %q, want it to point at base-reth-testnet", testnetCfg.Environment["BASE_NODE_L2_ENGINE_RPC"])
	}
	if testnetCfg.Environment["BASE_NODE_NETWORK"] != "base-sepolia" {
		t.Errorf("testnet BASE_NODE_NETWORK = %q, want base-sepolia", testnetCfg.Environment["BASE_NODE_NETWORK"])
	}
}

func TestBasePorts_MainnetAndTestnetDoNotCollide(t *testing.T) {
	viper.Set("data-dir", t.TempDir())
	t.Cleanup(func() { viper.Set("data-dir", "") })

	mainnetExec, _ := GetBaseExecutionNetworkComposeConfig("base", "abc")
	testnetExec, _ := GetBaseExecutionNetworkComposeConfig("base-testnet", "abc")
	mainnetCons, _ := GetBaseConsensusNetworkComposeConfig("base-consensus", "abc", L1Endpoints{ExecutionRPCURL: "http://x", BeaconURL: "http://y"})
	testnetCons, _ := GetBaseConsensusNetworkComposeConfig("base-consensus-testnet", "abc", L1Endpoints{ExecutionRPCURL: "http://x", BeaconURL: "http://y"})

	seen := map[string]string{}
	for label, cfg := range map[string]NetworkConfig{
		"base-exec": mainnetExec, "base-testnet-exec": testnetExec,
		"base-cons": mainnetCons, "base-testnet-cons": testnetCons,
	} {
		for _, p := range cfg.Ports {
			if owner, ok := seen[p]; ok && owner != label {
				t.Errorf("port mapping %q published by both %s and %s", p, owner, label)
			}
			seen[p] = label
		}
	}
}
