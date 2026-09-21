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

import (
	"strings"
	"testing"

	"github.com/spf13/viper"
)

// Kubo's RPC API has admin-level access to the node and upstream documents
// publishing it on 127.0.0.1 only. The image binds it inside the container, so
// the host-side mapping is the only thing keeping it off the network.
func TestKuboAPIAndGatewayArePublishedOnLoopbackOnly(t *testing.T) {
	viper.Set("data-dir", t.TempDir())
	t.Cleanup(func() { viper.Set("data-dir", "") })

	cfg, err := GetKuboNetworkComposeConfig("ipfs")
	if err != nil {
		t.Fatal(err)
	}

	published := map[string]string{}
	for _, mapping := range cfg.Ports {
		parts := strings.Split(mapping, ":")
		container := strings.TrimSuffix(parts[len(parts)-1], "/udp")
		bind := ""
		if len(parts) == 3 {
			bind = parts[0]
		}
		published[container] = bind
	}

	for _, container := range []string{"5001", "8080"} {
		bind, ok := published[container]
		if !ok {
			t.Errorf("container port %s is not published at all: %v", container, cfg.Ports)
			continue
		}
		if bind != "127.0.0.1" {
			t.Errorf("container port %s is published on %q, want 127.0.0.1 (mappings: %v)", container, bind, cfg.Ports)
		}
	}

	if bind, ok := published["4001"]; !ok || bind != "" {
		t.Errorf("swarm port 4001 must be published on all interfaces so peers can connect, got %q (mappings: %v)", bind, cfg.Ports)
	}
}
