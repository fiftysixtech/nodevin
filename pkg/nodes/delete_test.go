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
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func makeNodeDirs(t *testing.T, dataDir string, names ...string) map[string]string {
	t.Helper()
	dirs := map[string]string{}
	for _, name := range names {
		dir := filepath.Join(dataDir, name)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		dirs[name] = dir
	}
	return dirs
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// Regression for the bug where delete always resolved the mainnet container
// name regardless of --testnet, so it deleted (or missed) the wrong directory.
func TestDeleteNetworkDirectory_RespectsTestnetFlag(t *testing.T) {
	for _, testnet := range []bool{true, false} {
		name := "no testnet flag targets only the mainnet directory"
		remove, keep := "bitcoin-core", "bitcoin-core-testnet"
		if testnet {
			name = "testnet flag targets only the testnet directory"
			remove, keep = keep, remove
		}

		t.Run(name, func(t *testing.T) {
			dataDir := useDataDir(t)
			dirs := makeNodeDirs(t, dataDir, "bitcoin-core", "bitcoin-core-testnet")
			installFakeDocker(t, false, false)
			viper.Set("testnet", testnet)

			if err := deleteNetworkDirectory(dataDir, "bitcoin"); err != nil {
				t.Fatalf("deleteNetworkDirectory() error: %v", err)
			}

			if exists(dirs[remove]) {
				t.Errorf("expected %s to be removed", dirs[remove])
			}
			if !exists(dirs[keep]) {
				t.Errorf("expected %s to be left alone", dirs[keep])
			}
		})
	}
}

// Regression: with --data-dir, `stop` could not find the compose file, failed
// silently, and delete then removed the data of a node that was still running.
func TestDeleteNetworkDirectory_RefusesWhileTheNodeIsStillRunning(t *testing.T) {
	dataDir := useDataDir(t)
	dirs := makeNodeDirs(t, dataDir, "ipfs")
	fake := installFakeDocker(t, true, false)
	fake.setRunning(t, "ipfs") // still up after the stop attempt

	err := deleteNetworkDirectory(dataDir, "ipfs")
	if err == nil {
		t.Fatal("expected delete to refuse to remove the data of a running node")
	}
	if !strings.Contains(err.Error(), "still running") || !strings.Contains(err.Error(), "ipfs") {
		t.Errorf("error %q should say which container is still running", err)
	}
	if !exists(dirs["ipfs"]) {
		t.Error("the data directory of a running node must not be deleted")
	}
}

// The same protection when the stop itself failed because the compose file
// could not be found (a --data-dir mismatch).
func TestDeleteNetworkDirectory_RefusesWhenStopFailsAndTheNodeIsRunning(t *testing.T) {
	dataDir := useDataDir(t)
	dirs := makeNodeDirs(t, dataDir, "ipfs") // no compose file, so stop fails
	fake := installFakeDocker(t, true, false)
	fake.setRunning(t, "ipfs")

	if err := deleteNetworkDirectory(dataDir, "ipfs"); err == nil {
		t.Fatal("expected an error")
	}
	if !exists(dirs["ipfs"]) {
		t.Error("data must survive a failed stop of a running node")
	}
}

func TestDeleteNetworkDirectory_StopsThenDeletes(t *testing.T) {
	dataDir := useDataDir(t)
	composeFile := writeComposeFile(t, dataDir, "ipfs")
	dirs := makeNodeDirs(t, dataDir, "ipfs")
	fake := installFakeDocker(t, true, false) // running per compose, gone per `docker ps` afterwards

	if err := deleteNetworkDirectory(dataDir, "ipfs"); err != nil {
		t.Fatalf("deleteNetworkDirectory() error: %v", err)
	}
	if !strings.Contains(fake.calls(t), composeFile+" down") {
		t.Errorf("delete should stop the node first, calls:\n%s", fake.calls(t))
	}
	if exists(dirs["ipfs"]) {
		t.Error("expected the data directory to be removed")
	}
}

// With no Docker daemon there is nothing running to protect.
func TestDeleteNetworkDirectory_DeletesWhenDockerIsUnavailable(t *testing.T) {
	dataDir := useDataDir(t)
	dirs := makeNodeDirs(t, dataDir, "ipfs")
	installFakeDocker(t, false, true)

	if err := deleteNetworkDirectory(dataDir, "ipfs"); err != nil {
		t.Fatalf("deleteNetworkDirectory() error: %v", err)
	}
	if exists(dirs["ipfs"]) {
		t.Error("expected the data directory to be removed")
	}
}

func TestDeleteNetworkDirectory_Errors(t *testing.T) {
	dataDir := useDataDir(t)
	installFakeDocker(t, false, false)

	if err := deleteNetworkDirectory(dataDir, "not-a-network"); err == nil {
		t.Error("expected an error for an unsupported network")
	}
	if err := deleteNetworkDirectory(dataDir, "ipfs"); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("deleting data that does not exist should error with 'not found', got %v", err)
	}
}

func TestDeleteAllDirectories(t *testing.T) {
	t.Run("removes everything when nothing is running", func(t *testing.T) {
		dataDir := useDataDir(t)
		makeNodeDirs(t, dataDir, "ipfs", "bitcoin-core")
		installFakeDocker(t, false, false)

		if err := deleteAllDirectories(dataDir); err != nil {
			t.Fatalf("deleteAllDirectories() error: %v", err)
		}
		if exists(dataDir) {
			t.Error("expected the whole data directory to be removed")
		}
	})

	t.Run("refuses while any node is still running", func(t *testing.T) {
		dataDir := useDataDir(t)
		makeNodeDirs(t, dataDir, "ipfs", "bitcoin-core")
		fake := installFakeDocker(t, false, false)
		fake.setRunning(t, "bitcoin-core")

		if err := deleteAllDirectories(dataDir); err == nil {
			t.Fatal("expected delete all to refuse")
		}
		if !exists(dataDir) {
			t.Error("the data directory must survive")
		}
	})
}

func TestDeleteCommand_ReturnsTheError(t *testing.T) {
	useDataDir(t)
	installFakeDocker(t, false, false)

	if err := deleteCmd.RunE(deleteCmd, nil); err == nil {
		t.Error("`delete` with no network must return an error")
	}
	if err := deleteCmd.RunE(deleteCmd, []string{"not-a-network"}); err == nil {
		t.Error("`delete` of an unsupported network must return an error")
	}
}
