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

package utils

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

// TestNetworkInfoMapInvariants asserts structural rules every registered
// network must satisfy. Several call sites depend on these implicitly, so a
// new chain (e.g. Base, Arbitrum) that breaks one fails here first instead of
// misbehaving at runtime.
func TestNetworkInfoMapInvariants(t *testing.T) {
	seenContainers := make(map[string]string)

	for network, info := range networkInfoMap {
		t.Run(network, func(t *testing.T) {
			if info.ContainerName == "" {
				t.Error("ContainerName is empty")
			}
			if info.DockerHubImage == "" {
				t.Error("DockerHubImage is empty")
			}
			if info.RPCPort <= 0 {
				t.Errorf("RPCPort = %d, want > 0", info.RPCPort)
			}
			if info.StartMessage == "" {
				t.Error("StartMessage is empty")
			}

			if other, dup := seenContainers[info.ContainerName]; dup {
				t.Errorf("ContainerName %q is used by both %q and %q", info.ContainerName, other, network)
			}
			seenContainers[info.ContainerName] = network

			if !strings.HasSuffix(network, "-testnet") {
				return
			}

			// delete.go and stop_node.go derive a testnet container/directory
			// name by appending "-testnet" to the mainnet container name, so
			// the registry must follow the same convention.
			mainnet, ok := networkInfoMap[strings.TrimSuffix(network, "-testnet")]
			if !ok {
				t.Fatalf("testnet network %q has no mainnet counterpart", network)
			}
			if want := mainnet.ContainerName + "-testnet"; info.ContainerName != want {
				t.Errorf("ContainerName = %q, want %q (mainnet name + \"-testnet\")", info.ContainerName, want)
			}
			if info.CommandSupported {
				t.Error("testnet networks are started via --testnet, so CommandSupported should be false")
			}
			if info.DockerHubImage != mainnet.DockerHubImage {
				t.Errorf("DockerHubImage = %q, want the mainnet image %q", info.DockerHubImage, mainnet.DockerHubImage)
			}
		})
	}
}

func TestNetworkAccessors(t *testing.T) {
	for network, info := range networkInfoMap {
		if got, ok := GetDefaultLocalMappedContainerName(network); !ok || got != info.ContainerName {
			t.Errorf("GetDefaultLocalMappedContainerName(%q) = (%q, %v), want (%q, true)", network, got, ok, info.ContainerName)
		}
		if got, ok := GetStartMessage(network); !ok || got != info.StartMessage {
			t.Errorf("GetStartMessage(%q) = (%q, %v)", network, got, ok)
		}
		if got, ok := GetNetworkRequiredDataSize(network); !ok || got != info.DataSize {
			t.Errorf("GetNetworkRequiredDataSize(%q) = (%d, %v), want (%d, true)", network, got, ok, info.DataSize)
		}
		if got, ok := GetNetworkRequiredSnapshotSize(network); !ok || got != info.SnapshotSize {
			t.Errorf("GetNetworkRequiredSnapshotSize(%q) = (%d, %v), want (%d, true)", network, got, ok, info.SnapshotSize)
		}
		if got, ok := GetSnapshotCIDByNetwork(network); !ok || got != info.SnapshotCID {
			t.Errorf("GetSnapshotCIDByNetwork(%q) = (%q, %v), want (%q, true)", network, got, ok, info.SnapshotCID)
		}
		if got, ok := GetFiftysixDockerhubContainerName(network); !ok || got != "fiftysix/"+info.DockerHubImage {
			t.Errorf("GetFiftysixDockerhubContainerName(%q) = (%q, %v), want (%q, true)", network, got, ok, "fiftysix/"+info.DockerHubImage)
		}
		if got := NetworkContainerMap()[network]; got != info.ContainerName {
			t.Errorf("NetworkContainerMap()[%q] = %q, want %q", network, got, info.ContainerName)
		}
		if got := NetworkDefaultRPCPorts()[network]; got != info.RPCPort {
			t.Errorf("NetworkDefaultRPCPorts()[%q] = %d, want %d", network, got, info.RPCPort)
		}
	}
}

func TestNetworkAccessorsUnknownNetwork(t *testing.T) {
	const unknown = "not-a-real-network"

	if _, ok := GetDefaultLocalMappedContainerName(unknown); ok {
		t.Error("GetDefaultLocalMappedContainerName reported an unknown network as existing")
	}
	if _, ok := GetStartMessage(unknown); ok {
		t.Error("GetStartMessage reported an unknown network as existing")
	}
	if _, ok := GetNetworkRequiredDataSize(unknown); ok {
		t.Error("GetNetworkRequiredDataSize reported an unknown network as existing")
	}
	if _, ok := GetNetworkRequiredSnapshotSize(unknown); ok {
		t.Error("GetNetworkRequiredSnapshotSize reported an unknown network as existing")
	}
	if _, ok := GetSnapshotCIDByNetwork(unknown); ok {
		t.Error("GetSnapshotCIDByNetwork reported an unknown network as existing")
	}
	if _, ok := GetFiftysixDockerhubContainerName(unknown); ok {
		t.Error("GetFiftysixDockerhubContainerName reported an unknown network as existing")
	}
}

func TestGetAllSupportedNetworks(t *testing.T) {
	got := strings.Split(GetAllSupportedNetworks(), ", ")

	if len(got) != len(networkInfoMap) {
		t.Errorf("GetAllSupportedNetworks() returned %d networks, want %d", len(got), len(networkInfoMap))
	}
	if !sort.StringsAreSorted(got) {
		t.Errorf("GetAllSupportedNetworks() is not sorted: %v", got)
	}
	for _, network := range got {
		if _, ok := networkInfoMap[network]; !ok {
			t.Errorf("GetAllSupportedNetworks() returned unregistered network %q", network)
		}
	}
}

