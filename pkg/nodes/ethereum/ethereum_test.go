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

package ethereum

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/fiftysixcrypto/nodevin/internal/testutil"
	"github.com/spf13/viper"
)

// setup isolates the data dir, fakes `docker ps` to report running, and sets
// the given flags. Every key is reset afterwards.
func setup(t *testing.T, running string, flags map[string]interface{}) {
	t.Helper()
	testutil.FakeBins(t, map[string]string{"docker": `case "$1" in ps) printf '` + running + `' ;; esac`})

	viper.Set("data-dir", t.TempDir())
	for key, value := range flags {
		viper.Set(key, value)
	}
	t.Cleanup(func() {
		viper.Set("data-dir", "")
		viper.Set("testnet", false)
		for key := range flags {
			viper.Set(key, "")
		}
	})
}

func TestConflictingContainers(t *testing.T) {
	cases := []struct {
		name    string
		running []string
		exec    string
		cons    string
		want    []string
	}{
		{"nothing running", nil, "reth", "lighthouse", nil},
		{"same stack already up", []string{"reth", "lighthouse"}, "reth", "lighthouse", nil},
		{"unrelated containers are ignored", []string{"bitcoin-core", "watchtower-nodevin"}, "reth", "lighthouse", nil},
		{"a different execution client", []string{"geth", "lighthouse"}, "reth", "lighthouse", []string{"geth"}},
		{"a different consensus client", []string{"reth", "prysm"}, "reth", "lighthouse", []string{"prysm"}},
		{"execution-only start with a consensus client still up", []string{"reth", "lighthouse"}, "reth", "", []string{"lighthouse"}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := conflictingContainers(c.running, c.exec, c.cons); !reflect.DeepEqual(got, c.want) {
				t.Errorf("conflictingContainers() = %v, want %v", got, c.want)
			}
		})
	}
}

func TestCreateEthereumComposeFile_ExecutionOnly(t *testing.T) {
	setup(t, "", map[string]interface{}{"execution-client": "geth", "consensus-client": "none"})

	path, err := CreateEthereumComposeFile(t.TempDir())
	if err != nil {
		t.Fatalf("CreateEthereumComposeFile() error: %v", err)
	}
	if filepath.Base(path) != "docker-compose_geth.yml" {
		t.Errorf("compose file = %s, want docker-compose_geth.yml", filepath.Base(path))
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "lighthouse") {
		t.Errorf("--consensus-client=none must not add a consensus service:\n%s", data)
	}
}

func TestCreateEthereumComposeFile_RejectsBadInput(t *testing.T) {
	cases := []struct {
		name    string
		flags   map[string]interface{}
		wantErr string
	}{
		{"unknown execution client", map[string]interface{}{"execution-client": "parity"}, "--execution-client"},
		{"unknown consensus client", map[string]interface{}{"consensus-client": "grandine"}, "--consensus-client"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			setup(t, "", c.flags)
			if _, err := CreateEthereumComposeFile(t.TempDir()); err == nil || !strings.Contains(err.Error(), c.wantErr) {
				t.Fatalf("error = %v, want one containing %q", err, c.wantErr)
			}
		})
	}
}

func TestCreateEthereumComposeFile_RefusesToReplaceARunningStack(t *testing.T) {
	setup(t, "geth\\nlighthouse\\n", map[string]interface{}{"execution-client": "reth", "consensus-client": "none"})

	_, err := CreateEthereumComposeFile(t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "geth, lighthouse") || !strings.Contains(err.Error(), "stop ethereum") {
		t.Fatalf("error = %v, want a refusal naming the running containers and `stop ethereum`", err)
	}
}

func TestCreateEthereumComposeFile_RequiresACheckpointURL(t *testing.T) {
	setup(t, "", map[string]interface{}{"consensus-client": "lighthouse", "checkpoint-sync-url": ""})

	_, err := CreateEthereumComposeFile(t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "--checkpoint-sync-url is required") {
		t.Fatalf("error = %v, want one requiring --checkpoint-sync-url", err)
	}
}

func TestCreateEthereumComposeFile_TestnetExecutionOnly(t *testing.T) {
	setup(t, "", map[string]interface{}{"testnet": true, "execution-client": "geth", "consensus-client": "none"})

	path, err := CreateEthereumComposeFile(t.TempDir())
	if err != nil {
		t.Fatalf("CreateEthereumComposeFile() error: %v", err)
	}
	if filepath.Base(path) != "docker-compose_geth-testnet.yml" {
		t.Errorf("compose file = %s, want docker-compose_geth-testnet.yml", filepath.Base(path))
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"container_name: geth-testnet", "command: --sepolia", "127.0.0.1:8549:8545", "ethereum-testnet-net"} {
		if !strings.Contains(string(data), want) {
			t.Errorf("compose file lacks %q:\n%s", want, data)
		}
	}
	if strings.Contains(string(data), "container_name: geth\n") {
		t.Errorf("a testnet stack must not use the mainnet container name:\n%s", data)
	}
}

func TestCreateEthereumComposeFile_TestnetStillNeedsACheckpointURL(t *testing.T) {
	setup(t, "", map[string]interface{}{"testnet": true, "consensus-client": "lighthouse", "checkpoint-sync-url": ""})

	if _, err := CreateEthereumComposeFile(t.TempDir()); err == nil || !strings.Contains(err.Error(), "--checkpoint-sync-url is required") {
		t.Fatalf("error = %v, want one requiring --checkpoint-sync-url", err)
	}
}

// Mainnet and Sepolia share host ports, so neither may start over the other.
func TestCreateEthereumComposeFile_TestnetRefusesToStartOverMainnet(t *testing.T) {
	setup(t, "reth\nlighthouse\n", map[string]interface{}{"testnet": true, "consensus-client": "none"})

	_, err := CreateEthereumComposeFile(t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "reth, lighthouse") || !strings.Contains(err.Error(), "stop ethereum`") || strings.Contains(err.Error(), "--testnet") {
		t.Fatalf("error = %v, want a refusal naming the mainnet stack and plain `stop ethereum`", err)
	}
}

func TestCreateEthereumComposeFile_MainnetRefusesToStartOverTestnet(t *testing.T) {
	setup(t, "reth-testnet\nlighthouse-testnet\n", map[string]interface{}{"consensus-client": "none"})

	_, err := CreateEthereumComposeFile(t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "reth-testnet, lighthouse-testnet") || !strings.Contains(err.Error(), "stop ethereum --testnet") {
		t.Fatalf("error = %v, want a refusal telling the user to stop the testnet stack", err)
	}
}

func TestConflictingContainers_KnowsBothStacks(t *testing.T) {
	cases := []struct {
		name    string
		running []string
		exec    string
		cons    string
		want    []string
	}{
		{"same testnet stack already up", []string{"reth-testnet", "lighthouse-testnet"}, "reth-testnet", "lighthouse-testnet", nil},
		{"a different testnet client", []string{"geth-testnet", "lighthouse-testnet"}, "reth-testnet", "lighthouse-testnet", []string{"geth-testnet"}},
		{"mainnet running under a testnet start", []string{"reth"}, "reth-testnet", "lighthouse-testnet", []string{"reth"}},
		{"testnet running under a mainnet start", []string{"nimbus-testnet"}, "reth", "lighthouse", []string{"nimbus-testnet"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := conflictingContainers(c.running, c.exec, c.cons); !reflect.DeepEqual(got, c.want) {
				t.Errorf("conflictingContainers() = %v, want %v", got, c.want)
			}
		})
	}
}
