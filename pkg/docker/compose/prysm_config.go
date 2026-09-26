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

// GetPrysmNetworkComposeConfig builds the compose config for Prysm paired
// with whichever execution client --execution-client selects. See
// GetLighthouseNetworkComposeConfig and node-images'
// docs/ethereum-execution-consensus-pairing.md for the shared reasoning.
//
// Prysm selects its chain with a flag per network (--mainnet, --sepolia) and
// refuses more than one ("cannot use more than one network flag at the same
// time"). Its entrypoint always prepends --mainnet, so a Sepolia stack
// bypasses the entrypoint's flag injection - by starting the binary through
// gosu - and passes the flags the entrypoint would have set itself.
func GetPrysmNetworkComposeConfig(network string) (NetworkConfig, error) {
	return consensusConfig("prysm", network, []string{"127.0.0.1:3500:3500", "13000:13000", "12000:12000/udp"}, func(engineEndpoint, jwtPath, chain string) string {
		base := fmt.Sprintf("beacon-chain --execution-endpoint %s --jwt-secret %s", engineEndpoint, jwtPath)
		if chain == "mainnet" {
			return base
		}
		return fmt.Sprintf("gosu nodeuser beacon-chain --accept-terms-of-use --%s --datadir /node/prysm/data "+
			"--http-host 0.0.0.0 --http-port 3500 --p2p-tcp-port 13000 --p2p-udp-port 12000 "+
			"--execution-endpoint %s --jwt-secret %s", chain, engineEndpoint, jwtPath)
	})
}
