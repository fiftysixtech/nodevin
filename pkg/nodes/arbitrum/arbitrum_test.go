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

package arbitrum

import (
	"os"
	"strings"
	"testing"

	"github.com/fiftysixcrypto/nodevin/internal/testutil"
	"github.com/spf13/viper"
)

func reset(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		viper.Set("data-dir", "")
		viper.Set("testnet", false)
		viper.Set("l1-execution-rpc-url", "")
		viper.Set("l1-beacon-url", "")
	})
}

func TestCreateArbitrumComposeFile_ExternalL1(t *testing.T) {
	reset(t)
	viper.Set("data-dir", t.TempDir())
	viper.Set("l1-execution-rpc-url", "https://l1.example.com/rpc")
	viper.Set("l1-beacon-url", "https://l1.example.com/beacon")

	path, err := CreateArbitrumComposeFile(t.TempDir())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read generated compose file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "l1.example.com") {
		t.Errorf("compose file does not reference the external L1 URL:\n%s", content)
	}
	if !strings.Contains(content, "nitro") {
		t.Errorf("compose file does not mention the nitro container:\n%s", content)
	}
}

func TestCreateArbitrumComposeFile_NoL1Configured(t *testing.T) {
	reset(t)
	viper.Set("data-dir", t.TempDir())
	testutil.FakeBins(t, map[string]string{"docker": `case "$1" in ps) printf '' ;; esac`})

	if _, err := CreateArbitrumComposeFile(t.TempDir()); err == nil {
		t.Fatal("expected an error when no L1 is configured or running")
	}
}

func TestCreateArbitrumComposeFile_AutoAttachesToLocalL1(t *testing.T) {
	reset(t)
	viper.Set("data-dir", t.TempDir())
	testutil.FakeBins(t, map[string]string{"docker": `case "$1" in ps) printf 'reth\nlighthouse\n' ;; esac`})

	path, err := CreateArbitrumComposeFile(t.TempDir())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read generated compose file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "http://reth:8545") {
		t.Errorf("compose file does not reference the local execution client by container name:\n%s", content)
	}
	if !strings.Contains(content, "http://lighthouse:5052") {
		t.Errorf("compose file does not reference the local consensus client by container name:\n%s", content)
	}
	if !strings.Contains(content, "ethereum-net") {
		t.Errorf("compose file does not join the Ethereum stack's Docker network:\n%s", content)
	}
}

func TestCreateArbitrumComposeFile_Testnet(t *testing.T) {
	reset(t)
	viper.Set("data-dir", t.TempDir())
	viper.Set("testnet", true)
	viper.Set("l1-execution-rpc-url", "https://sepolia-l1.example.com/rpc")
	viper.Set("l1-beacon-url", "https://sepolia-l1.example.com/beacon")

	path, err := CreateArbitrumComposeFile(t.TempDir())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read generated compose file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "nitro-testnet") {
		t.Errorf("compose file does not use the testnet container name:\n%s", content)
	}
	if !strings.Contains(content, "421614") || !strings.Contains(content, "11155111") {
		t.Errorf("compose file does not use Arbitrum/Ethereum Sepolia chain IDs:\n%s", content)
	}
}
