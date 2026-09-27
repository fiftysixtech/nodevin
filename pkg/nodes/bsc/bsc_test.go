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

package bsc

import (
	"os"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func reset(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		viper.Set("data-dir", "")
		viper.Set("testnet", false)
	})
}

func TestCreateBscComposeFile_Mainnet(t *testing.T) {
	reset(t)
	viper.Set("data-dir", t.TempDir())

	path, err := CreateBscComposeFile(t.TempDir())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read generated compose file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "container_name: bsc\n") {
		t.Errorf("compose file does not use the bsc container name:\n%s", content)
	}
	if !strings.Contains(content, "NETWORK: mainnet") {
		t.Errorf("compose file does not select the mainnet network:\n%s", content)
	}
}

func TestCreateBscComposeFile_Testnet(t *testing.T) {
	reset(t)
	viper.Set("data-dir", t.TempDir())
	viper.Set("testnet", true)

	path, err := CreateBscComposeFile(t.TempDir())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read generated compose file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "bsc-testnet") {
		t.Errorf("compose file does not use the testnet container name:\n%s", content)
	}
	if !strings.Contains(content, "NETWORK: testnet") {
		t.Errorf("compose file does not select the testnet network:\n%s", content)
	}
}
