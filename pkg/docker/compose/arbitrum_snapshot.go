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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
	"syscall"

	"github.com/spf13/viper"
)

// arbitrumSnapshotBaseURL is Nitro's own default for --init.latest-base;
// used here only to pre-check disk space before phase 1 runs, not passed to
// Nitro itself (it resolves the pointer again on its own).
const arbitrumSnapshotBaseURL = "https://snapshot.arbitrum.foundation/"

// httpGet is a variable so tests can substitute it without reaching the real
// network (mirrors internal/utils.listRunningContainers' own reason for
// being a var).
var httpGet = http.Get

// ResolveArbitrumSnapshotKind returns "" (no snapshot init requested) when
// --snapshot is unset, or the validated kind otherwise: "pruned" (the
// default - hash-scheme database) or "full-path" (opt-in, and not yet usable:
// Nitro's --init.latest refuses it - see the warning on
// ArbitrumSnapshotMetadata.StateScheme and docs/cli-commands.md).
func ResolveArbitrumSnapshotKind() (string, error) {
	kind := strings.TrimSpace(viper.GetString("snapshot"))
	if kind == "" {
		return "", nil
	}
	if kind != "pruned" && kind != "full-path" {
		return "", fmt.Errorf(`invalid --snapshot %q: use "pruned" or "full-path"`, kind)
	}
	return kind, nil
}

// ArbitrumSnapshotMetadata is snapshot.arbitrum.foundation's own
// metadata.json, confirmed against the real files published for both arb1
// (mainnet) and sepolia-rollup (testnet), for both the "pruned" (hash-scheme)
// and "full-path" (path-scheme) kinds.
type ArbitrumSnapshotMetadata struct {
	ChainName    string `json:"chain_name"`
	SnapshotKind string `json:"snapshot_kind"`
	// StateScheme is "hash" for "pruned", "path" for "full-path". Nitro
	// issue #4746 reports a path-scheme snapshot bootstrap stall that hangs
	// silently - "full-path" is opt-in specifically because of this.
	StateScheme  string `json:"state_scheme"`
	NitroVersion string `json:"nitro_version"`
	Size         struct {
		TotalBytes int64  `json:"total_bytes"`
		TotalHuman string `json:"total_human"`
	} `json:"size"`
}

// FetchArbitrumSnapshotMetadata resolves the latest snapshot of kind for
// chainName ("arb1" or "sepolia-rollup") the same way Nitro's own
// --init.latest does: reading <chainName>/latest-<kind>.txt for the current
// snapshot's directory, then that directory's metadata.json. Used only for
// the pre-flight disk check below - Nitro re-resolves this itself at phase 1,
// so a snapshot published between this call and phase 1 starting is Nitro's
// to pick up, not a bug here.
func FetchArbitrumSnapshotMetadata(chainName, kind string) (*ArbitrumSnapshotMetadata, error) {
	pointerURL := arbitrumSnapshotBaseURL + chainName + "/latest-" + kind + ".txt"
	dir, err := fetchArbitrumSnapshotText(pointerURL)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve the latest %s snapshot for %s (%s): %w", kind, chainName, pointerURL, err)
	}
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return nil, fmt.Errorf("empty pointer at %s", pointerURL)
	}
	if !strings.HasSuffix(dir, "/") {
		dir += "/"
	}

	metaURL := arbitrumSnapshotBaseURL + dir + "metadata.json"
	body, err := fetchArbitrumSnapshotText(metaURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch snapshot metadata (%s): %w", metaURL, err)
	}

	var meta ArbitrumSnapshotMetadata
	if err := json.Unmarshal([]byte(body), &meta); err != nil {
		return nil, fmt.Errorf("failed to parse snapshot metadata from %s: %w", metaURL, err)
	}
	return &meta, nil
}

func fetchArbitrumSnapshotText(url string) (string, error) {
	resp, err := httpGet(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status %s", resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// AvailableDiskSpace returns the free space, in bytes, of the filesystem
// holding path, creating path first if it does not exist yet (statfs needs a
// real directory to inspect). Unix-only, matching this repo's existing
// disk-usage code (pkg/docker.CalculateDirSize already shells out to `du`,
// Unix-only too) - no Windows nodevin deployment has needed this yet.
//
// A package-level var, not a plain func, so callers in other packages (the
// Arbitrum snapshot-init flow's own tests) can substitute it to exercise the
// "not enough space" path deterministically, without needing a real
// artificially-constrained filesystem to test against.
var AvailableDiskSpace = func(path string) (uint64, error) {
	if runtime.GOOS == "windows" {
		return 0, fmt.Errorf("disk space checks are not supported on Windows yet")
	}
	if err := os.MkdirAll(path, 0755); err != nil {
		return 0, fmt.Errorf("failed to create %s to check its free space: %w", path, err)
	}
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, fmt.Errorf("failed to statfs %s: %w", path, err)
	}
	return uint64(stat.Bavail) * uint64(stat.Bsize), nil
}
