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

package compose

import (
	"strings"
	"testing"

	"github.com/fiftysixcrypto/nodevin/internal/testutil"
	"github.com/spf13/viper"
)

func resetL1Flags(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		viper.Set("l1-execution-rpc-url", "")
		viper.Set("l1-beacon-url", "")
		viper.Set("execution-client", "")
		viper.Set("consensus-client", "")
	})
}

func TestResolveL1Endpoints_ExternalBothSet(t *testing.T) {
	resetL1Flags(t)
	viper.Set("l1-execution-rpc-url", "https://l1.example.com/rpc/")
	viper.Set("l1-beacon-url", "https://l1.example.com/beacon/")

	l1, err := ResolveL1Endpoints(false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if l1.ExecutionRPCURL != "https://l1.example.com/rpc" {
		t.Errorf("ExecutionRPCURL = %q, want trailing slash trimmed", l1.ExecutionRPCURL)
	}
	if l1.BeaconURL != "https://l1.example.com/beacon" {
		t.Errorf("BeaconURL = %q, want trailing slash trimmed", l1.BeaconURL)
	}
	if l1.DockerNetwork != "" {
		t.Errorf("DockerNetwork = %q, want empty for an external L1", l1.DockerNetwork)
	}
}

func TestResolveL1Endpoints_ExternalInvalidURL(t *testing.T) {
	resetL1Flags(t)
	viper.Set("l1-execution-rpc-url", "not-a-url")
	viper.Set("l1-beacon-url", "https://l1.example.com/beacon")

	if _, err := ResolveL1Endpoints(false); err == nil {
		t.Fatal("expected an error for an invalid --l1-execution-rpc-url, got nil")
	}
}

func TestResolveL1Endpoints_OnlyOneFlagSetIsAnError(t *testing.T) {
	resetL1Flags(t)
	viper.Set("l1-execution-rpc-url", "https://l1.example.com/rpc")

	_, err := ResolveL1Endpoints(false)
	if err == nil {
		t.Fatal("expected an error when only one of the two L1 flags is set")
	}
	if !strings.Contains(err.Error(), "must both be set") {
		t.Errorf("error = %q, want it to mention both flags must be set together", err.Error())
	}
}

func TestResolveL1Endpoints_AutoAttachNeedsAConsensusClient(t *testing.T) {
	resetL1Flags(t)
	viper.Set("consensus-client", "none")

	_, err := ResolveL1Endpoints(false)
	if err == nil {
		t.Fatal("expected an error when auto-attaching with --consensus-client=none")
	}
}

func TestResolveL1Endpoints_AutoAttachRequiresARunningL1(t *testing.T) {
	resetL1Flags(t)
	testutil.FakeBins(t, map[string]string{"docker": `case "$1" in ps) printf '' ;; esac`})

	_, err := ResolveL1Endpoints(false)
	if err == nil {
		t.Fatal("expected an error when nodevin's own Ethereum node is not running")
	}
	if !strings.Contains(err.Error(), "reth") || !strings.Contains(err.Error(), "lighthouse") {
		t.Errorf("error = %q, want it to name the missing reth/lighthouse containers", err.Error())
	}
}

func TestResolveL1Endpoints_AutoAttachMainnet(t *testing.T) {
	resetL1Flags(t)
	testutil.FakeBins(t, map[string]string{"docker": `case "$1" in ps) printf 'reth\nlighthouse\n' ;; esac`})

	l1, err := ResolveL1Endpoints(false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if l1.ExecutionRPCURL != "http://reth:8545" {
		t.Errorf("ExecutionRPCURL = %q, want http://reth:8545", l1.ExecutionRPCURL)
	}
	if l1.BeaconURL != "http://lighthouse:5052" {
		t.Errorf("BeaconURL = %q, want http://lighthouse:5052", l1.BeaconURL)
	}
	if l1.DockerNetwork != "ethereum-net" {
		t.Errorf("DockerNetwork = %q, want ethereum-net", l1.DockerNetwork)
	}
}

func TestResolveL1Endpoints_AutoAttachTestnetWithCustomClients(t *testing.T) {
	resetL1Flags(t)
	viper.Set("execution-client", "geth")
	viper.Set("consensus-client", "teku")
	testutil.FakeBins(t, map[string]string{"docker": `case "$1" in ps) printf 'geth-testnet\nteku-testnet\n' ;; esac`})

	l1, err := ResolveL1Endpoints(true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if l1.ExecutionRPCURL != "http://geth-testnet:8545" {
		t.Errorf("ExecutionRPCURL = %q, want http://geth-testnet:8545", l1.ExecutionRPCURL)
	}
	if l1.BeaconURL != "http://teku-testnet:5051" {
		t.Errorf("BeaconURL = %q, want http://teku-testnet:5051 (Teku's own port)", l1.BeaconURL)
	}
	if l1.DockerNetwork != "ethereum-testnet-net" {
		t.Errorf("DockerNetwork = %q, want ethereum-testnet-net", l1.DockerNetwork)
	}
}

func TestResolveL1Endpoints_AutoAttachPartiallyRunningIsAnError(t *testing.T) {
	resetL1Flags(t)
	// Execution client up, consensus client not - should still refuse rather
	// than pointing Base/Arbitrum at an execution client with nothing serving
	// blobs alongside it.
	testutil.FakeBins(t, map[string]string{"docker": `case "$1" in ps) printf 'reth\n' ;; esac`})

	_, err := ResolveL1Endpoints(false)
	if err == nil {
		t.Fatal("expected an error when only the execution client is running")
	}
	if strings.Contains(err.Error(), "reth,") || !strings.Contains(err.Error(), "lighthouse") {
		t.Errorf("error = %q, want it to name only the missing lighthouse container", err.Error())
	}
}
