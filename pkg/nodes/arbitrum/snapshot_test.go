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

package arbitrum

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fiftysixcrypto/nodevin/internal/testutil"
	"github.com/fiftysixcrypto/nodevin/pkg/docker/compose"
	"github.com/spf13/viper"
)

// snapshotTestSetup resets every flag EnsureSnapshotInitialised reads and
// stubs out both runDockerCommand (never actually invoke Docker) and
// compose.AvailableDiskSpace (deterministic, no real disk constraints
// needed) - restored automatically at the end of the test.
func snapshotTestSetup(t *testing.T, availBytes uint64) *bool {
	t.Helper()
	ran := false

	origRun := runDockerCommand
	runDockerCommand = func(args ...string) error {
		ran = true
		return nil
	}

	origDisk := compose.AvailableDiskSpace
	compose.AvailableDiskSpace = func(path string) (uint64, error) {
		return availBytes, nil
	}

	viper.Set("data-dir", t.TempDir())
	t.Cleanup(func() {
		runDockerCommand = origRun
		compose.AvailableDiskSpace = origDisk
		viper.Set("data-dir", "")
		viper.Set("testnet", false)
		viper.Set("snapshot", "")
		viper.Set("snapshot-download-path", "")
		viper.Set("l1-execution-rpc-url", "")
		viper.Set("l1-beacon-url", "")
	})
	return &ran
}

func TestEnsureSnapshotInitialised_NoFlagIsANoop(t *testing.T) {
	ran := snapshotTestSetup(t, 100<<40) // 100 TB, plenty
	// --snapshot deliberately left unset.

	if err := EnsureSnapshotInitialised(t.TempDir()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if *ran {
		t.Error("runDockerCommand was called with --snapshot unset; this must be a complete no-op (requirement: no regression without --snapshot)")
	}
}

func TestEnsureSnapshotInitialised_SkipsAnAlreadyInitialisedDatadir(t *testing.T) {
	ran := snapshotTestSetup(t, 100<<40)
	viper.Set("snapshot", "pruned")

	localChainDataPath, err := compose.ArbitrumLocalChainDataPath("arbitrum")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	marker := filepath.Join(localChainDataPath, "arb1", "nitro", "l2chaindata")
	if err := os.MkdirAll(marker, 0755); err != nil {
		t.Fatalf("failed to create fake database marker: %v", err)
	}

	if err := EnsureSnapshotInitialised(t.TempDir()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if *ran {
		t.Error("runDockerCommand was called against an already-initialised datadir; --snapshot must be a no-op there, never re-initialise (requirement 3)")
	}
}

func TestEnsureSnapshotInitialised_RequiresAnL1(t *testing.T) {
	ran := snapshotTestSetup(t, 100<<40)
	viper.Set("snapshot", "pruned")
	testutil.FakeBins(t, map[string]string{"docker": `case "$1" in ps) printf '' ;; esac`})

	err := EnsureSnapshotInitialised(t.TempDir())
	if err == nil {
		t.Fatal("expected an error when no L1 is configured or running")
	}
	if *ran {
		t.Error("runDockerCommand was called despite no L1 being resolvable")
	}
}

func TestEnsureSnapshotInitialised_RefusesWhenDownloadPathIsTooSmall(t *testing.T) {
	// 1 KB free - far below any real snapshot's size, so this must refuse
	// before ever invoking Docker (requirement 5).
	ran := snapshotTestSetup(t, 1024)
	viper.Set("snapshot", "pruned")
	viper.Set("l1-execution-rpc-url", "https://l1.example.com/rpc")
	viper.Set("l1-beacon-url", "https://l1.example.com/beacon")

	err := EnsureSnapshotInitialised(t.TempDir())
	if err == nil {
		t.Fatal("expected a disk-space refusal, got nil")
	}
	if !strings.Contains(err.Error(), "not enough free space") {
		t.Errorf("error = %q, want it to clearly name insufficient free space", err.Error())
	}
	if *ran {
		t.Error("runDockerCommand was called despite the disk pre-check failing; must never start a download that cannot finish")
	}
}

func TestEnsureSnapshotInitialised_RunsInitWhenEverythingChecksOut(t *testing.T) {
	snapshotTestSetup(t, 100<<40)
	viper.Set("snapshot", "pruned")
	viper.Set("l1-execution-rpc-url", "https://l1.example.com/rpc")
	viper.Set("l1-beacon-url", "https://l1.example.com/beacon")

	var capturedArgs []string
	runDockerCommand = func(args ...string) error {
		capturedArgs = args
		return nil
	}

	if err := EnsureSnapshotInitialised(t.TempDir()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	joined := strings.Join(capturedArgs, " ")
	for _, want := range []string{
		"fiftysix/nitro:latest",
		"--chain.id=42161", "--parent-chain.id=1",
		"--parent-chain.connection.url=https://l1.example.com/rpc",
		"--parent-chain.blob-client.beacon-url=https://l1.example.com/beacon",
		"--init.latest=pruned", "--init.then-quit",
		":/node/nitro/data", ":/node/nitro/snapshot-staging",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("docker run args = %q, want it to contain %q", joined, want)
		}
	}
}

func TestEnsureSnapshotInitialised_PropagatesAFailedInit(t *testing.T) {
	ran := snapshotTestSetup(t, 100<<40)
	viper.Set("snapshot", "pruned")
	viper.Set("l1-execution-rpc-url", "https://l1.example.com/rpc")
	viper.Set("l1-beacon-url", "https://l1.example.com/beacon")

	runDockerCommand = func(args ...string) error {
		*ran = true
		return errors.New("exit status 1")
	}

	err := EnsureSnapshotInitialised(t.TempDir())
	if err == nil {
		t.Fatal("expected the init failure to propagate")
	}
	if !strings.Contains(err.Error(), "staging left intact") {
		t.Errorf("error = %q, want it to reassure that staging was left intact for a resumed retry (requirement 9)", err.Error())
	}
}

func TestEnsureSnapshotInitialised_TestnetUsesSepoliaRollupAndChainIDs(t *testing.T) {
	snapshotTestSetup(t, 100<<40)
	viper.Set("snapshot", "pruned")
	viper.Set("testnet", true)
	viper.Set("l1-execution-rpc-url", "https://l1.example.com/rpc")
	viper.Set("l1-beacon-url", "https://l1.example.com/beacon")

	var capturedArgs []string
	runDockerCommand = func(args ...string) error {
		capturedArgs = args
		return nil
	}

	if err := EnsureSnapshotInitialised(t.TempDir()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	joined := strings.Join(capturedArgs, " ")
	if !strings.Contains(joined, "--chain.id=421614") || !strings.Contains(joined, "--parent-chain.id=11155111") {
		t.Errorf("docker run args = %q, want Arbitrum/Ethereum Sepolia chain IDs", joined)
	}
}

func TestEnsureSnapshotInitialised_InvalidKindIsAnError(t *testing.T) {
	snapshotTestSetup(t, 100<<40)
	viper.Set("snapshot", "archive") // not a supported value (requirement 1: pruned or full-path only)

	if err := EnsureSnapshotInitialised(t.TempDir()); err == nil {
		t.Fatal("expected an error for an unsupported --snapshot value")
	}
}
