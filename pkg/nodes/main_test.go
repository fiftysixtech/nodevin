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
	"os"
	"testing"

	"github.com/fiftysixcrypto/nodevin/internal/logger"
)

// TestMain ensures logger.Init() has run before any test in this package.
// logger.LogInfo/LogError panic on a nil *log.Logger otherwise (they're
// package-level vars normally set up once at CLI startup in main.go, which
// a test binary never runs), and several functions under test here
// (stopNode, called from deleteNetworkDirectory) log unconditionally before
// doing anything else.
func TestMain(m *testing.M) {
	logger.Init()
	os.Exit(m.Run())
}
