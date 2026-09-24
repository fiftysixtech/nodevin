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
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

// setupResolve isolates the data dir and the --execution-client flag, fakes
// `docker ps`, and creates a data directory for each named client.
func setupResolve(t *testing.T, flag string, running []string, dataDirs ...string) {
	t.Helper()
	viper.Set("data-dir", t.TempDir())
	viper.Set("execution-client", flag)
	t.Cleanup(func() {
		viper.Set("data-dir", "")
		viper.Set("execution-client", "")
	})

	orig := listRunningContainers
	listRunningContainers = func() ([]string, error) { return running, nil }
	t.Cleanup(func() { listRunningContainers = orig })

	base, err := GetNodevinDataDir()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range dataDirs {
		if err := os.MkdirAll(filepath.Join(base, name), 0755); err != nil {
			t.Fatal(err)
		}
	}
}

func TestResolveContainerName(t *testing.T) {
	cases := []struct {
		name     string
		flag     string
		running  []string
		dataDirs []string
		want     string
		wantErr  string
	}{
		{name: "nothing anywhere falls back to the default", want: "reth"},
		{name: "explicit flag wins over a running client", flag: "geth", running: []string{"erigon"}, want: "geth"},
		{name: "explicit flag wins over data on disk", flag: "besu", dataDirs: []string{"geth"}, want: "besu"},
		{name: "invalid flag is rejected", flag: "notaclient", wantErr: "unsupported --execution-client"},
		{name: "single running client", running: []string{"bitcoin-core", "geth", "lighthouse"}, want: "geth"},
		{name: "running beats data on disk", running: []string{"nethermind"}, dataDirs: []string{"reth", "geth"}, want: "nethermind"},
		{name: "two running clients are ambiguous", running: []string{"geth", "erigon"}, wantErr: "are running"},
		{name: "stopped client found by its data dir", dataDirs: []string{"erigon"}, want: "erigon"},
		{name: "two data dirs are ambiguous", dataDirs: []string{"reth", "geth"}, wantErr: "have data on disk"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			setupResolve(t, c.flag, c.running, c.dataDirs...)

			got, err := ResolveContainerName("ethereum")
			if c.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), c.wantErr) {
					t.Fatalf("error = %v, want one containing %q", err, c.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != c.want {
				t.Errorf("ResolveContainerName(ethereum) = %q, want %q", got, c.want)
			}
		})
	}
}

func TestResolveContainerName_AmbiguityIsTyped(t *testing.T) {
	setupResolve(t, "", nil, "reth", "geth")
	_, err := ResolveContainerName("ethereum")

	var ambiguous *AmbiguousClientError
	if !errors.As(err, &ambiguous) {
		t.Fatalf("error = %v, want *AmbiguousClientError", err)
	}
	if ambiguous.Running || len(ambiguous.Candidates) != 2 {
		t.Errorf("got %+v, want two on-disk candidates", ambiguous)
	}
}

func TestResolveContainerName_IgnoresDockerFailure(t *testing.T) {
	setupResolve(t, "", nil, "geth")
	listRunningContainers = func() ([]string, error) { return nil, errors.New("docker not available") }

	if got, err := ResolveContainerName("ethereum"); err != nil || got != "geth" {
		t.Errorf("ResolveContainerName = (%q, %v), want (geth, nil)", got, err)
	}
}

// Networks with a single possible container must never depend on docker or on
// the --execution-client flag.
func TestResolveContainerName_OtherNetworksUnchanged(t *testing.T) {
	setupResolve(t, "geth", []string{"geth"}, "geth")

	for network, info := range networkInfoMap {
		if len(info.AlternateContainerNames) > 0 {
			continue
		}
		got, err := ResolveContainerName(network)
		if err != nil || got != info.ContainerName {
			t.Errorf("ResolveContainerName(%q) = (%q, %v), want (%q, nil)", network, got, err, info.ContainerName)
		}
	}

	if _, err := ResolveContainerName("no-such-network"); err == nil {
		t.Error("expected an error for an unknown network")
	}
}

func TestExplicitContainerName(t *testing.T) {
	t.Run("requires the flag and reports what it found", func(t *testing.T) {
		setupResolve(t, "", nil, "reth", "geth")
		_, err := ExplicitContainerName("ethereum")
		if err == nil || !strings.Contains(err.Error(), "--execution-client") || !strings.Contains(err.Error(), "reth, geth") {
			t.Fatalf("error = %v, want one naming the flag and the data found", err)
		}
	})

	t.Run("never infers even a single running client", func(t *testing.T) {
		setupResolve(t, "", []string{"geth"}, "geth")
		if _, err := ExplicitContainerName("ethereum"); err == nil {
			t.Fatal("expected an error without the flag")
		}
	})

	t.Run("accepts the flag", func(t *testing.T) {
		setupResolve(t, "erigon", nil)
		if got, err := ExplicitContainerName("ethereum"); err != nil || got != "erigon" {
			t.Errorf("got (%q, %v), want (erigon, nil)", got, err)
		}
	})

	t.Run("other networks need no flag", func(t *testing.T) {
		setupResolve(t, "", nil)
		if got, err := ExplicitContainerName("bitcoin"); err != nil || got != "bitcoin-core" {
			t.Errorf("got (%q, %v), want (bitcoin-core, nil)", got, err)
		}
	})
}

func TestAllContainerNamesCoversAlternates(t *testing.T) {
	all := make(map[string]bool)
	for _, name := range AllContainerNames() {
		all[name] = true
	}
	for _, want := range []string{"reth", "geth", "erigon", "besu", "nethermind", "lighthouse", "lodestar", "bitcoin-core"} {
		if !all[want] {
			t.Errorf("AllContainerNames() is missing %q", want)
		}
	}
}

func TestStackOwnerAndComponents(t *testing.T) {
	if owner, ok := StackOwner("lighthouse"); !ok || owner != "ethereum" {
		t.Errorf("StackOwner(lighthouse) = (%q, %v), want (ethereum, true)", owner, ok)
	}
	if _, ok := StackOwner("ethereum"); ok {
		t.Error("ethereum must not be part of another stack")
	}
	want := "lighthouse,lodestar,nimbus,prysm,teku"
	if got := strings.Join(ComponentNetworks("ethereum"), ","); got != want {
		t.Errorf("ComponentNetworks(ethereum) = %s, want %s", got, want)
	}
}

func TestFindComposeFileForContainer(t *testing.T) {
	setupResolve(t, "geth", nil)
	base, _ := GetNodevinDataDir()
	gethFile := filepath.Join(base, "docker-compose_geth.yml")
	if err := os.WriteFile(gethFile, []byte("services: {}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	for _, container := range []string{"geth", "lighthouse"} {
		got, err := FindComposeFileForContainer(container)
		if err != nil || got != gethFile {
			t.Errorf("FindComposeFileForContainer(%q) = (%q, %v), want (%q, nil)", container, got, err, gethFile)
		}
	}
}
