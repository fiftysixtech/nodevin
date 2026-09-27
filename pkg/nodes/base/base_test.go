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

package base

import (
	"os"
	"strings"
	"testing"

	"github.com/fiftysixcrypto/nodevin/internal/testutil"
	"github.com/spf13/viper"
)

// setup fakes `docker ps` (for ResolveL1Endpoints' auto-attach check) and
// stubs out the real image pull (pkg/docker.PullImage talks to the Docker
// Engine API directly and needs a real daemon connection a test binary
// doesn't have - see pullConsensusImage).
func setup(t *testing.T, running string) {
	t.Helper()
	testutil.FakeBins(t, map[string]string{
		"docker": `case "$1" in ps) printf '` + running + `' ;; esac`,
	})
	orig := pullConsensusImage
	pullConsensusImage = func(string) error { return nil }
	t.Cleanup(func() { pullConsensusImage = orig })

	viper.Set("data-dir", t.TempDir())
	t.Cleanup(func() {
		viper.Set("data-dir", "")
		viper.Set("testnet", false)
		viper.Set("l1-execution-rpc-url", "")
		viper.Set("l1-beacon-url", "")
	})
}

func TestCreateBaseComposeFile_ExternalL1(t *testing.T) {
	setup(t, "")
	viper.Set("l1-execution-rpc-url", "https://l1.example.com/rpc")
	viper.Set("l1-beacon-url", "https://l1.example.com/beacon")

	path, err := CreateBaseComposeFile(t.TempDir())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read generated compose file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "base-reth") || !strings.Contains(content, "base-consensus") {
		t.Errorf("compose file does not contain both base-reth and base-consensus:\n%s", content)
	}
	if !strings.Contains(content, "l1.example.com") {
		t.Errorf("compose file does not reference the external L1:\n%s", content)
	}
}

func TestCreateBaseComposeFile_NoL1Configured(t *testing.T) {
	setup(t, "")
	if _, err := CreateBaseComposeFile(t.TempDir()); err == nil {
		t.Fatal("expected an error when no L1 is configured or running")
	}
}

func TestCreateBaseComposeFile_AutoAttachesToLocalL1(t *testing.T) {
	setup(t, "reth\nlighthouse\n")

	path, err := CreateBaseComposeFile(t.TempDir())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read generated compose file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "http://reth:8545") || !strings.Contains(content, "http://lighthouse:5052") {
		t.Errorf("compose file does not reference the local L1 by container name:\n%s", content)
	}
	if !strings.Contains(content, "ethereum-net") {
		t.Errorf("compose file does not join the Ethereum stack's Docker network:\n%s", content)
	}
}

func TestCreateBaseComposeFile_SharesTheSameJWTBetweenBothContainers(t *testing.T) {
	setup(t, "reth\nlighthouse\n")

	path, err := CreateBaseComposeFile(t.TempDir())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read generated compose file: %v", err)
	}

	var secrets []string
	for _, line := range strings.Split(string(data), "\n") {
		if strings.Contains(line, "BASE_NODE_L2_ENGINE_AUTH_RAW") {
			secrets = append(secrets, strings.TrimSpace(line))
		}
	}
	if len(secrets) != 2 {
		t.Fatalf("expected BASE_NODE_L2_ENGINE_AUTH_RAW to appear twice (once per container), found %d: %v", len(secrets), secrets)
	}
	if secrets[0] != secrets[1] {
		t.Errorf("base-reth and base-consensus got different JWT secrets: %q != %q", secrets[0], secrets[1])
	}
}

func TestCreateBaseComposeFile_Testnet(t *testing.T) {
	setup(t, "reth-testnet\nlighthouse-testnet\n")
	viper.Set("testnet", true)

	path, err := CreateBaseComposeFile(t.TempDir())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read generated compose file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "base-reth-testnet") || !strings.Contains(content, "base-consensus-testnet") {
		t.Errorf("compose file does not use the testnet container names:\n%s", content)
	}
	if !strings.Contains(content, "base-sepolia") {
		t.Errorf("compose file does not select the base-sepolia chain:\n%s", content)
	}
}