func TestGetCommandSupportedNetworks(t *testing.T) {
	var want []string
	for network, info := range networkInfoMap {
		if info.CommandSupported {
			want = append(want, network)
		}
	}
	sort.Strings(want)

	got := GetCommandSupportedNetworks()
	if got != strings.Join(want, ", ") {
		t.Errorf("GetCommandSupportedNetworks() = %q, want %q", got, strings.Join(want, ", "))
	}
	if strings.Contains(got, "testnet") {
		t.Errorf("GetCommandSupportedNetworks() should not list testnet networks: %q", got)
	}
}

func TestGetSizeDescription(t *testing.T) {
	const (
		KB = int64(1024)
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
		PB = TB * 1024
	)

	cases := []struct {
		name string
		size int64
		want string
	}{
		{"zero is unknown", 0, "unknown (do you have proper permissions?)"},
		{"negative is unknown", -1, "unknown (do you have proper permissions?)"},
		{"bytes", 512, "512 B"},
		{"just under 1 KB", KB - 1, "1023 B"},
		{"exactly 1 KB", KB, "1.00 KB"},
		{"1.5 KB", KB + KB/2, "1.50 KB"},
		{"exactly 1 MB", MB, "1.00 MB"},
		{"exactly 1 GB", GB, "1.00 GB"},
		{"660 GB (bitcoin-sized)", 660 * GB, "660.00 GB"},
		{"exactly 1 TB", TB, "1.00 TB"},
		{"exactly 1 PB", PB, "1.00 PB"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := GetSizeDescription(c.size); got != c.want {
				t.Errorf("GetSizeDescription(%d) = %q, want %q", c.size, got, c.want)
			}
		})
	}
}

func TestCheckIfTestnetOrTestnetNetworkFlag(t *testing.T) {
	cases := []struct {
		name    string
		testnet bool
		network string
		want    bool
	}{
		{"neither set", false, "", false},
		{"testnet flag", true, "", true},
		{"network=testnet", false, "testnet", true},
		{"both set", true, "testnet", true},
		{"unrelated network name", false, "goerli", false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			viper.Set("testnet", c.testnet)
			viper.Set("network", c.network)
			t.Cleanup(func() {
				viper.Set("testnet", false)
				viper.Set("network", "")
			})

			if got := CheckIfTestnetOrTestnetNetworkFlag(); got != c.want {
				t.Errorf("CheckIfTestnetOrTestnetNetworkFlag() = %v, want %v", got, c.want)
			}
		})
	}
}

func TestIsSupportedExtendedInfoSoftware(t *testing.T) {
	cases := map[string]bool{
		"bitcoin-core":  true,
		"litecoin-core": true,
		"dogecoin-core": true,
		"core-geth":     true,
		"ord":           false,
		"kubo":          false,
		"ipfs-cluster":  false,
		"":              false,
	}

	for software, want := range cases {
		if got := IsSupportedExtendedInfoSoftware(software); got != want {
			t.Errorf("IsSupportedExtendedInfoSoftware(%q) = %v, want %v", software, got, want)
		}
	}
}

func TestFindComposeFile(t *testing.T) {
	dataParent := t.TempDir()
	viper.Set("data-dir", dataParent)
	t.Cleanup(func() { viper.Set("data-dir", "") })

	dataDir, err := GetNodevinDataDir()
	if err != nil {
		t.Fatal(err)
	}

	t.Run("finds the file in the configured data dir", func(t *testing.T) {
		want := filepath.Join(dataDir, "docker-compose_ipfs.yml")
		if err := os.WriteFile(want, []byte("services: {}\n"), 0644); err != nil {
			t.Fatal(err)
		}
		got, err := FindComposeFile("ipfs")
		if err != nil {
			t.Fatalf("FindComposeFile() error: %v", err)
		}
		if got != want {
			t.Errorf("FindComposeFile() = %q, want %q", got, want)
		}
	})

	t.Run("does not look in the default home data dir when --data-dir is set", func(t *testing.T) {
		if _, err := FindComposeFile("no-such-node"); err == nil {
			t.Fatal("expected an error for a compose file that does not exist")
		}
	})

	t.Run("the error says how to fix it", func(t *testing.T) {
		_, err := FindComposeFile("no-such-node")
		if err == nil {
			t.Fatal("expected an error")
		}
		for _, want := range []string{"docker-compose_no-such-node.yml", "--data-dir", dataDir} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("error %q should mention %q", err, want)
			}
		}
	})

	t.Run("falls back to the current directory for files older versions wrote there", func(t *testing.T) {
		cwd := t.TempDir()
		old, _ := os.Getwd()
		if err := os.Chdir(cwd); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { os.Chdir(old) })

		want := filepath.Join(cwd, "docker-compose_legacy.yml")
		if err := os.WriteFile(want, []byte("services: {}\n"), 0644); err != nil {
			t.Fatal(err)
		}
		got, err := FindComposeFile("legacy")
		if err != nil {
			t.Fatalf("FindComposeFile() error: %v", err)
		}
		// t.TempDir on macOS is under a symlinked /var, so compare resolved paths.
		gotResolved, _ := filepath.EvalSymlinks(got)
		wantResolved, _ := filepath.EvalSymlinks(want)
		if gotResolved != wantResolved {
			t.Errorf("FindComposeFile() = %q, want %q", got, want)
		}
	})
}
