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
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fiftysixcrypto/nodevin/internal/testutil"
)

// Records the arguments each fake binary was called with, so the tests can see
// which Compose flavor ComposeCommand actually picked.
func recorder(name, logfile, extra string) string {
	return `echo "` + name + ` $*" >> ` + logfile + "\n" + extra
}

func run(t *testing.T, args ...string) string {
	t.Helper()
	cmd, err := ComposeCommand(args...)
	if err != nil {
		t.Fatalf("ComposeCommand(%v) returned error: %v", args, err)
	}
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("running %v failed: %v\n%s", cmd.Args, err, out)
	}
	return strings.Join(cmd.Args, " ")
}

func TestComposeCommand_PrefersStandaloneBinary(t *testing.T) {
	log := filepath.Join(t.TempDir(), "calls.log")
	testutil.FakeBins(t, map[string]string{
		"docker-compose": recorder("docker-compose", log, ""),
		"docker":         recorder("docker", log, ""),
	})

	if got := run(t, "-f", "x.yml", "up", "-d"); got != "docker-compose -f x.yml up -d" {
		t.Errorf("command = %q, want the standalone docker-compose", got)
	}
}

// A current Docker Engine install on Linux ships only the plugin. This is the
// case that used to make `nodevin start` fail with `exec: "docker-compose":
// executable file not found in $PATH`.
func TestComposeCommand_FallsBackToThePlugin(t *testing.T) {
	log := filepath.Join(t.TempDir(), "calls.log")
	testutil.FakeBins(t, map[string]string{
		"docker": recorder("docker", log, ""), // `docker compose version` succeeds
	})

	if got := run(t, "-f", "x.yml", "down"); got != "docker compose -f x.yml down" {
		t.Errorf("command = %q, want the docker compose plugin", got)
	}
}

func TestComposeCommand_ErrorsWhenNeitherIsAvailable(t *testing.T) {
	t.Run("docker present without the compose plugin", func(t *testing.T) {
		testutil.FakeBins(t, map[string]string{
			"docker": `[ "$1" = compose ] && { echo "docker: 'compose' is not a docker command" >&2; exit 1; }; exit 0`,
		})
		if _, err := ComposeCommand("up"); !errors.Is(err, ErrComposeNotFound) {
			t.Errorf("err = %v, want ErrComposeNotFound", err)
		}
	})

	t.Run("no docker at all", func(t *testing.T) {
		testutil.FakeBins(t, map[string]string{})
		if _, err := ComposeCommand("up"); !errors.Is(err, ErrComposeNotFound) {
			t.Errorf("err = %v, want ErrComposeNotFound", err)
		}
	})
}

func TestComposeNotFoundMessageNamesBothOptions(t *testing.T) {
	msg := ErrComposeNotFound.Error()
	for _, want := range []string{"docker compose", "docker-compose"} {
		if !strings.Contains(msg, want) {
			t.Errorf("error %q should mention %q", msg, want)
		}
	}
}

// TestComposeVersionOutput_UsesTheRightInvocationPerFlavor is the regression
// test for a real bug found by running `nodevin init` for real against a
// stock, plugin-only Docker install: `docker compose --version` is not
// recognized by the plugin (it treats "--version" as an unknown global flag
// and prints its help text instead of a version banner), so
// checkDockerComposeVersion's version regex could never match and `init`
// reported Compose as not installed even though it was.
func TestComposeVersionOutput_UsesTheRightInvocationPerFlavor(t *testing.T) {
	t.Run("standalone binary uses --version", func(t *testing.T) {
		log := filepath.Join(t.TempDir(), "calls.log")
		testutil.FakeBins(t, map[string]string{
			"docker-compose": recorder("docker-compose", log, `echo "Docker Compose version v2.31.0"`),
			"docker":         recorder("docker", log, ""),
		})

		out, err := ComposeVersionOutput()
		if err != nil {
			t.Fatalf("ComposeVersionOutput() error: %v", err)
		}
		if !strings.Contains(out, "v2.31.0") {
			t.Errorf("output = %q, want it to contain the version", out)
		}

		calls, _ := os.ReadFile(log)
		if !strings.Contains(string(calls), "docker-compose --version") {
			t.Errorf("expected 'docker-compose --version' to run, calls:\n%s", calls)
		}
	})

	t.Run("plugin uses the version subcommand, not --version", func(t *testing.T) {
		log := filepath.Join(t.TempDir(), "calls.log")
		testutil.FakeBins(t, map[string]string{
			// No docker-compose binary at all. `docker compose version` (no
			// dashes) succeeds and prints a banner; `docker compose --version`
			// would, on the real plugin, print help text instead.
			"docker": recorder("docker", log, `
if [ "$1" = compose ] && [ "$2" = "version" ]; then echo "Docker Compose version v2.40.0"; exit 0; fi
if [ "$1" = compose ] && [ "$2" = "--version" ]; then echo "Usage: docker compose [OPTIONS] COMMAND"; exit 0; fi
exit 0`),
		})

		out, err := ComposeVersionOutput()
		if err != nil {
			t.Fatalf("ComposeVersionOutput() error: %v", err)
		}
		if !strings.Contains(out, "v2.40.0") {
			t.Errorf("output = %q, want the plugin's version banner (got what --version would have printed instead?)", out)
		}
	})

	t.Run("neither available", func(t *testing.T) {
		testutil.FakeBins(t, map[string]string{
			"docker": `[ "$1" = compose ] && exit 1; exit 0`,
		})
		if _, err := ComposeVersionOutput(); !errors.Is(err, ErrComposeNotFound) {
			t.Errorf("err = %v, want ErrComposeNotFound", err)
		}
	})
}
