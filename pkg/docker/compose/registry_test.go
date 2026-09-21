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
	"strings"
	"testing"

	"github.com/fiftysixcrypto/nodevin/internal/utils"
	"github.com/spf13/viper"
)

// networkBuilders is the hand-maintained mapping from every registered
// network name (internal/utils.networkInfoMap) to the builder function that
// handles it. There is no naming convention linking a network key to its
// builder, so this table must be updated by hand whenever a network is
// added or removed — TestRegistryTableCoversAllNetworks exists specifically
// to catch the case where someone forgets to.
var networkBuilders = []struct {
	network string
	builder func(string) (NetworkConfig, error)
}{
	{"bitcoin", GetBitcoinNetworkComposeConfig},
	{"bitcoin-testnet", GetBitcoinNetworkComposeConfig},
	{"dogecoin", GetDogecoinNetworkComposeConfig},
	{"dogecoin-testnet", GetDogecoinNetworkComposeConfig},
	{"litecoin", GetLitecoinNetworkComposeConfig},
	{"litecoin-testnet", GetLitecoinNetworkComposeConfig},
	{"ethereum-classic", GetEthereumClassicNetworkComposeConfig},
	{"ethereum-classic-testnet", GetEthereumClassicNetworkComposeConfig},
	{"ipfs", GetKuboNetworkComposeConfig},
	{"ipfs-cluster", GetIpfsClusterNetworkComposeConfig},
	{"ord", GetOrdNetworkComposeConfig},
	{"ord-testnet", GetOrdNetworkComposeConfig},
	{"ord-litecoin", GetOrdLitecoinNetworkComposeConfig},
	{"ord-litecoin-testnet", GetOrdLitecoinNetworkComposeConfig},
}

// TestRegistryConsistency asserts, for every registered network, that its
// builder (a) succeeds, (b) never returns an empty Command unless it's
// explicitly marked intentional, and (c) returns the same ContainerName that
// internal/utils.networkInfoMap has on file for it. This is the test that
// would have caught the Ethereum Classic bug directly: its builder shipped
// with an empty Command and Dogecoin's ports, copy-pasted and never wired up.
func TestRegistryConsistency(t *testing.T) {
	for _, row := range networkBuilders {
		t.Run(row.network, func(t *testing.T) {
			viper.Set("data-dir", t.TempDir())
			t.Cleanup(func() { viper.Set("data-dir", "") })

			cfg, err := row.builder(row.network)
			if err != nil {
				t.Fatalf("builder(%q) returned error: %v", row.network, err)
			}

			if cfg.Command == "" && !cfg.CommandIsIntentionallyEmpty {
				t.Errorf("builder(%q) returned an empty Command without setting CommandIsIntentionallyEmpty", row.network)
			}

			wantContainerName, exists := utils.GetDefaultLocalMappedContainerName(row.network)
			if !exists {
				t.Fatalf("network %q has a builder in this test's table but no entry in internal/utils.networkInfoMap", row.network)
			}
			if cfg.ContainerName != wantContainerName {
				t.Errorf("builder(%q).ContainerName = %q, want %q (per networkInfoMap)", row.network, cfg.ContainerName, wantContainerName)
			}
		})
	}
}

// TestRegistryTableCoversAllNetworks asserts that every network registered in
// internal/utils.networkInfoMap has exactly one row in networkBuilders above.
// This is what actually generalizes the protection: it's what catches "a new
// network was registered but nobody added a builder-table row" automatically,
// instead of relying on a human to remember to update this file too.
func TestRegistryTableCoversAllNetworks(t *testing.T) {
	allNetworks := strings.Split(utils.GetAllSupportedNetworks(), ", ")

	covered := make(map[string]int)
	for _, row := range networkBuilders {
		covered[row.network]++
	}

	for _, network := range allNetworks {
		switch covered[network] {
		case 0:
			t.Errorf("network %q is registered in networkInfoMap but has no row in networkBuilders (this test file)", network)
		case 1:
			// expected
		default:
			t.Errorf("network %q has %d rows in networkBuilders, want exactly 1", network, covered[network])
		}
	}

	if len(allNetworks) != len(networkBuilders) {
		t.Errorf("networkInfoMap has %d networks but networkBuilders has %d rows — table is out of sync", len(allNetworks), len(networkBuilders))
	}
}

func hostPorts(t *testing.T, cfg NetworkConfig) []string {
	t.Helper()
	var out []string
	for _, p := range cfg.Ports {
		spec, proto := p, "tcp"
		if i := strings.Index(p, "/"); i >= 0 {
			spec, proto = p[:i], p[i+1:]
		}
		parts := strings.Split(spec, ":")
		if len(parts) < 2 {
			t.Fatalf("port mapping %q is not host:container", p)
		}
		out = append(out, parts[len(parts)-2]+"/"+proto)
	}
	return out
}

