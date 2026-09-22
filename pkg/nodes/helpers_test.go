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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fiftysixcrypto/nodevin/internal/utils"
)

// info_test.go covers the happy paths of parseRPCCount; these are the edges
// around it. A few document current leniency/strictness on purpose so a
// future change to either is a conscious one.
func TestParseRPCCount_EdgeCases(t *testing.T) {
	cases := []struct {
		name      string
		result    interface{}
		wantCount int
		wantOk    bool
	}{
		{"bare 0x prefix with no digits", "0x", 0, false},
		{"empty string", "", 0, false},
		{"hex without 0x prefix is accepted", "1a2b", 6699, true},
		{"uppercase 0X prefix is rejected", "0X1a", 0, false},
		{"negative hex is rejected", "-0x1", 0, false},
		{"fractional number truncates", float64(1.9), 1, true},
		{"large block height", float64(23_000_000), 23_000_000, true},
		{"large hex block height", "0x1000000", 16_777_216, true},
		{"integer type is unsupported (json only yields float64)", 5, 0, false},
		{"slice is unsupported", []interface{}{1}, 0, false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := parseRPCCount(c.result)
			if ok != c.wantOk {
				t.Fatalf("parseRPCCount(%v) ok = %v, want %v", c.result, ok, c.wantOk)
			}
			if ok && got != c.wantCount {
				t.Errorf("parseRPCCount(%v) = %d, want %d", c.result, got, c.wantCount)
			}
		})
	}
}

// TestIsEthereumStyleRPC_MatchesRegistry ties the hand-maintained
// isEthereumStyleRPC switch to internal/utils' registry, so adding an
// EVM-style chain there without updating this switch (which would make
// `info`/`request` speak Bitcoin-style RPC to it) is caught.
func TestIsEthereumStyleRPC_MatchesRegistry(t *testing.T) {
	for _, network := range []string{"ethereum-classic", "ethereum-classic-testnet"} {
		container, ok := utils.GetDefaultLocalMappedContainerName(network)
		if !ok {
			t.Fatalf("%q is not registered", network)
		}
		if !isEthereumStyleRPC(network) {
			t.Errorf("isEthereumStyleRPC(%q) = false, want true", network)
		}
		if !isEthereumStyleRPC(container) {
			t.Errorf("isEthereumStyleRPC(%q) = false, want true (container name of %q)", container, network)
		}
	}
}

// TestLocalEndpointMatchesRegistryPort asserts that for every chain that
// `info` queries over RPC, the hard-coded local endpoint's port equals the RPC
// port in the registry. They're maintained separately, so they can drift.
func TestLocalEndpointMatchesRegistryPort(t *testing.T) {
	for network, container := range utils.NetworkContainerMap() {
		if !utils.IsSupportedExtendedInfoSoftware(container) && !isTestnetOfExtendedInfo(container) {
			continue
		}

		want := fmt.Sprintf("http://127.0.0.1:%d", utils.NetworkDefaultRPCPorts()[network])
		if got := getLocalEndpointByContainerName(container); got != want {
			t.Errorf("getLocalEndpointByContainerName(%q) = %q, want %q (registry RPC port for %q)", container, got, want, network)
		}
	}
}

func isTestnetOfExtendedInfo(container string) bool {
	return strings.HasSuffix(container, "-testnet") &&
		utils.IsSupportedExtendedInfoSoftware(strings.TrimSuffix(container, "-testnet"))
}

func TestGetLocalEndpointByContainerName_UnknownFallsBackToBareHost(t *testing.T) {
	if got := getLocalEndpointByContainerName("something-else"); got != "http://127.0.0.1" {
		t.Errorf("got %q, want the bare host", got)
	}
}

