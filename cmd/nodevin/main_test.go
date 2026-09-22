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

package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var binary string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "nodevin-bin")
	if err != nil {
		panic(err)
	}
	binary = filepath.Join(dir, "nodevin")
	if out, err := exec.Command("go", "build", "-o", binary, ".").CombinedOutput(); err != nil {
		os.Stderr.Write(out)
		os.RemoveAll(dir)
		os.Exit(1)
	}

	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

// runNodevin runs the built binary in an empty directory with an empty HOME,
// so no .env file or existing data directory can influence it.
func runNodevin(t *testing.T, args ...string) (exitCode int, output string) {
	t.Helper()

	cmd := exec.Command(binary, args...)
	cmd.Dir = t.TempDir()
	cmd.Env = []string{"HOME=" + t.TempDir(), "PATH=/usr/bin:/bin"}
	out, err := cmd.CombinedOutput()

	var exitErr *exec.ExitError
	switch {
	case err == nil:
		return 0, string(out)
	case errors.As(err, &exitErr):
		return exitErr.ExitCode(), string(out)
	default:
		t.Fatalf("could not run nodevin: %v", err)
		return -1, ""
	}
}

// Regression: main logged errors from cobra and from commands but always
// exited 0, so scripts and CI could not tell that anything had failed.
func TestExitCodes(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want int
	}{
		{"version succeeds", []string{"version"}, 0},
		{"list succeeds", []string{"list"}, 0},
		{"help succeeds", []string{"--help"}, 0},
		{"unknown command", []string{"nonsense"}, 1},
		{"unknown flag", []string{"version", "--no-such-flag"}, 1},
		{"start without a network", []string{"start"}, 1},
		{"start an unsupported network", []string{"start", "not-a-network"}, 1},
		{"stop without a network", []string{"stop"}, 1},
		{"stop with no compose file", []string{"stop", "ipfs"}, 1},
		{"delete without a network", []string{"delete"}, 1},
		{"delete an unsupported network", []string{"delete", "not-a-network"}, 1},
		{"logs without a network", []string{"logs"}, 1},
		{"request without a method", []string{"request", "bitcoin"}, 1},
		{"shell for an unsupported network", []string{"shell", "not-a-network"}, 1},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, out := runNodevin(t, c.args...)
			if got != c.want {
				t.Errorf("nodevin %s exited %d, want %d\noutput:\n%s", strings.Join(c.args, " "), got, c.want, out)
			}
		})
	}
}

// Without a .env file (the normal case) nothing may be logged as an ERROR.
func TestNoEnvFileProducesNoErrorOutput(t *testing.T) {
	code, out := runNodevin(t, "version")
	if code != 0 {
		t.Fatalf("version exited %d: %s", code, out)
	}
	if strings.Contains(out, "ERROR") {
		t.Errorf("a missing .env file must not be reported as an error:\n%s", out)
	}
}

// A failure is reported once, not twice (by cobra and again by main).
func TestErrorsAreReportedOnce(t *testing.T) {
	_, out := runNodevin(t, "start", "not-a-network")
	if n := strings.Count(out, "unsupported blockchain network"); n != 1 {
		t.Errorf("the error appeared %d times, want 1:\n%s", n, out)
	}
}
