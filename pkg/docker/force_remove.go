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
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ForceRemoveAll removes path, which may contain files a Docker container
// wrote as its own internal UID (typically via `gosu nodeuser` in the
// fiftysix images' entrypoints). On native Linux, unlike Docker Desktop on
// macOS, a bind mount does not remap that UID onto the host user, so the
// host process running nodevin can genuinely lack permission to unlink files
// its own container created.
//
// It tries a normal os.RemoveAll first, which is enough whenever there is no
// such mismatch (macOS, or a Linux setup where the container happens to run
// as the host user's own UID). Only on a permission error does it fall back
// to removing path from inside a throwaway container running as root, which
// can delete the files regardless of their host-visible owning UID.
func ForceRemoveAll(path string) error {
	err := os.RemoveAll(path)
	if err == nil || !os.IsPermission(err) {
		return err
	}

	dir, base := filepath.Dir(path), filepath.Base(path)
	cmd := exec.Command("docker", "run", "--rm",
		"-v", dir+":/target",
		"busybox", "rm", "-rf", "/target/"+base)

	out, cerr := cmd.CombinedOutput()
	if cerr != nil {
		return fmt.Errorf("%v (retrying via a container also failed: %s: %w)", err, strings.TrimSpace(string(out)), cerr)
	}
	return nil
}
