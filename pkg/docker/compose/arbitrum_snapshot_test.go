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
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func resetSnapshotFlag(t *testing.T) {
	t.Helper()
	t.Cleanup(func() { viper.Set("snapshot", "") })
}

func TestResolveArbitrumSnapshotKind(t *testing.T) {
	cases := []struct {
		name    string
		value   string
		want    string
		wantErr bool
	}{
		{"unset", "", "", false},
		{"bare flag (NoOptDefVal)", "pruned", "pruned", false},
		{"explicit full-path", "full-path", "full-path", false},
		{"invalid value", "archive", "", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			resetSnapshotFlag(t)
			viper.Set("snapshot", c.value)
			got, err := ResolveArbitrumSnapshotKind()
			if c.wantErr {
				if err == nil {
					t.Fatalf("expected an error for --snapshot=%q, got nil", c.value)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != c.want {
				t.Errorf("ResolveArbitrumSnapshotKind() = %q, want %q", got, c.want)
			}
		})
	}
}

// fakeHTTPGet returns canned responses keyed by exact URL, so
// fetchArbitrumSnapshotText's two sequential calls (pointer, then
// metadata.json) can be scripted without touching the real network.
func fakeHTTPGet(t *testing.T, responses map[string]string) func() {
	t.Helper()
	orig := httpGet
	httpGet = func(url string) (*http.Response, error) {
		body, ok := responses[url]
		if !ok {
			t.Fatalf("unexpected URL requested: %s", url)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Body:       io.NopCloser(strings.NewReader(body)),
		}, nil
	}
	return func() { httpGet = orig }
}

func TestFetchArbitrumSnapshotMetadata(t *testing.T) {
	restore := fakeHTTPGet(t, map[string]string{
		arbitrumSnapshotBaseURL + "arb1/latest-pruned.txt": "arb1/2026-09-23-141f8381/",
		arbitrumSnapshotBaseURL + "arb1/2026-09-23-141f8381/metadata.json": `{
			"database_type": "pebble",
			"state_scheme": "hash",
			"snapshot_kind": "pruned",
			"chain_name": "arb1",
			"nitro_version": "v3.12.0-rc.3-ebe9e83",
			"size": {"total_bytes": 2648571136000, "total_human": "2.41 TB"}
		}`,
	})
	defer restore()

	meta, err := FetchArbitrumSnapshotMetadata("arb1", "pruned")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.ChainName != "arb1" {
		t.Errorf("ChainName = %q, want arb1", meta.ChainName)
	}
	if meta.StateScheme != "hash" {
		t.Errorf("StateScheme = %q, want hash", meta.StateScheme)
	}
	if meta.Size.TotalBytes != 2648571136000 {
		t.Errorf("Size.TotalBytes = %d, want 2648571136000", meta.Size.TotalBytes)
	}
}

func TestFetchArbitrumSnapshotMetadata_FullPathIsPathScheme(t *testing.T) {
	// Confirmed against the real snapshot.arbitrum.foundation metadata: the
	// full-path kind really is state_scheme "path", the one requirement 6
	// warns about (OffchainLabs/nitro#4746).
	restore := fakeHTTPGet(t, map[string]string{
		arbitrumSnapshotBaseURL + "sepolia-rollup/latest-full-path.txt": "sepolia-rollup/2026-09-26-8895cd7b/",
		arbitrumSnapshotBaseURL + "sepolia-rollup/2026-09-26-8895cd7b/metadata.json": `{
			"state_scheme": "path",
			"snapshot_kind": "full-path",
			"chain_name": "sepolia-rollup",
			"size": {"total_bytes": 1888409057280, "total_human": "1.72 TB"}
		}`,
	})
	defer restore()

	meta, err := FetchArbitrumSnapshotMetadata("sepolia-rollup", "full-path")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.StateScheme != "path" {
		t.Errorf("StateScheme = %q, want path", meta.StateScheme)
	}
}

func TestFetchArbitrumSnapshotMetadata_PointerNotFound(t *testing.T) {
	orig := httpGet
	defer func() { httpGet = orig }()
	httpGet = func(url string) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusNotFound, Status: "404 Not Found", Body: io.NopCloser(strings.NewReader(""))}, nil
	}

	if _, err := FetchArbitrumSnapshotMetadata("arb1", "pruned"); err == nil {
		t.Fatal("expected an error when the pointer file is not found")
	}
}

func TestAvailableDiskSpace(t *testing.T) {
	avail, err := AvailableDiskSpace(t.TempDir())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if avail == 0 {
		t.Error("AvailableDiskSpace() = 0, want a real nonzero value for a real temp dir")
	}
}
