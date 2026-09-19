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

package nodes

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

// TestDeleteNetworkDirectory_RespectsTestnetFlag is the regression test for
// the bug where delete always resolved the mainnet container name regardless
// of --testnet, so it deleted (or missed) the wrong data directory.
//
// deleteNetworkDirectory calls stopNode() first, which shells out to
// `docker-compose ... ps -q`; if that binary/daemon isn't available it logs
// an error and returns without panicking (confirmed by reading
// pkg/nodes/stop_node.go), so this is safe to run without Docker present.
func TestDeleteNetworkDirectory_RespectsTestnetFlag(t *testing.T) {
	t.Run("testnet flag targets only the testnet directory", func(t *testing.T) {
		baseDir := t.TempDir()
		mainnetDir := filepath.Join(baseDir, "bitcoin-core")
		testnetDir := filepath.Join(baseDir, "bitcoin-core-testnet")
		mustMkdirAll(t, mainnetDir)
		mustMkdirAll(t, testnetDir)

		viper.Set("testnet", true)
		t.Cleanup(func() { viper.Set("testnet", false) })

		deleteNetworkDirectory(baseDir, "bitcoin")

		if _, err := os.Stat(testnetDir); !os.IsNotExist(err) {
			t.Errorf("expected testnet directory %q to be removed, stat err: %v", testnetDir, err)
		}
		if _, err := os.Stat(mainnetDir); err != nil {
			t.Errorf("expected mainnet directory %q to be left alone, got: %v", mainnetDir, err)
		}
	})

	t.Run("no testnet flag targets only the mainnet directory", func(t *testing.T) {
		baseDir := t.TempDir()
		mainnetDir := filepath.Join(baseDir, "bitcoin-core")
		testnetDir := filepath.Join(baseDir, "bitcoin-core-testnet")
		mustMkdirAll(t, mainnetDir)
		mustMkdirAll(t, testnetDir)

		viper.Set("testnet", false)
		t.Cleanup(func() { viper.Set("testnet", false) })

		deleteNetworkDirectory(baseDir, "bitcoin")

		if _, err := os.Stat(mainnetDir); !os.IsNotExist(err) {
			t.Errorf("expected mainnet directory %q to be removed, stat err: %v", mainnetDir, err)
		}
		if _, err := os.Stat(testnetDir); err != nil {
			t.Errorf("expected testnet directory %q to be left alone, got: %v", testnetDir, err)
		}
	})
}

func mustMkdirAll(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("failed to create fixture directory %q: %v", dir, err)
	}
}
