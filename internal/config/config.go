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

package config

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/fiftysixcrypto/nodevin/internal/logger"
	"github.com/spf13/viper"
)

type Config struct {
	Port      string `mapstructure:"port"`
	DataDir   string `mapstructure:"data-dir"`
	ExtraArgs string `mapstructure:"extra-args"`
	//ResourceLimit string `mapstructure:"resource_limit"`
}

var AppConfig Config

// listKeys are the flags that take a comma-separated list on the command line,
// so they are split the same way when they come from a .env file.
var listKeys = map[string]bool{
	"ports":              true,
	"volumes":            true,
	"volume-definitions": true,
	"docker-networks":    true,
}

// envFileCandidates returns where a .env file is looked for, in priority
// order: the current directory, ~/.nodevin, /etc/nodevin, then the directory of
// the nodevin executable.
func envFileCandidates() []string {
	paths := []string{".env"}

	if homeDir, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(homeDir, ".nodevin", ".env"))
	}

	paths = append(paths, "/etc/nodevin/.env")

	if exePath, err := os.Executable(); err == nil {
		paths = append(paths, filepath.Join(filepath.Dir(exePath), ".env"))
	}

	return paths
}

// InitConfig loads the first .env file found and makes its values available
// to viper below command-line flags. Having no .env file is normal and silent.
func InitConfig() {
	// Allow overriding with environment variables
	viper.AutomaticEnv()

	for _, path := range envFileCandidates() {
		file, err := os.Open(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			logger.LogError(fmt.Sprintf("Warning: could not read %s: %v", path, err))
			return
		}

		values, warnings := parseEnvFile(file)
		file.Close()

		for _, warning := range warnings {
			logger.LogError(fmt.Sprintf("Warning: %s: %s", path, warning))
		}

		if err := viper.MergeConfigMap(values); err != nil {
			logger.LogError(fmt.Sprintf("Warning: could not apply %s: %v", path, err))
			return
		}

		logger.LogInfo(fmt.Sprintf("Successfully loaded configuration from %s", path))
		return
	}
}

// parseEnvFile reads KEY=VALUE lines. Keys are flag names (rpc-user, data-dir,
// ...); upper case and underscores are accepted too, so RPC_USER means rpc-user.
// Blank lines and lines starting with # are ignored, values may be quoted, and
// an unquoted value may end in a " # comment". A line that cannot be parsed is
// reported and skipped without affecting the others.
func parseEnvFile(r io.Reader) (map[string]interface{}, []string) {
	values := make(map[string]interface{})
	var warnings []string

	scanner := bufio.NewScanner(r)
	for lineNumber := 1; scanner.Scan(); lineNumber++ {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimSpace(strings.TrimPrefix(line, "export "))

		key, value, found := strings.Cut(line, "=")
		key = normalizeKey(key)
		if !found || key == "" {
			warnings = append(warnings, fmt.Sprintf("line %d ignored, expected KEY=VALUE", lineNumber))
			continue
		}

		values[key] = typedValue(key, parseValue(value))
	}

	return values, warnings
}

func normalizeKey(key string) string {
	return strings.ReplaceAll(strings.ToLower(strings.TrimSpace(key)), "_", "-")
}

func parseValue(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return ""
	}

	if quote := value[0]; quote == '"' || quote == '\'' {
		if end := strings.IndexByte(value[1:], quote); end >= 0 {
			return value[1 : end+1]
		}
		return value[1:]
	}

	if i := strings.Index(value, " #"); i >= 0 {
		value = value[:i]
	}
	return strings.TrimSpace(value)
}

func typedValue(key, value string) interface{} {
	switch {
	case listKeys[key]:
		var items []string
		for _, item := range strings.Split(value, ",") {
			if item = strings.TrimSpace(item); item != "" {
				items = append(items, item)
			}
		}
		return items
	case key == "volume-labels":
		labels := make(map[string]string)
		for _, pair := range strings.Split(value, ",") {
			if k, v, ok := strings.Cut(pair, "="); ok && strings.TrimSpace(k) != "" {
				labels[strings.TrimSpace(k)] = strings.TrimSpace(v)
			}
		}
		return labels
	default:
		return value
	}
}
