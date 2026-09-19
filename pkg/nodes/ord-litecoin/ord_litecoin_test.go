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

package ord_litecoin

import (
	"testing"

	"github.com/fiftysixcrypto/nodevin/pkg/docker/compose"
	"github.com/spf13/viper"
)

// TestResolveOrdLitecoinNetwork is the regression test for a one-character
// typo (network = "ord=litecoin" instead of "ord-litecoin") that used to
// live inline in CreateOrdLitecoinComposeFile's non-testnet branch, which
// meant `nodevin start ord-litecoin` (or `litecoin --ord-litecoin` without
// --testnet) always failed with "unknown network: ord=litecoin". The
// resolution logic was extracted into resolveOrdLitecoinNetwork specifically
// so this could be tested directly, without going through
// CreateOrdLitecoinComposeFile's Docker-dependent tail.
func TestResolveOrdLitecoinNetwork(t *testing.T) {
	t.Run("non-testnet resolves to ord-litecoin", func(t *testing.T) {
		viper.Set("testnet", false)
		viper.Set("network", "")
		t.Cleanup(func() {
			viper.Set("testnet", false)
			viper.Set("network", "")
		})

		if got := resolveOrdLitecoinNetwork(); got != "ord-litecoin" {
			t.Errorf("resolveOrdLitecoinNetwork() = %q, want %q", got, "ord-litecoin")
		}
	})

	t.Run("testnet flag resolves to ord-litecoin-testnet", func(t *testing.T) {
		viper.Set("testnet", true)
		t.Cleanup(func() { viper.Set("testnet", false) })

		if got := resolveOrdLitecoinNetwork(); got != "ord-litecoin-testnet" {
			t.Errorf("resolveOrdLitecoinNetwork() = %q, want %q", got, "ord-litecoin-testnet")
		}
	})
}

// TestResolveOrdLitecoinNetwork_MatchesARealComposeBuilder proves the
// resolved string is one GetOrdLitecoinNetworkComposeConfig actually
// recognizes, closing the loop between "the typo is gone" and "the value we
// produce now is one the compose layer accepts" -- exactly the failure mode
// this bug caused end-to-end.
func TestResolveOrdLitecoinNetwork_MatchesARealComposeBuilder(t *testing.T) {
	viper.Set("data-dir", t.TempDir())
	viper.Set("testnet", false)
	viper.Set("network", "")
	t.Cleanup(func() {
		viper.Set("data-dir", "")
		viper.Set("testnet", false)
		viper.Set("network", "")
	})

	network := resolveOrdLitecoinNetwork()
	if _, err := compose.GetOrdLitecoinNetworkComposeConfig(network); err != nil {
		t.Fatalf("GetOrdLitecoinNetworkComposeConfig(%q) returned error: %v", network, err)
	}
}
