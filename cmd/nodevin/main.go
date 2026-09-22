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

package main

import (
	"os"

	"github.com/fiftysixcrypto/nodevin/internal/config"
	"github.com/fiftysixcrypto/nodevin/internal/logger"
	"github.com/fiftysixcrypto/nodevin/pkg/root"
)

func main() {
	// Initialize loggers
	logger.Init()

	// Initialize configurations
	config.InitConfig()

	// Execute the root command
	if err := root.Execute(); err != nil {
		logger.LogError(err.Error())
		os.Exit(1)
	}
}
