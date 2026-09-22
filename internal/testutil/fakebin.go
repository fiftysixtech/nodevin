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
	"testing"
)

// FakeBins puts fake executables at the front of PATH for the duration of the
// test and returns the directory holding them. Each map key is an executable
// name and each value is the body of a POSIX shell script. Tests that use it
// are skipped on Windows. Only the fake directory (plus /usr/bin:/bin for
// basic tools) is on PATH, so a real docker or docker-compose can never leak in.
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
	t.Setenv("PATH", dir+string(os.PathListSeparator)+"/usr/bin"+string(os.PathListSeparator)+"/bin")
	return dir
}
