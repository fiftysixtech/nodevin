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
	"testing"

	"github.com/fiftysixcrypto/nodevin/internal/utils"
)

// Every execution client the "ethereum" network can run as must be treated as
// a geth-family JSON-RPC node, on the published RPC port, by info/view/request.
func TestEthereumExecutionClients_InfoAndView(t *testing.T) {
	wantURL := fmt.Sprintf("http://127.0.0.1:%d", utils.NetworkDefaultRPCPorts()["ethereum"])

	names := append([]string{"ethereum"}, utils.CandidateContainerNames("ethereum")...)
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			if !isEthereumStyleRPC(name) {
				t.Errorf("isEthereumStyleRPC(%q) = false, want true", name)
			}
			if name == "ethereum" {
				return
			}
			if !utils.IsSupportedExtendedInfoSoftware(name) {
				t.Errorf("IsSupportedExtendedInfoSoftware(%q) = false, want true", name)
			}
			if got := getLocalEndpointByContainerName(name); got != wantURL {
				t.Errorf("getLocalEndpointByContainerName(%q) = %q, want %q", name, got, wantURL)
			}
			if got := getSoftwareNetworkName(name); got != "ethereum" {
				t.Errorf("getSoftwareNetworkName(%q) = %q, want ethereum", name, got)
			}
			if got := getNodevinName(name); got != "Ethvin" {
				t.Errorf("getNodevinName(%q) = %q, want Ethvin", name, got)
			}
		})
	}
}

// Consensus clients speak the beacon REST API, not JSON-RPC: they must not be
// picked up by the RPC-based info/view paths.
func TestConsensusClients_NotTreatedAsRPCNodes(t *testing.T) {
	for _, network := range utils.ComponentNetworks("ethereum") {
		if utils.IsSupportedExtendedInfoSoftware(network) || isEthereumStyleRPC(network) {
			t.Errorf("%s must not be handled as a JSON-RPC node", network)
		}
	}
}

// `request ethereum` reaches a geth-family server, which rejects anything but
// "jsonrpc":"2.0".
func TestMakeRequest_EthereumSendsJSONRPC2(t *testing.T) {
	for _, network := range []string{"ethereum", "geth", "reth"} {
		t.Run(network, func(t *testing.T) {
			var got map[string]interface{}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, _ := io.ReadAll(r.Body)
				json.Unmarshal(body, &got)
				w.Write([]byte(`{"jsonrpc":"2.0","id":"nodevin","result":"0x1"}`))
			}))
			defer server.Close()

			if _, err := makeRequest(network, server.URL, "eth_chainId", "", "", "", ""); err != nil {
				t.Fatal(err)
			}
			if got["jsonrpc"] != "2.0" || got["method"] != "eth_chainId" {
				t.Errorf("request body = %v, want jsonrpc 2.0 and method eth_chainId", got)
			}
		})
	}
}

func TestVersionFromEnv(t *testing.T) {
	cases := []struct {
		name  string
		image string
		env   []string
		want  string
	}{
		{"NODE_VERSION as before", "fiftysix/bitcoin-core:latest", []string{"PATH=/bin", "NODE_VERSION=31.1"}, "31.1"},
		{"NODE_VERSION wins", "fiftysix/reth:latest", []string{"RETH_VERSION=2.6.0", "NODE_VERSION=9.9"}, "9.9"},
		{"client-named", "fiftysix/reth:latest", []string{"PATH=/bin", "RETH_VERSION=2.6.0"}, "2.6.0"},
		{"client-named with CLIENT", "fiftysix/besu:latest", []string{"BESU_CLIENT_VERSION=26.8.1"}, "26.8.1"},
		{"nimbus", "fiftysix/nimbus:latest", []string{"NIMBUS_CLIENT_VERSION=26.8.0", "TARBALL_NAME=x"}, "26.8.0"},
		{"lodestar's bare CLIENT_VERSION", "fiftysix/lodestar:latest", []string{"CLIENT_VERSION=1.48.0"}, "1.48.0"},
		{"other vendors' version vars are ignored", "fiftysix/teku:latest", []string{"JAVA_VERSION=26", "TEKU_CLIENT_VERSION=26.9.0"}, "26.9.0"},
		{"nothing found", "fiftysix/geth:latest", []string{"PATH=/bin", "JAVA_VERSION=26"}, "unknown"},
		{"empty image", "", []string{"PATH=/bin"}, "unknown"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := versionFromEnv(c.image, c.env); got != c.want {
				t.Errorf("versionFromEnv(%q, %v) = %q, want %q", c.image, c.env, got, c.want)
			}
		})
	}
}
