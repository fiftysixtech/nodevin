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

// Package testutil holds helpers shared by tests.
package testutil

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// protectedBinaries are executables FakeBins guarantees are never reachable
// from the real system, even when a test's scripts map doesn't mention them
// -- e.g. to simulate that binary being entirely absent. Extend this if a
// future test needs to hide something else.
var protectedBinaries = []string{"docker", "docker-compose"}

// FakeBins puts fake executables at the front of PATH for the duration of the
// test and returns the directory holding them. Each map key is an executable
// name and each value is the body of a POSIX shell script. Tests that use it
// are skipped on Windows.
//
// The rest of the real PATH is kept after the fake directory, so the fake
// scripts' own basic tools (sh, cat, ...) still resolve, but any directory
// that itself provides a real protectedBinaries entry is dropped from it.
// Without this, a test that omits e.g. "docker" from scripts to simulate it
// being entirely absent can be defeated by a real docker sitting in a
// directory -- /usr/bin on many Linux distros, for one -- that was only kept
// on PATH for unrelated basic tools.
func FakeBins(t *testing.T, scripts map[string]string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake executables use POSIX shell scripts")
	}

	dir := t.TempDir()
	for name, body := range scripts {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0755); err != nil {
			t.Fatalf("failed to write fake %s: %v", name, err)
		}
	}

	var kept []string
	for _, realDir := range filepath.SplitList(os.Getenv("PATH")) {
		hasProtected := false
		for _, name := range protectedBinaries {
			if _, err := os.Stat(filepath.Join(realDir, name)); err == nil {
				hasProtected = true
				break
			}
		}
		if !hasProtected {
			kept = append(kept, realDir)
		}
	}

	t.Setenv("PATH", strings.Join(append([]string{dir}, kept...), string(os.PathListSeparator)))
	return dir
}
