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
	"io"
	"os"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func useExecutionClient(t *testing.T, client string) {
	t.Helper()
	viper.Set("execution-client", client)
	t.Cleanup(func() { viper.Set("execution-client", "") })
}

// The bug this guards: stop/delete resolved the registry's static "reth", so
// anyone running another execution client could not stop it.
func TestStopNode_FindsTheRunningExecutionClient(t *testing.T) {
	dataDir := useDataDir(t)
	gethFile := writeComposeFile(t, dataDir, "geth")
	fake := installFakeDocker(t, true, false)
	fake.setRunning(t, "geth", "lighthouse")

	if err := stopNode("ethereum"); err != nil {
		t.Fatalf("stopNode(ethereum) error: %v", err)
	}
	if !strings.Contains(fake.calls(t), gethFile+" down") {
		t.Errorf("expected the geth stack to be brought down, calls:\n%s", fake.calls(t))
	}
}

func TestStopNode_ConsensusClientCannotBeStoppedAlone(t *testing.T) {
	useDataDir(t)
	fake := installFakeDocker(t, true, false)

	err := stopNode("lighthouse")
	if err == nil || !strings.Contains(err.Error(), "stop ethereum") {
		t.Fatalf("error = %v, want one pointing at `stop ethereum`", err)
	}
	if strings.Contains(fake.calls(t), " down") {
		t.Errorf("nothing should have been stopped, calls:\n%s", fake.calls(t))
	}
}

func TestStopNode_AmbiguousDataButNothingRunningIsNotAnError(t *testing.T) {
	dataDir := useDataDir(t)
	makeNodeDirs(t, dataDir, "reth", "geth")
	fake := installFakeDocker(t, false, false)

	if err := stopNode("ethereum"); err != nil {
		t.Fatalf("stopNode should treat this as nothing running, got: %v", err)
	}
	if strings.Contains(fake.calls(t), " down") {
		t.Errorf("nothing should have been stopped, calls:\n%s", fake.calls(t))
	}
}

func TestStopNode_TwoRunningClientsIsAnError(t *testing.T) {
	useDataDir(t)
	fake := installFakeDocker(t, true, false)
	fake.setRunning(t, "reth", "geth")

	if err := stopNode("ethereum"); err == nil || !strings.Contains(err.Error(), "--execution-client") {
		t.Fatalf("error = %v, want one asking for --execution-client", err)
	}
}

func TestDeleteNetworkDirectory_EthereumRequiresTheClientFlag(t *testing.T) {
	dataDir := useDataDir(t)
	dirs := makeNodeDirs(t, dataDir, "reth")
	installFakeDocker(t, false, false)

	err := deleteNetworkDirectory(dataDir, "ethereum")
	if err == nil || !strings.Contains(err.Error(), "--execution-client") {
		t.Fatalf("error = %v, want one requiring --execution-client", err)
	}
	if !exists(dirs["reth"]) {
		t.Error("nothing may be deleted when the client was not named, even if only one exists")
	}
}

func TestDeleteNetworkDirectory_EthereumDeletesOnlyTheNamedClient(t *testing.T) {
	dataDir := useDataDir(t)
	dirs := makeNodeDirs(t, dataDir, "reth", "geth", "lighthouse")
	writeComposeFile(t, dataDir, "geth")
	installFakeDocker(t, true, false)
	useExecutionClient(t, "geth")

	if err := deleteNetworkDirectory(dataDir, "ethereum"); err != nil {
		t.Fatalf("deleteNetworkDirectory() error: %v", err)
	}
	if exists(dirs["geth"]) {
		t.Error("expected the geth data to be removed")
	}
	if !exists(dirs["reth"]) || !exists(dirs["lighthouse"]) {
		t.Error("reth and consensus client data must be left alone")
	}
}

func TestDeleteNetworkDirectory_ConsensusClient(t *testing.T) {
	t.Run("removes only its own data", func(t *testing.T) {
		dataDir := useDataDir(t)
		dirs := makeNodeDirs(t, dataDir, "reth", "lighthouse")
		installFakeDocker(t, false, false)

		if err := deleteNetworkDirectory(dataDir, "lighthouse"); err != nil {
			t.Fatalf("deleteNetworkDirectory() error: %v", err)
		}
		if exists(dirs["lighthouse"]) || !exists(dirs["reth"]) {
			t.Error("expected only the lighthouse data to be removed")
		}
	})

	t.Run("refuses while it is running", func(t *testing.T) {
		dataDir := useDataDir(t)
		dirs := makeNodeDirs(t, dataDir, "lighthouse")
		fake := installFakeDocker(t, true, false)
		fake.setRunning(t, "lighthouse")

		err := deleteNetworkDirectory(dataDir, "lighthouse")
		if err == nil || !strings.Contains(err.Error(), "stop ethereum") {
			t.Fatalf("error = %v, want one pointing at `stop ethereum`", err)
		}
		if !exists(dirs["lighthouse"]) {
			t.Error("data of a running consensus client must not be deleted")
		}
		if strings.Contains(fake.calls(t), " down") {
			t.Error("deleting a consensus client must not stop the stack")
		}
	})
}

