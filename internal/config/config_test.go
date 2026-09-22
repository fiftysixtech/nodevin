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
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/fiftysixcrypto/nodevin/internal/logger"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func parse(t *testing.T, content string) (map[string]interface{}, []string) {
	t.Helper()
	return parseEnvFile(strings.NewReader(content))
}

// Regression: viper's env-file reader rejected the documented format
// (`rpc-user=admin`) as a whole, so no line of the file took effect.
func TestParseEnvFile_DocumentedFormat(t *testing.T) {
	values, warnings := parse(t, `
# Docker Configuration
image=fiftysix/bitcoin-core
version=27.0
restart=always
cpu-limit=2.0

# Nodevin Configuration
data-dir=/home/user/.nodevin

rpc-user=admin
rpc-pass=securepassword123
`)

	if len(warnings) != 0 {
		t.Errorf("unexpected warnings: %v", warnings)
	}
	want := map[string]interface{}{
		"image":     "fiftysix/bitcoin-core",
		"version":   "27.0",
		"restart":   "always",
		"cpu-limit": "2.0",
		"data-dir":  "/home/user/.nodevin",
		"rpc-user":  "admin",
		"rpc-pass":  "securepassword123",
	}
	if !reflect.DeepEqual(values, want) {
		t.Errorf("values = %#v\nwant     %#v", values, want)
	}
}

// The example in the docs must always parse cleanly, or the docs are lying.
func TestParseEnvFile_TheExampleInTheDocsParses(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	docs, err := os.ReadFile(filepath.Join(filepath.Dir(thisFile), "..", "..", "docs", "cli-commands.md"))
	if err != nil {
		t.Fatalf("cannot read docs: %v", err)
	}

	text := string(docs)
	start := strings.Index(text, "### Example `.env` File")
	if start < 0 {
		t.Fatal("docs no longer have an '### Example `.env` File' section")
	}
	block := text[start:]
	open := strings.Index(block, "```")
	if open < 0 {
		t.Fatal("no fenced example after the heading")
	}
	block = block[open+3:]
	block = block[strings.Index(block, "\n")+1:] // drop the ```bash line
	block = block[:strings.Index(block, "```")]

	values, warnings := parseEnvFile(strings.NewReader(block))
	if len(warnings) != 0 {
		t.Errorf("the docs' example produced warnings: %v", warnings)
	}
	for _, key := range []string{"image", "rpc-user", "rpc-pass", "data-dir", "restart"} {
		if _, ok := values[key]; !ok {
			t.Errorf("the docs' example should set %q, got %v", key, values)
		}
	}
}

func TestParseEnvFile_ValueSyntax(t *testing.T) {
	cases := []struct {
		name, line, key string
		want            interface{}
	}{
		{"spaces around equals", "rpc-user = admin", "rpc-user", "admin"},
		{"double quotes", `rpc-pass="has spaces # and hash"`, "rpc-pass", "has spaces # and hash"},
		{"single quotes", `rpc-pass='p@ss word'`, "rpc-pass", "p@ss word"},
		{"inline comment on an unquoted value", "restart=always # keep it up", "restart", "always"},
		{"hash without a leading space is part of the value", "rpc-pass=abc#123", "rpc-pass", "abc#123"},
		{"equals signs inside the value", "command=--rpcallowip=0.0.0.0/0 -rpcuser=u", "command", "--rpcallowip=0.0.0.0/0 -rpcuser=u"},
		{"export prefix", "export rpc-user=admin", "rpc-user", "admin"},
		{"upper case and underscores", "RPC_USER=admin", "rpc-user", "admin"},
		{"empty value", "rpc-pass=", "rpc-pass", ""},
		{"comma list", "ports=127.0.0.1:8332:8332, 8333:8333", "ports", []string{"127.0.0.1:8332:8332", "8333:8333"}},
		{"single item list", "volumes=/a:/b", "volumes", []string{"/a:/b"}},
		{"labels", "volume-labels=a=1,b=2", "volume-labels", map[string]string{"a": "1", "b": "2"}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			values, warnings := parse(t, c.line)
			if len(warnings) != 0 {
				t.Fatalf("unexpected warnings: %v", warnings)
			}
			if got := values[c.key]; !reflect.DeepEqual(got, c.want) {
				t.Errorf("values[%q] = %#v, want %#v", c.key, got, c.want)
			}
		})
	}
}

// One bad line must not throw away the rest of the file.
func TestParseEnvFile_ABadLineDoesNotDiscardTheOthers(t *testing.T) {
	values, warnings := parse(t, "rpc-user=admin\nthis line is not valid\n=novalue\nrestart=always\n")

	if values["rpc-user"] != "admin" || values["restart"] != "always" {
		t.Errorf("valid lines were lost: %v", values)
	}
	if len(warnings) != 2 {
		t.Fatalf("got %d warnings, want 2: %v", len(warnings), warnings)
	}
	if !strings.Contains(warnings[0], "line 2") || !strings.Contains(warnings[1], "line 3") {
		t.Errorf("warnings should name the offending lines: %v", warnings)
	}
}

