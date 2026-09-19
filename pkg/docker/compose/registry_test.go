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