// Regression for the data-loss path: `delete all` only checked the registry's
// default names, so a running geth was not noticed.
func TestDeleteAllDirectories_RefusesWhileANonDefaultExecutionClientRuns(t *testing.T) {
	dataDir := useDataDir(t)
	dirs := makeNodeDirs(t, dataDir, "geth")
	fake := installFakeDocker(t, false, false)
	fake.setRunning(t, "geth")

	err := deleteAllDirectories(dataDir)
	if err == nil || !strings.Contains(err.Error(), "geth") {
		t.Fatalf("error = %v, want a refusal naming geth", err)
	}
	if !exists(dirs["geth"]) {
		t.Error("data of a running geth must not be deleted")
	}
}

func TestStopAllNodes_IncludesNonDefaultExecutionClients(t *testing.T) {
	useDataDir(t)
	fake := installFakeDocker(t, false, false)
	fake.setRunning(t, "id111 geth", "id222 unrelated-container") // `docker ps --format "{{.ID}} {{.Names}}"`

	if err := stopAllNodes(); err != nil {
		t.Fatalf("stopAllNodes() error: %v", err)
	}
	calls := fake.calls(t)
	if !strings.Contains(calls, "docker stop id111") || strings.Contains(calls, "id222") {
		t.Fatalf("expected only geth (id111) to be stopped, calls:\n%s", calls)
	}
}

func TestDeleteNetworkDirectory_EthereumAlsoRemovesItsStackFile(t *testing.T) {
	dataDir := useDataDir(t)
	makeNodeDirs(t, dataDir, "geth", "reth")
	gethFile := writeComposeFile(t, dataDir, "geth")
	rethFile := writeComposeFile(t, dataDir, "reth")
	installFakeDocker(t, true, false)
	useExecutionClient(t, "geth")

	if err := deleteNetworkDirectory(dataDir, "ethereum"); err != nil {
		t.Fatalf("deleteNetworkDirectory() error: %v", err)
	}
	if exists(gethFile) {
		t.Error("the deleted client's stack file should be removed")
	}
	if !exists(rethFile) {
		t.Error("another client's stack file must be left alone")
	}
}

func TestDataDirEntries(t *testing.T) {
	got := dataDirEntries("ethereum")
	if len(got) != 5 || got[0] != (dataDirEntry{"ethereum (reth)", "reth"}) || got[1] != (dataDirEntry{"ethereum (geth)", "geth"}) {
		t.Errorf("dataDirEntries(ethereum) = %v, want one entry per execution client, reth first", got)
	}
	if got := dataDirEntries("bitcoin"); len(got) != 1 || got[0] != (dataDirEntry{"bitcoin", "bitcoin-core"}) {
		t.Errorf("dataDirEntries(bitcoin) = %v, want the single bitcoin-core entry", got)
	}
	if got := dataDirEntries("lighthouse"); len(got) != 1 || got[0].container != "lighthouse" {
		t.Errorf("dataDirEntries(lighthouse) = %v", got)
	}
}