func TestGetGlobalEndpointByContainerName(t *testing.T) {
	cases := map[string]bool{
		"bitcoin-core":          true,
		"bitcoin-core-testnet":  true,
		"litecoin-core":         true,
		"dogecoin-core":         true,
		"litecoin-core-testnet": false,
		"dogecoin-core-testnet": false,
		"core-geth":             false,
		"unknown":               false,
	}

	for container, wantURL := range cases {
		got := getGlobalEndpointByContainerName(container)
		if wantURL && got == "" {
			t.Errorf("getGlobalEndpointByContainerName(%q) = \"\", want a URL", container)
		}
		if !wantURL && got != "" {
			t.Errorf("getGlobalEndpointByContainerName(%q) = %q, want \"\"", container, got)
		}
	}
}

func TestFormatPorts(t *testing.T) {
	cases := []struct {
		name  string
		ports string
		want  string
	}{
		{"empty", "", ""},
		{"single mapping", "0.0.0.0:8332->8332/tcp", "8332"},
		{"dedupes host and container port", "0.0.0.0:8332->8332/tcp, :::8332->8332/tcp", "8332"},
		{"multiple distinct ports keep order", "0.0.0.0:8332->8332/tcp, 0.0.0.0:8333->8333/tcp", "8332, 8333"},
		{"tcp and udp on the same port collapse", "0.0.0.0:30303->30303/tcp, 0.0.0.0:30303->30303/udp", "30303"},
		{"port range", "0.0.0.0:4001-4002->4001-4002/tcp", "4001-4002"},
		// Regression: a loopback-bound port (introduced by publishing RPC/API
		// ports as 127.0.0.1:PORT:PORT in #24/#25) used to have its host-IP
		// octets misread as extra ports, e.g. "127.0.0.1:5001->5001/tcp"
		// produced "127, 1, 5001" instead of just "5001". Captured verbatim
		// from a real `docker ps` on a running ipfs container.
		{"loopback-bound port", "127.0.0.1:5001->5001/tcp", "5001"},
		{"real ipfs container: mixed loopback and public ports", "127.0.0.1:5001->5001/tcp, 0.0.0.0:4001->4001/tcp, 127.0.0.1:8080->8080/tcp", "5001, 4001, 8080"},
		{"unpublished port has no arrow", "4001/tcp", "4001"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := formatPorts(c.ports); got != c.want {
				t.Errorf("formatPorts(%q) = %q, want %q", c.ports, got, c.want)
			}
		})
	}
}

func TestExtractUptime(t *testing.T) {
	cases := []struct {
		status string
		want   int
	}{
		{"Up 3 days", 3},
		{"Up 14 days (healthy)", 14},
		{"Up 1 day", 1},
		{"Up 2 weeks", 14},
		{"Up 3 months", 90},
		{"Up 2 years", 730},
		{"Up 5 hours", 1},
		{"Up About an hour", 1},
		{"Up 2 minutes", 0},
		{"Up Less than a second", 0},
		{"Exited (0) 3 seconds ago", 0},
		{"", 0},
	}

	for _, c := range cases {
		if got := extractUptime(c.status); got != c.want {
			t.Errorf("extractUptime(%q) = %d, want %d", c.status, got, c.want)
		}
	}
}

func TestNodevinNameAndSoftwareNetworkName(t *testing.T) {
	cases := []struct {
		container   string
		wantName    string
		wantNetwork string
	}{
		{"bitcoin-core", "Bitvin", "bitcoin"},
		{"litecoin-core", "Litevin", "litecoin"},
		{"dogecoin-core", "Dogevin", "dogecoin"},
		{"core-geth", "Classicvin", "ethereum-classic"},
		{"BITCOIN-CORE", "Bitvin", ""},
		{"ord", "Nodevin", ""},
		{"", "Nodevin", ""},
	}

	for _, c := range cases {
		if got := getNodevinName(c.container); got != c.wantName {
			t.Errorf("getNodevinName(%q) = %q, want %q", c.container, got, c.wantName)
		}
		if got := getSoftwareNetworkName(c.container); got != c.wantNetwork {
			t.Errorf("getSoftwareNetworkName(%q) = %q, want %q", c.container, got, c.wantNetwork)
		}
	}
}

