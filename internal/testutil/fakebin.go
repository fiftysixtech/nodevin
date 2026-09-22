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
	"io"
	"os"
	"os/exec"
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

// externalTools are non-builtin commands the fake scripts themselves invoke
// (checked against every scripts body in this repo at the time of writing:
// only "cat"; everything else they use -- echo, case, [, exit -- is a sh
// builtin). They are copied into the fake directory so they keep resolving
// even when a real directory that happens to also host one of them (e.g.
// /usr/bin on a merged-usr Linux, which hosts both `docker` and `cat`) is
// dropped from PATH below. Extend this if a future fake script needs another
// external tool.
var externalTools = []string{"cat"}

// FakeBins puts fake executables at the front of PATH for the duration of the
// test and returns the directory holding them. Each map key is an executable
// name and each value is the body of a POSIX shell script. Tests that use it
// are skipped on Windows.
//
// The rest of the real PATH is kept after the fake directory, so anything the
// fake scripts need beyond externalTools still resolves normally, but any
// directory that itself provides a real protectedBinaries entry is dropped
// from it. Without this, a test that omits e.g. "docker" from scripts to
// simulate it being entirely absent can be defeated by a real docker sitting
// in a directory that was only kept on PATH for basic tools.
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
	for _, name := range externalTools {
		if _, ok := scripts[name]; ok {
			continue // the test wants its own fake for this one
		}
		copyExecutable(t, name, dir)
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

// copyExecutable resolves name on the test process's real, original PATH and
// copies it into dir, so it is available from dir regardless of what happens
// to the rest of PATH afterward.
func copyExecutable(t *testing.T, name, dir string) {
	t.Helper()

	src, err := exec.LookPath(name)
	if err != nil {
		t.Fatalf("FakeBins needs a real %q on PATH to copy for the fake scripts, but none was found: %v", name, err)
	}

	in, err := os.Open(src)
	if err != nil {
		t.Fatalf("failed to open %s: %v", src, err)
	}
	defer in.Close()

	out, err := os.OpenFile(filepath.Join(dir, name), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		t.Fatalf("failed to create fake-dir copy of %s: %v", name, err)
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		t.Fatalf("failed to copy %s into the fake dir: %v", name, err)
	}
}
