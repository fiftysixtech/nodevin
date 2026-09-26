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

import "fmt"

// GetTekuNetworkComposeConfig builds the compose config for Teku paired with
// whichever execution client --execution-client selects. See
// GetLighthouseNetworkComposeConfig for the shared reasoning.
//
// Unlike Lighthouse/Prysm/Lodestar, Teku has no "beacon"/"bn" subcommand, and
// its execution flags are --ee-endpoint / --ee-jwt-secret-file.
func GetTekuNetworkComposeConfig(network string) (NetworkConfig, error) {
	return consensusConfig("teku", network, []string{"127.0.0.1:5051:5051", "9000:9000", "9000:9000/udp"}, func(engineEndpoint, jwtPath, chain string) string {
		return fmt.Sprintf("teku --ee-endpoint %s --ee-jwt-secret-file %s --network %s", engineEndpoint, jwtPath, chain)
	})
}
