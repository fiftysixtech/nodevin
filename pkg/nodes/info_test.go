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

import "testing"

func TestParseRPCCount(t *testing.T) {
	cases := []struct {
		name      string
		result    interface{}
		wantCount int
		wantOk    bool
	}{
		{"bitcoin-style plain number", float64(884213), 884213, true},
		{"bitcoin-style zero", float64(0), 0, true},
		{"geth-style hex string", "0x1a2b", 6699, true},
		{"geth-style hex zero", "0x0", 0, true},
		{"geth-style hex without leading zero handling", "0xf", 15, true},
		{"invalid hex string", "not-hex", 0, false},
		{"unsupported type", nil, 0, false},
		{"unsupported type bool", true, 0, false},
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

func TestIsEthereumStyleRPC(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		{"core-geth", true},
		{"core-geth-testnet", true},
		{"ethereum-classic", true},
		{"ethereum-classic-testnet", true},
		{"bitcoin-core", false},
		{"litecoin-core", false},
		{"dogecoin-core", false},
		{"dogecoin-core-testnet", false},
		{"bitcoin", false},
		{"", false},
	}

	for _, c := range cases {
		if got := isEthereumStyleRPC(c.name); got != c.want {
			t.Errorf("isEthereumStyleRPC(%q) = %v, want %v", c.name, got, c.want)
		}
	}
}
