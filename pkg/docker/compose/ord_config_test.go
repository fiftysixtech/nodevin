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

// resetOrdViperKeys clears every viper key GetOrdNetworkComposeConfig reads,
// so tests don't leak state into each other via the shared global viper
// instance (see internal/utils's GetNodevinDataDir for the same hazard).
func resetOrdViperKeys(t *testing.T) {
	t.Helper()
	keys := []string{"ord-cookie-auth", "ord-rpc-user", "ord-rpc-pass"}
	for _, k := range keys {
		viper.Set(k, "")
	}
	t.Cleanup(func() {
		for _, k := range keys {
			viper.Set(k, "")
		}
	})
}

// TestGetOrdNetworkComposeConfig_RPCAuth is the regression test for the bug
// where --ord-rpc-user/--ord-rpc-pass/--ord-cookie-auth were read via viper
// but never registered as CLI flags anywhere, so they could never actually
// be set by a user -- ord always silently fell back to the hardcoded
// user/fiftysix credentials. Now that pkg/root registers them, this proves
// they flow through into the generated Command.
func TestGetOrdNetworkComposeConfig_RPCAuth(t *testing.T) {
	viper.Set("data-dir", t.TempDir())
	t.Cleanup(func() { viper.Set("data-dir", "") })

	t.Run("custom rpc-user/rpc-pass flow into the command", func(t *testing.T) {
		resetOrdViperKeys(t)
		viper.Set("ord-rpc-user", "custom-user")
		viper.Set("ord-rpc-pass", "custom-pass")

		cfg, err := GetOrdNetworkComposeConfig("ord")
		if err != nil {
			t.Fatalf("GetOrdNetworkComposeConfig() returned error: %v", err)
		}
		if !strings.Contains(cfg.Command, "--bitcoin-rpc-username custom-user --bitcoin-rpc-password custom-pass") {
			t.Errorf("Command = %q, want it to contain the custom rpc-user/rpc-pass", cfg.Command)
		}
	})

	t.Run("defaults apply when unset", func(t *testing.T) {
		resetOrdViperKeys(t)

		cfg, err := GetOrdNetworkComposeConfig("ord")
		if err != nil {
			t.Fatalf("GetOrdNetworkComposeConfig() returned error: %v", err)
		}
		if !strings.Contains(cfg.Command, "--bitcoin-rpc-username user --bitcoin-rpc-password fiftysix") {
			t.Errorf("Command = %q, want the default user/fiftysix credentials", cfg.Command)
		}
	})

	t.Run("cookie-auth skips rpc-user/rpc-pass entirely", func(t *testing.T) {
		resetOrdViperKeys(t)
		viper.Set("ord-cookie-auth", true)
		viper.Set("ord-rpc-user", "should-not-appear")

		cfg, err := GetOrdNetworkComposeConfig("ord")
		if err != nil {
			t.Fatalf("GetOrdNetworkComposeConfig() returned error: %v", err)
		}
		if strings.Contains(cfg.Command, "--bitcoin-rpc-username") {
			t.Errorf("Command = %q, want no --bitcoin-rpc-username flag when cookie-auth is set", cfg.Command)
		}
	})
}
