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

func resetOrdLitecoinViperKeys(t *testing.T) {
	t.Helper()
	keys := []string{
		"ord-litecoin-cookie-auth", "ord-litecoin-rpc-user", "ord-litecoin-rpc-pass",
		"ord-cookie-auth", "ord-rpc-user", "ord-rpc-pass",
	}
	for _, k := range keys {
		viper.Set(k, "")
	}
	t.Cleanup(func() {
		for _, k := range keys {
			viper.Set(k, "")
		}
	})
}

// TestGetOrdLitecoinNetworkComposeConfig_RPCAuthFallbackChain proves the
// 3-tier fallback (ord-litecoin-rpc-* -> ord-rpc-* -> hardcoded default) at
// each tier, and is the regression test for the same class of bug as
// ord_config_test.go: none of these keys were registered as CLI flags before
// pkg/root added them, so a user could never actually set them.
func TestGetOrdLitecoinNetworkComposeConfig_RPCAuthFallbackChain(t *testing.T) {
	viper.Set("data-dir", t.TempDir())
	t.Cleanup(func() { viper.Set("data-dir", "") })

	t.Run("tier 1: ord-litecoin-rpc-user/pass used directly when set", func(t *testing.T) {
		resetOrdLitecoinViperKeys(t)
		viper.Set("ord-litecoin-rpc-user", "litecoin-specific-user")
		viper.Set("ord-litecoin-rpc-pass", "litecoin-specific-pass")
		viper.Set("ord-rpc-user", "should-not-be-used")
		viper.Set("ord-rpc-pass", "should-not-be-used")

		cfg, err := GetOrdLitecoinNetworkComposeConfig("ord-litecoin")
		if err != nil {
			t.Fatalf("GetOrdLitecoinNetworkComposeConfig() returned error: %v", err)
		}
		if !strings.Contains(cfg.Command, "--litecoin-rpc-username litecoin-specific-user --litecoin-rpc-password litecoin-specific-pass") {
			t.Errorf("Command = %q, want the ord-litecoin-specific credentials to win", cfg.Command)
		}
	})

	t.Run("tier 2: falls back to ord-rpc-user/pass when ord-litecoin-specific is unset", func(t *testing.T) {
		resetOrdLitecoinViperKeys(t)
		viper.Set("ord-rpc-user", "shared-ord-user")
		viper.Set("ord-rpc-pass", "shared-ord-pass")

		cfg, err := GetOrdLitecoinNetworkComposeConfig("ord-litecoin")
		if err != nil {
			t.Fatalf("GetOrdLitecoinNetworkComposeConfig() returned error: %v", err)
		}
		if !strings.Contains(cfg.Command, "--litecoin-rpc-username shared-ord-user --litecoin-rpc-password shared-ord-pass") {
			t.Errorf("Command = %q, want it to fall back to the shared ord-rpc-user/pass", cfg.Command)
		}
	})

	t.Run("tier 3: falls back to hardcoded default when nothing is set", func(t *testing.T) {
		resetOrdLitecoinViperKeys(t)

		cfg, err := GetOrdLitecoinNetworkComposeConfig("ord-litecoin")
		if err != nil {
			t.Fatalf("GetOrdLitecoinNetworkComposeConfig() returned error: %v", err)
		}
		if !strings.Contains(cfg.Command, "--litecoin-rpc-username user --litecoin-rpc-password fiftysix") {
			t.Errorf("Command = %q, want the hardcoded user/fiftysix defaults", cfg.Command)
		}
	})

	t.Run("cookie-auth via ord-litecoin-cookie-auth skips rpc flags", func(t *testing.T) {
		resetOrdLitecoinViperKeys(t)
		viper.Set("ord-litecoin-cookie-auth", true)

		cfg, err := GetOrdLitecoinNetworkComposeConfig("ord-litecoin")
		if err != nil {
			t.Fatalf("GetOrdLitecoinNetworkComposeConfig() returned error: %v", err)
		}
		if strings.Contains(cfg.Command, "--litecoin-rpc-username") {
			t.Errorf("Command = %q, want no --litecoin-rpc-username flag when ord-litecoin-cookie-auth is set", cfg.Command)
		}
	})

	t.Run("cookie-auth via shared ord-cookie-auth also skips rpc flags", func(t *testing.T) {
		resetOrdLitecoinViperKeys(t)
		viper.Set("ord-cookie-auth", true)

		cfg, err := GetOrdLitecoinNetworkComposeConfig("ord-litecoin")
		if err != nil {
			t.Fatalf("GetOrdLitecoinNetworkComposeConfig() returned error: %v", err)
		}
		if strings.Contains(cfg.Command, "--litecoin-rpc-username") {
			t.Errorf("Command = %q, want no --litecoin-rpc-username flag when the shared ord-cookie-auth is set", cfg.Command)
		}
	})
}
