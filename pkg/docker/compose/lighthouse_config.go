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

// GetLighthouseNetworkComposeConfig builds the compose config for Lighthouse
// paired with whichever execution client --execution-client selects, on
// mainnet ("lighthouse") or Sepolia ("lighthouse-testnet").
//
// The execution client's host data directory is mounted read-only and the JWT
// flag points at the shared jwt.hex: a consensus client must never generate
// its own secret, or it would never authenticate with the execution client
// (see node-images' docs/ethereum-execution-consensus-pairing.md). The chain is
// named with --network, which the entrypoint only defaults when absent.
func GetLighthouseNetworkComposeConfig(network string) (NetworkConfig, error) {
	return consensusConfig("lighthouse", network, []string{"127.0.0.1:5052:5052", "9000:9000", "9000:9000/udp"}, func(engineEndpoint, jwtPath, chain string) string {
		return fmt.Sprintf("bn --execution-endpoint %s --execution-jwt %s --network %s", engineEndpoint, jwtPath, chain)
	})
}
