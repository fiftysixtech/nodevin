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
	"os/exec"
)

// ErrComposeNotFound is returned when neither the standalone `docker-compose`
// binary nor the `docker compose` plugin is available.
var ErrComposeNotFound = errors.New("Docker Compose not found: install the Docker Compose plugin (`docker compose`) or the standalone `docker-compose` binary")

type composeFlavor int

const (
	composeNone composeFlavor = iota
	composeStandalone
	composePlugin
)

// detectCompose reports which Docker Compose is available, preferring the
// standalone `docker-compose` binary, which is what nodevin has always used,
// and falling back to the `docker compose` plugin. A current Docker Engine
// install on Linux ships only the plugin, and Docker Desktop provides both.
func detectCompose() composeFlavor {
	if _, err := exec.LookPath("docker-compose"); err == nil {
		return composeStandalone
	}
	if exec.Command("docker", "compose", "version").Run() == nil {
		return composePlugin
	}
	return composeNone
}

// ComposeCommand returns a command that runs Docker Compose with args.
func ComposeCommand(args ...string) (*exec.Cmd, error) {
	switch detectCompose() {
	case composeStandalone:
		return exec.Command("docker-compose", args...), nil
	case composePlugin:
		return exec.Command("docker", append([]string{"compose"}, args...)...), nil
	default:
		return nil, ErrComposeNotFound
	}
}

// ComposeVersionOutput returns Docker Compose's own version banner.
//
// The standalone binary and the plugin use different invocations to print it
// ("docker-compose --version" vs "docker compose version" — the plugin treats
// "--version" as an unrecognized global flag and prints its help text
// instead), so this cannot be expressed as a plain ComposeCommand("--version")
// call.
func ComposeVersionOutput() (string, error) {
	switch detectCompose() {
	case composeStandalone:
		out, err := exec.Command("docker-compose", "--version").CombinedOutput()
		return string(out), err
	case composePlugin:
		out, err := exec.Command("docker", "compose", "version").CombinedOutput()
		return string(out), err
	default:
		return "", ErrComposeNotFound
	}
}
