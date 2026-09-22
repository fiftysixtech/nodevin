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

package docker

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/fiftysixcrypto/nodevin/internal/testutil"
)

func TestForceRemoveAll_NormalCaseNeverInvokesDocker(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "data")
	if err := os.MkdirAll(filepath.Join(target, "sub"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(target, "sub", "f"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	log := filepath.Join(t.TempDir(), "calls.log")
	testutil.FakeBins(t, map[string]string{
		"docker": `echo "docker $*" >> ` + log,
	})

	if err := ForceRemoveAll(target); err != nil {
		t.Fatalf("ForceRemoveAll() error: %v", err)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Errorf("expected %s to be removed", target)
	}
	if b, _ := os.ReadFile(log); len(b) > 0 {
		t.Errorf("docker should never be invoked on the normal path, but was called: %s", b)
	}
}

// permissionDeniedDir makes an os.RemoveAll(target) call fail with a genuine
// permission error, portably and without root: the parent directory's write
// bit is what actually gates unlinking an entry on it, regardless of the
// entry's own permissions, so removing the parent's write bit is enough to
// reproduce the real failure mode (a file the calling user cannot unlink)
// without needing Docker, containers, or a UID mismatch at all.
func permissionDeniedDir(t *testing.T) (root, target string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("permission bits work differently on Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("root ignores the permission bits this test relies on")
	}

	root = t.TempDir()
	target = filepath.Join(root, "data")
	if err := os.Mkdir(target, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(target, "f"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(root, 0500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(root, 0755) }) // let t.TempDir() clean up
	return root, target
}

func TestForceRemoveAll_FallsBackToAContainerOnPermissionError(t *testing.T) {
	_, target := permissionDeniedDir(t)

	log := filepath.Join(t.TempDir(), "calls.log")
	testutil.FakeBins(t, map[string]string{
		"docker": `echo "docker $*" >> ` + log,
	})

	if err := ForceRemoveAll(target); err != nil {
		t.Fatalf("ForceRemoveAll() error: %v, want the fake docker call to report success", err)
	}

	calls, err := os.ReadFile(log)
	if err != nil {
		t.Fatalf("docker was never invoked: %v", err)
	}
	got := string(calls)
	for _, want := range []string{"run", "--rm", "-v", "busybox", "rm", "-rf", "/target/data"} {
		if !strings.Contains(got, want) {
			t.Errorf("docker call %q should contain %q", got, want)
		}
	}
}

func TestForceRemoveAll_ReportsBothErrorsWhenTheFallbackAlsoFails(t *testing.T) {
	_, target := permissionDeniedDir(t)

	testutil.FakeBins(t, map[string]string{
		"docker": `echo "no busybox image and no network" >&2; exit 1`,
	})

	err := ForceRemoveAll(target)
	if err == nil {
		t.Fatal("expected an error when both the direct removal and the fallback fail")
	}
	if !strings.Contains(err.Error(), "permission denied") {
		t.Errorf("error %q should mention the original permission error", err)
	}
	if !strings.Contains(err.Error(), "no busybox image and no network") {
		t.Errorf("error %q should mention why the fallback failed too", err)
	}
}