// Every extended-info chain must map back to a registered network, or `view`
// can't look up its size/stats.
func TestSoftwareNetworkNameIsRegistered(t *testing.T) {
	for _, container := range []string{"bitcoin-core", "litecoin-core", "dogecoin-core", "core-geth"} {
		network := getSoftwareNetworkName(container)
		got, ok := utils.GetDefaultLocalMappedContainerName(network)
		if !ok || got != container {
			t.Errorf("getSoftwareNetworkName(%q) = %q, which maps back to (%q, %v), want (%q, true)", container, network, got, ok, container)
		}
	}
}

func TestGetDirectorySize_Nodes(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a"), make([]byte, 40), 0644); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(dir, "sub")
	if err := os.MkdirAll(sub, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "b"), make([]byte, 2), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := getDirectorySize(dir)
	if err != nil {
		t.Fatalf("getDirectorySize() error: %v", err)
	}
	if got != 42 {
		t.Errorf("getDirectorySize() = %d, want 42", got)
	}

	if _, err := getDirectorySize(filepath.Join(dir, "does-not-exist")); err == nil {
		t.Error("expected an error for a missing directory")
	}
}

// TestMakeRequest_JSONRPCVersion is the regression test for PR #19: geth-family
// servers reject anything but "jsonrpc":"2.0", while Bitcoin-derived daemons
// have always been sent "1.0".
func TestMakeRequest_JSONRPCVersion(t *testing.T) {
	cases := []struct {
		network string
		want    string
	}{
		{"core-geth", "2.0"},
		{"core-geth-testnet", "2.0"},
		{"ethereum-classic", "2.0"},
		{"bitcoin-core", "1.0"},
		{"litecoin-core", "1.0"},
		{"dogecoin-core", "1.0"},
	}

	for _, c := range cases {
		t.Run(c.network, func(t *testing.T) {
			var got map[string]interface{}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, _ := io.ReadAll(r.Body)
				json.Unmarshal(body, &got)
				w.Write([]byte(`{}`))
			}))
			defer server.Close()

			if _, err := makeRequest(c.network, server.URL, "some_method", `["a",1]`, "", "", ""); err != nil {
				t.Fatalf("makeRequest() error: %v", err)
			}

			if got["jsonrpc"] != c.want {
				t.Errorf("jsonrpc = %v, want %q", got["jsonrpc"], c.want)
			}
			if got["method"] != "some_method" {
				t.Errorf("method = %v, want some_method", got["method"])
			}
			if params, _ := got["params"].([]interface{}); len(params) != 2 {
				t.Errorf("params = %v, want the 2 params passed in", got["params"])
			}
		})
	}
}

func TestMakeRequest_InvalidParamsAndHeaders(t *testing.T) {
	t.Run("invalid params JSON returns an error before any request", func(t *testing.T) {
		if _, err := makeRequest("bitcoin-core", "http://127.0.0.1:1", "m", "{not json", "", "", ""); err == nil {
			t.Fatal("expected an error for invalid params")
		}
	})

	t.Run("custom headers are forwarded", func(t *testing.T) {
		var gotHeader string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotHeader = r.Header.Get("X-Test")
			w.Write([]byte(`{}`))
		}))
		defer server.Close()

		if _, err := makeRequest("bitcoin-core", server.URL, "m", "", "X-Test: hello, Ignored", "", ""); err != nil {
			t.Fatal(err)
		}
		if gotHeader != "hello" {
			t.Errorf("X-Test header = %q, want %q", gotHeader, "hello")
		}
	})

	t.Run("non-200 includes status and body", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("boom"))
		}))
		defer server.Close()

		_, err := makeRequest("bitcoin-core", server.URL, "m", "", "", "", "")
		if err == nil {
			t.Fatal("expected an error")
		}
		if msg := err.Error(); !strings.Contains(msg, "500") && strings.Contains(msg, "boom") {
			t.Errorf("error %q should mention the status code and body", msg)
		}
	})
}
