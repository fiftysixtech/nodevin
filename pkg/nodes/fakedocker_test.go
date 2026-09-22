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

	"github.com/fiftysixcrypto/nodevin/internal/testutil"
	"github.com/spf13/viper"
)

// fakeDocker installs a fake `docker` and `docker-compose` on PATH.
//
//   - `docker ps` prints the contents of the returned psFile (names, one per
//     line), so a test can decide which containers "are running".
//   - every invocation of either binary is appended to the returned callsFile.
//   - `docker-compose ps -q` prints an id iff composeRunning is true.
//
// When dockerBroken is true, `docker` exits 1 for everything, like a stopped
// daemon.
type fakeDocker struct {
	psFile    string
	callsFile string
}

func installFakeDocker(t *testing.T, composeRunning, dockerBroken bool) fakeDocker {
	t.Helper()
	dir := t.TempDir()
	f := fakeDocker{
		psFile:    filepath.Join(dir, "ps.txt"),
		callsFile: filepath.Join(dir, "calls.txt"),
	}
	os.WriteFile(f.psFile, nil, 0644)
	os.WriteFile(f.callsFile, nil, 0644)

	dockerBody := `echo "docker $*" >> ` + f.callsFile + "\n"
	if dockerBroken {
		dockerBody += "exit 1"
	} else {
		dockerBody += `case "$1" in ps) cat ` + f.psFile + ` ;; esac`
	}

	composeBody := `echo "docker-compose $*" >> ` + f.callsFile + "\n"
	if composeRunning {
		composeBody += `case "$*" in *"ps -q"*) echo abc123 ;; esac`
	}

	testutil.FakeBins(t, map[string]string{"docker": dockerBody, "docker-compose": composeBody})
	return f
}

func (f fakeDocker) setRunning(t *testing.T, names ...string) {
	t.Helper()
	if err := os.WriteFile(f.psFile, []byte(strings.Join(names, "\n")+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
}

func (f fakeDocker) calls(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(f.callsFile)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// useDataDir points --data-dir at a fresh directory and returns the
// ".nodevin/data" directory nodevin will actually use inside it.
func useDataDir(t *testing.T) string {
	t.Helper()
	viper.Set("data-dir", t.TempDir())
	viper.Set("testnet", false)
	viper.Set("network", "")
	t.Cleanup(func() {
		viper.Set("data-dir", "")
		viper.Set("testnet", false)
		viper.Set("network", "")
	})
	dir := filepath.Join(viper.GetString("data-dir"), ".nodevin", "data")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func writeComposeFile(t *testing.T, dataDir, container string) string {
	t.Helper()
	path := filepath.Join(dataDir, "docker-compose_"+container+".yml")
	if err := os.WriteFile(path, []byte("services: {}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
