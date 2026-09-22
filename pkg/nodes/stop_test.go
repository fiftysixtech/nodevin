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
	"errors"
	"strings"
	"testing"

	"github.com/fiftysixcrypto/nodevin/internal/testutil"
	"github.com/fiftysixcrypto/nodevin/pkg/docker"
	"github.com/spf13/viper"
)

// Regression: `stop` used to look for the compose file in a hardcoded
// ~/.nodevin/data and ignore --data-dir. It then logged an error, exited 0 and
// left the node running.
func TestStopNode_HonorsDataDir(t *testing.T) {
	dataDir := useDataDir(t)
	composeFile := writeComposeFile(t, dataDir, "ipfs")
	fake := installFakeDocker(t, true, false)

	if err := stopNode("ipfs"); err != nil {
		t.Fatalf("stopNode() error: %v", err)
	}

	if want := "docker-compose -f " + composeFile + " down"; !strings.Contains(fake.calls(t), want) {
		t.Errorf("expected %q to be run, calls were:\n%s", want, fake.calls(t))
	}
}

func TestStopNode_UsesTheTestnetComposeFile(t *testing.T) {
	dataDir := useDataDir(t)
	mainnet := writeComposeFile(t, dataDir, "bitcoin-core")
	testnet := writeComposeFile(t, dataDir, "bitcoin-core-testnet")
	fake := installFakeDocker(t, true, false)

	viper.Set("testnet", true)
	if err := stopNode("bitcoin"); err != nil {
		t.Fatalf("stopNode() error: %v", err)
	}

	calls := fake.calls(t)
	if !strings.Contains(calls, testnet+" down") {
		t.Errorf("expected the testnet compose file to be stopped, calls:\n%s", calls)
	}
	if strings.Contains(calls, mainnet) {
		t.Errorf("the mainnet compose file must not be touched, calls:\n%s", calls)
	}
}

// A stop that cannot find the compose file must fail loudly, and say how to
// fix it, instead of pretending it worked.
func TestStopNode_ErrorsWhenTheComposeFileIsMissing(t *testing.T) {
	useDataDir(t)
	installFakeDocker(t, true, false)

	err := stopNode("ipfs")
	if err == nil {
		t.Fatal("expected an error when no compose file exists")
	}
	if !strings.Contains(err.Error(), "--data-dir") {
		t.Errorf("error %q should point at --data-dir", err)
	}
}

func TestStopNode_NothingRunningIsNotAnError(t *testing.T) {
	dataDir := useDataDir(t)
	writeComposeFile(t, dataDir, "ipfs")
	fake := installFakeDocker(t, false, false)

	if err := stopNode("ipfs"); err != nil {
		t.Fatalf("stopNode() error: %v", err)
	}
	if strings.Contains(fake.calls(t), " down") {
		t.Errorf("compose down must not run when nothing is running, calls:\n%s", fake.calls(t))
	}
}

func TestStopNode_UnsupportedNetwork(t *testing.T) {
	useDataDir(t)
	installFakeDocker(t, true, false)

	if err := stopNode("not-a-network"); err == nil {
		t.Fatal("expected an error for an unsupported network")
	}
}

// Regression: on a stock Linux Docker Engine there is no `docker-compose`
// binary, only the `docker compose` plugin, and stop used to fail with
// `exec: "docker-compose": executable file not found in $PATH`.
func TestStopNode_WorksWithTheComposePluginOnly(t *testing.T) {
	dataDir := useDataDir(t)
	composeFile := writeComposeFile(t, dataDir, "ipfs")

	calls := t.TempDir() + "/calls.txt"
	testutil.FakeBins(t, map[string]string{
		"docker": `echo "docker $*" >> ` + calls + "\n" +
			`case "$*" in *"ps -q"*) echo abc123 ;; esac`,
	})

	if err := stopNode("ipfs"); err != nil {
		t.Fatalf("stopNode() error: %v", err)
	}

	got := readFile(t, calls)
	if want := "docker compose -f " + composeFile + " down"; !strings.Contains(got, want) {
		t.Errorf("expected %q, calls:\n%s", want, got)
	}
}

func TestStopNode_ErrorsWhenThereIsNoComposeAtAll(t *testing.T) {
	dataDir := useDataDir(t)
	writeComposeFile(t, dataDir, "ipfs")
	testutil.FakeBins(t, map[string]string{
		"docker": `[ "$1" = compose ] && exit 1; exit 0`,
	})

	if err := stopNode("ipfs"); !errors.Is(err, docker.ErrComposeNotFound) {
		t.Errorf("err = %v, want ErrComposeNotFound", err)
	}
}

func TestStopCommand_ReturnsTheError(t *testing.T) {
	useDataDir(t)
	installFakeDocker(t, true, false)

	if err := stopNodeCmd.RunE(stopNodeCmd, []string{"ipfs"}); err == nil {
		t.Error("`stop` must return an error (so the process exits non-zero) when it cannot stop the node")
	}
	if err := stopNodeCmd.RunE(stopNodeCmd, nil); err == nil {
		t.Error("`stop` with no network must return an error")
	}
}
