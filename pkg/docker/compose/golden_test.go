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
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

// TestCreateComposeFile_BitcoinGolden guards against subtle docker-compose
// YAML-shape regressions that a struct-level assertion wouldn't catch (field
// ordering, indentation, an accidentally-added/removed key). It's the first
// golden-file test in this repo -- to refresh the fixture after an
// intentional change, run:
//
//	UPDATE_GOLDEN=1 go test ./pkg/docker/compose/... -run TestCreateComposeFile_BitcoinGolden
//
// Every viper key GetBitcoinNetworkComposeConfig/CreateComposeFile reads is
// pinned to a fixed value below, EXCEPT "volumes"/"volume-definitions"/
// "volume-labels": CreateComposeFile checks viper.IsSet (not just the value)
// on those three to decide whether to skip the init-container step, and
// viper.Set marks a key as "set" even when set to its zero value (the same
// hazard GetNodevinDataDir has around "data-dir") -- so those three are left
// completely untouched to keep the init-container present deterministically,
// matching an always-empty t.TempDir().
func TestCreateComposeFile_BitcoinGolden(t *testing.T) {
	tmpDir := t.TempDir()
	viper.Set("data-dir", tmpDir)

	pinned := map[string]interface{}{
		"cookie-auth":           false,
		"rpc-user":              "",
		"rpc-pass":              "",
		"image":                 "",
		"version":               "",
		"restart":               "",
		"container-name":        "",
		"command":               "",
		"ports":                 []string{},
		"docker-networks":       []string{},
		"network-driver":        "",
		"cpu-limit":             "",
		"mem-limit":             "",
		"cpu-reservation":       "",
		"mem-reservation":       "",
		"snapshot-sync":         false,
		"snapshot-sync-command": "",
	}
	for k, v := range pinned {
		viper.Set(k, v)
	}
	t.Cleanup(func() {
		viper.Set("data-dir", "")
		for k, v := range pinned {
			viper.Set(k, v)
		}
	})

	cfg, err := GetBitcoinNetworkComposeConfig("bitcoin")
	if err != nil {
		t.Fatalf("GetBitcoinNetworkComposeConfig() returned error: %v", err)
	}

	path, err := CreateComposeFile("bitcoin-core", cfg, []string{}, []NetworkConfig{}, tmpDir)
	if err != nil {
		t.Fatalf("CreateComposeFile() returned error: %v", err)
	}

	rawActual, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read generated compose file: %v", err)
	}

	// t.TempDir() embeds a fresh, run-specific path into every generated
	// volume mount, so it must be normalized to a stable placeholder before
	// comparing against (or writing) the golden fixture -- otherwise this
	// test could never pass on a second run.
	actual := strings.ReplaceAll(string(rawActual), tmpDir, "TMPDIR")

	goldenPath := filepath.Join("testdata", "bitcoin-compose-golden.yml")

	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.MkdirAll(filepath.Dir(goldenPath), 0755); err != nil {
			t.Fatalf("failed to create testdata directory: %v", err)
		}
		if err := os.WriteFile(goldenPath, []byte(actual), 0644); err != nil {
			t.Fatalf("failed to write golden fixture: %v", err)
		}
		t.Logf("updated golden fixture at %s", goldenPath)
		return
	}

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("failed to read golden fixture at %s (run with UPDATE_GOLDEN=1 to create it): %v", goldenPath, err)
	}

	if actual != string(want) {
		t.Errorf("generated compose file does not match golden fixture %s.\n--- got ---\n%s\n--- want ---\n%s", goldenPath, actual, want)
	}
}
