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

func TestGetArbitrumNetworkComposeConfig_ChainIDs(t *testing.T) {
	viper.Set("data-dir", t.TempDir())
	t.Cleanup(func() { viper.Set("data-dir", "") })

	l1 := L1Endpoints{ExecutionRPCURL: "http://reth:8545", BeaconURL: "http://lighthouse:5052"}

	cases := []struct {
		network           string
		wantChainID       string
		wantParentChainID string
	}{
		{"arbitrum", "42161", "1"},
		{"arbitrum-testnet", "421614", "11155111"},
	}

	for _, c := range cases {
		t.Run(c.network, func(t *testing.T) {
			cfg, err := GetArbitrumNetworkComposeConfig(c.network, l1)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !strings.Contains(cfg.Command, "--chain.id "+c.wantChainID) {
				t.Errorf("Command = %q, want it to contain --chain.id %s", cfg.Command, c.wantChainID)
			}
			if !strings.Contains(cfg.Command, "--parent-chain.id "+c.wantParentChainID) {
				t.Errorf("Command = %q, want it to contain --parent-chain.id %s", cfg.Command, c.wantParentChainID)
			}
			if !strings.Contains(cfg.Command, "--parent-chain.connection.url "+l1.ExecutionRPCURL) {
				t.Errorf("Command = %q, want it to contain the L1 execution RPC URL", cfg.Command)
			}
			if !strings.Contains(cfg.Command, "--parent-chain.blob-client.beacon-url "+l1.BeaconURL) {
				t.Errorf("Command = %q, want it to contain the L1 beacon URL", cfg.Command)
			}
		})
	}
}

func TestGetArbitrumNetworkComposeConfig_AutoAttachJoinsL1Network(t *testing.T) {
	viper.Set("data-dir", t.TempDir())
	t.Cleanup(func() { viper.Set("data-dir", "") })

	l1 := L1Endpoints{ExecutionRPCURL: "http://reth:8545", BeaconURL: "http://lighthouse:5052", DockerNetwork: "ethereum-net"}

	cfg, err := GetArbitrumNetworkComposeConfig("arbitrum", l1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
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
	if _, ok := cfg.NetworkDefs["ethereum-net"]; !ok {
		t.Errorf("NetworkDefs = %v, want an entry for the auto-attached L1's Docker network", cfg.NetworkDefs)
	}
}

func TestGetArbitrumNetworkComposeConfig_ExternalL1NoExtraNetwork(t *testing.T) {
	viper.Set("data-dir", t.TempDir())
	t.Cleanup(func() { viper.Set("data-dir", "") })

	l1 := L1Endpoints{ExecutionRPCURL: "https://l1.example.com/rpc", BeaconURL: "https://l1.example.com/beacon"}

	cfg, err := GetArbitrumNetworkComposeConfig("arbitrum", l1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Networks) != 1 {
		t.Errorf("Networks = %v, want exactly nitro's own network for an external L1", cfg.Networks)
	}
}

func TestGetArbitrumNetworkComposeConfig_PublishesSequencerFeed(t *testing.T) {
	viper.Set("data-dir", t.TempDir())
	t.Cleanup(func() { viper.Set("data-dir", "") })

	l1 := L1Endpoints{ExecutionRPCURL: "http://reth:8545", BeaconURL: "http://lighthouse:5052"}

	mainnet, err := GetArbitrumNetworkComposeConfig("arbitrum", l1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	testnet, err := GetArbitrumNetworkComposeConfig("arbitrum-testnet", l1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(strings.Join(mainnet.Ports, ","), ":9642") {
		t.Errorf("mainnet Ports = %v, want the 9642 sequencer feed published", mainnet.Ports)
	}
	if mainnet.Ports[len(mainnet.Ports)-1] == testnet.Ports[len(testnet.Ports)-1] {
		t.Errorf("mainnet and testnet both publish sequencer feed port %q; they must differ so both can run at once", mainnet.Ports[len(mainnet.Ports)-1])
	}
}