func setUp(t *testing.T) (cwd, home string, logs *bytes.Buffer) {
	t.Helper()

	viper.Reset()
	t.Cleanup(viper.Reset)

	logs = &bytes.Buffer{}
	logger.Init()
	logger.SetOutput(logs)

	home = t.TempDir()
	t.Setenv("HOME", home)

	cwd = t.TempDir()
	old, _ := os.Getwd()
	if err := os.Chdir(cwd); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(old) })
	return cwd, home, logs
}

func TestInitConfig_LoadsTheFileFromTheCurrentDirectory(t *testing.T) {
	cwd, _, logs := setUp(t)
	os.WriteFile(filepath.Join(cwd, ".env"), []byte("rpc-user=admin\ndata-dir=/tmp/from-env-file\n"), 0644)

	InitConfig()

	if got := viper.GetString("rpc-user"); got != "admin" {
		t.Errorf("rpc-user = %q, want admin", got)
	}
	if !viper.IsSet("data-dir") || viper.GetString("data-dir") != "/tmp/from-env-file" {
		t.Errorf("data-dir from the .env file was not applied (IsSet=%v, value=%q)", viper.IsSet("data-dir"), viper.GetString("data-dir"))
	}
	if !strings.Contains(logs.String(), "Successfully loaded configuration") {
		t.Errorf("expected a load message, logs: %q", logs.String())
	}
}

// Having no .env file is the normal state and must not print an ERROR line on
// every command.
func TestInitConfig_NoFileIsSilent(t *testing.T) {
	_, _, logs := setUp(t)

	InitConfig()

	if logs.Len() != 0 {
		t.Errorf("expected no output without a .env file, got %q", logs.String())
	}
}

func TestInitConfig_PrioritizesTheCurrentDirectoryOverHome(t *testing.T) {
	cwd, home, _ := setUp(t)
	os.MkdirAll(filepath.Join(home, ".nodevin"), 0755)
	os.WriteFile(filepath.Join(home, ".nodevin", ".env"), []byte("rpc-user=from-home\nrestart=always\n"), 0644)
	os.WriteFile(filepath.Join(cwd, ".env"), []byte("rpc-user=from-cwd\n"), 0644)

	InitConfig()

	if got := viper.GetString("rpc-user"); got != "from-cwd" {
		t.Errorf("rpc-user = %q, want the current directory's file to win", got)
	}
	if viper.IsSet("restart") {
		t.Error("only the first file found is used, but a value from the home file leaked in")
	}
}

func TestInitConfig_FallsBackToTheHomeDirectory(t *testing.T) {
	_, home, _ := setUp(t)
	os.MkdirAll(filepath.Join(home, ".nodevin"), 0755)
	os.WriteFile(filepath.Join(home, ".nodevin", ".env"), []byte("rpc-user=from-home\n"), 0644)

	InitConfig()

	if got := viper.GetString("rpc-user"); got != "from-home" {
		t.Errorf("rpc-user = %q, want from-home", got)
	}
}

// The docs promise command-line flags override the .env file, and the .env
// file overrides a flag's built-in default.
func TestInitConfig_PrecedenceWithFlags(t *testing.T) {
	cwd, _, _ := setUp(t)
	os.WriteFile(filepath.Join(cwd, ".env"), []byte("rpc-user=from-env-file\nrpc-pass=from-env-file\n"), 0644)

	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	flags.String("rpc-user", "user", "")
	flags.String("rpc-pass", "fiftysix", "")
	viper.BindPFlag("rpc-user", flags.Lookup("rpc-user"))
	viper.BindPFlag("rpc-pass", flags.Lookup("rpc-pass"))

	InitConfig()
	if err := flags.Parse([]string{"--rpc-user=from-command-line"}); err != nil {
		t.Fatal(err)
	}

	if got := viper.GetString("rpc-user"); got != "from-command-line" {
		t.Errorf("rpc-user = %q, want the command-line flag to win", got)
	}
	if got := viper.GetString("rpc-pass"); got != "from-env-file" {
		t.Errorf("rpc-pass = %q, want the .env file to beat the flag's default", got)
	}
}

func TestInitConfig_AListValueBehavesLikeTheFlag(t *testing.T) {
	cwd, _, _ := setUp(t)
	os.WriteFile(filepath.Join(cwd, ".env"), []byte("ports=127.0.0.1:8332:8332,8333:8333\n"), 0644)

	InitConfig()

	want := []string{"127.0.0.1:8332:8332", "8333:8333"}
	if got := viper.GetStringSlice("ports"); !reflect.DeepEqual(got, want) {
		t.Errorf("ports = %#v, want %#v", got, want)
	}
}