func TestDisplayNodeDirectoryInfo_ListsEveryEthereumClient(t *testing.T) {
	dataDir := useDataDir(t)
	makeNodeDirs(t, dataDir, "reth", "geth", "bitcoin-core")

	r, w, _ := os.Pipe()
	orig := os.Stdout
	os.Stdout = w
	displayNodeDirectoryInfo("")
	w.Close()
	os.Stdout = orig
	out, _ := io.ReadAll(r)

	for _, want := range []string{"ethereum (reth)", "ethereum (geth)", "bitcoin "} {
		if !strings.Contains(string(out), want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(string(out), "ethereum (erigon)") {
		t.Errorf("a client with no data must not be listed:\n%s", out)
	}
}

func useTestnetFlag(t *testing.T) {
	t.Helper()
	viper.Set("testnet", true)
	t.Cleanup(func() { viper.Set("testnet", false) })
}

func TestStopNode_TestnetFindsTheRunningTestnetClient(t *testing.T) {
	dataDir := useDataDir(t)
	testnetFile := writeComposeFile(t, dataDir, "geth-testnet")
	fake := installFakeDocker(t, true, false)
	fake.setRunning(t, "geth-testnet", "lighthouse-testnet")
	useTestnetFlag(t)

	if err := stopNode("ethereum"); err != nil {
		t.Fatalf("stopNode(ethereum --testnet) error: %v", err)
	}
	if !strings.Contains(fake.calls(t), testnetFile+" down") {
		t.Errorf("expected the testnet stack to be brought down, calls:\n%s", fake.calls(t))
	}
}

// Stopping mainnet must not touch a running Sepolia stack, and vice versa.
func TestStopNode_MainnetLeavesTheTestnetStackAlone(t *testing.T) {
	dataDir := useDataDir(t)
	writeComposeFile(t, dataDir, "reth-testnet")
	fake := installFakeDocker(t, false, false)
	fake.setRunning(t, "reth-testnet")

	// Like any network that was never started, there is no mainnet stack file
	// to stop - that is an error, but it must not reach the testnet stack.
	if err := stopNode("ethereum"); err == nil {
		t.Error("expected an error: there is no mainnet stack to stop")
	}
	if strings.Contains(fake.calls(t), " down") {
		t.Errorf("nothing on mainnet is running, so nothing should be stopped, calls:\n%s", fake.calls(t))
	}
}

func TestStopNode_TestnetConsensusClientPointsAtTheTestnetStack(t *testing.T) {
	useDataDir(t)
	installFakeDocker(t, true, false)
	useTestnetFlag(t)

	err := stopNode("lighthouse")
	if err == nil || !strings.Contains(err.Error(), "stop ethereum --testnet") {
		t.Fatalf("error = %v, want one pointing at `stop ethereum --testnet`", err)
	}
}

func TestDeleteNetworkDirectory_TestnetTouchesOnlyTestnetData(t *testing.T) {
	dataDir := useDataDir(t)
	dirs := makeNodeDirs(t, dataDir, "geth", "geth-testnet", "lighthouse", "lighthouse-testnet")
	testnetFile := writeComposeFile(t, dataDir, "geth-testnet")
	mainnetFile := writeComposeFile(t, dataDir, "geth")
	installFakeDocker(t, true, false)
	useExecutionClient(t, "geth")
	useTestnetFlag(t)

	if err := deleteNetworkDirectory(dataDir, "ethereum"); err != nil {
		t.Fatalf("deleteNetworkDirectory() error: %v", err)
	}
	if exists(dirs["geth-testnet"]) || exists(testnetFile) {
		t.Error("expected the geth-testnet data and stack file to be removed")
	}
	for _, kept := range []string{"geth", "lighthouse", "lighthouse-testnet"} {
		if !exists(dirs[kept]) {
			t.Errorf("%s data must be left alone", kept)
		}
	}
	if !exists(mainnetFile) {
		t.Error("the mainnet stack file must be left alone")
	}
}

func TestDeleteNetworkDirectory_TestnetRequiresTheClientFlag(t *testing.T) {
	dataDir := useDataDir(t)
	dirs := makeNodeDirs(t, dataDir, "reth-testnet")
	installFakeDocker(t, false, false)
	useTestnetFlag(t)

	err := deleteNetworkDirectory(dataDir, "ethereum")
	if err == nil || !strings.Contains(err.Error(), "--execution-client") {
		t.Fatalf("error = %v, want one requiring --execution-client", err)
	}
	if !exists(dirs["reth-testnet"]) {
		t.Error("nothing may be deleted when the client was not named")
	}
}

func TestDeleteNetworkDirectory_TestnetConsensusClient(t *testing.T) {
	dataDir := useDataDir(t)
	dirs := makeNodeDirs(t, dataDir, "lighthouse", "lighthouse-testnet")
	installFakeDocker(t, false, false)
	useTestnetFlag(t)

	if err := deleteNetworkDirectory(dataDir, "lighthouse"); err != nil {
		t.Fatalf("deleteNetworkDirectory() error: %v", err)
	}
	if exists(dirs["lighthouse-testnet"]) || !exists(dirs["lighthouse"]) {
		t.Error("expected only the lighthouse-testnet data to be removed")
	}
}

func TestDataDirEntries_Testnet(t *testing.T) {
	got := dataDirEntries("ethereum-testnet")
	if len(got) != 5 || got[0] != (dataDirEntry{"ethereum-testnet (reth-testnet)", "reth-testnet"}) {
		t.Errorf("dataDirEntries(ethereum-testnet) = %v, want one entry per testnet execution client", got)
	}
}