// TestNoHostPortCollisions asserts no two networks publish the same host
// port. Before this check, ord, ord-testnet, ord-litecoin and
// ord-litecoin-testnet all bound host port 80, so `ord` and `ord-litecoin`
// could not run at the same time. Every network here can plausibly run
// alongside any other, so there is deliberately no allowlist: a new chain
// (e.g. an L2 defaulting to 8545, which Ethereum Classic already uses) must
// pick a free port.
func TestNoHostPortCollisions(t *testing.T) {
	viper.Set("data-dir", t.TempDir())
	t.Cleanup(func() { viper.Set("data-dir", "") })

	owners := make(map[string][]string)
	for _, row := range networkBuilders {
		cfg, err := row.builder(row.network)
		if err != nil {
			t.Fatalf("builder(%q): %v", row.network, err)
		}
		for _, port := range hostPorts(t, cfg) {
			owners[port] = append(owners[port], row.network)
		}
	}

	for port, networks := range owners {
		if len(networks) > 1 {
			t.Errorf("host port %s is published by more than one network: %v", port, networks)
		}
	}
}

// TestRegistryRPCPortIsPublished ties internal/utils' RPCPort (what
// `request` connects to by default) to what each builder actually publishes,
// so the two can't drift apart.
func TestRegistryRPCPortIsPublished(t *testing.T) {
	viper.Set("data-dir", t.TempDir())
	t.Cleanup(func() { viper.Set("data-dir", "") })

	for _, row := range networkBuilders {
		t.Run(row.network, func(t *testing.T) {
			cfg, err := row.builder(row.network)
			if err != nil {
				t.Fatal(err)
			}

			want := fmt.Sprintf("%d/tcp", utils.NetworkDefaultRPCPorts()[row.network])
			for _, port := range hostPorts(t, cfg) {
				if port == want {
					return
				}
			}
			t.Errorf("registry RPCPort %s is not published by the builder (publishes %v)", want, cfg.Ports)
		})
	}
}

// publishedBinds maps each published host port ("5001/tcp") to the interface
// it is bound on ("" means all interfaces).
func publishedBinds(t *testing.T, cfg NetworkConfig) map[string]string {
	t.Helper()
	out := make(map[string]string)
	for _, p := range cfg.Ports {
		spec, proto := p, "tcp"
		if i := strings.Index(p, "/"); i >= 0 {
			spec, proto = p[:i], p[i+1:]
		}
		parts := strings.Split(spec, ":")
		if len(parts) < 2 {
			t.Fatalf("port mapping %q is not host:container", p)
		}
		bind := ""
		if len(parts) == 3 {
			bind = parts[0]
		}
		out[parts[len(parts)-2]+"/"+proto] = bind
	}
	return out
}

// networksWithPublicByDesign are exempt from the loopback-only RPC rule.
var networksWithPublicByDesign = map[string]string{
	"ord":                  "web explorer meant to be browsed",
	"ord-testnet":          "web explorer meant to be browsed",
	"ord-litecoin":         "web explorer meant to be browsed",
	"ord-litecoin-testnet": "web explorer meant to be browsed",
	"ipfs-cluster":         "REST API on 9094 has not been reviewed; still published on all interfaces",
}

// TestRPCPortsArePublishedOnLoopbackOnly asserts every network's RPC port (the
// registry's RPCPort) is published on 127.0.0.1 only, while at least one peer
// port stays public so the node can be reached by other nodes. RPC ports are
// admin-level APIs; for the Bitcoin family they are protected only by a shared
// password over plain HTTP, with --rpcallowip=0.0.0.0/0 inside the container.
// A new chain must either follow this or be added to networksWithPublicByDesign
// with a reason.
func TestRPCPortsArePublishedOnLoopbackOnly(t *testing.T) {
	viper.Set("data-dir", t.TempDir())
	t.Cleanup(func() { viper.Set("data-dir", "") })

	for _, row := range networkBuilders {
		if reason, exempt := networksWithPublicByDesign[row.network]; exempt {
			t.Logf("%s exempt: %s", row.network, reason)
			continue
		}
		t.Run(row.network, func(t *testing.T) {
			cfg, err := row.builder(row.network)
			if err != nil {
				t.Fatal(err)
			}
			binds := publishedBinds(t, cfg)

			rpc := fmt.Sprintf("%d/tcp", utils.NetworkDefaultRPCPorts()[row.network])
			bind, ok := binds[rpc]
			if !ok {
				t.Fatalf("RPC port %s is not published (publishes %v)", rpc, cfg.Ports)
			}
			if bind != "127.0.0.1" {
				t.Errorf("RPC port %s is published on %q, want 127.0.0.1 (mappings: %v)", rpc, bind, cfg.Ports)
			}

			public := false
			for port, b := range binds {
				if port != rpc && b == "" {
					public = true
				}
			}
			if !public {
				t.Errorf("no peer port is published on all interfaces, so other nodes could not connect (mappings: %v)", cfg.Ports)
			}
		})
	}
}
