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

package utils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

func TestExpandHomeDir(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get home dir: %v", err)
	}

	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"bare tilde", "~", home},
		{"tilde with subpath", "~/Desktop", filepath.Join(home, "Desktop")},
		{"tilde with nested subpath", "~/Desktop/data", filepath.Join(home, "Desktop", "data")},
		{"absolute path is untouched", "/tmp/nodevin-abs-test", "/tmp/nodevin-abs-test"},
		{"relative path is untouched", "relative/subdir", "relative/subdir"},
		{"empty string is untouched", "", ""},
		{"tilde not followed by slash is untouched", "~notreallyhome", "~notreallyhome"},
		{"tilde in the middle is untouched", "/tmp/~/Desktop", "/tmp/~/Desktop"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := ExpandHomeDir(c.input)
			if err != nil {
				t.Fatalf("ExpandHomeDir(%q) returned error: %v", c.input, err)
			}
			if got != c.want {
				t.Errorf("ExpandHomeDir(%q) = %q, want %q", c.input, got, c.want)
			}
		})
	}
}

func TestGetNodevinDataDirExpandsTilde(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	viper.Set("data-dir", "~/nested")
	t.Cleanup(func() { viper.Set("data-dir", "") })

	dir, err := GetNodevinDataDir()
	if err != nil {
		t.Fatalf("GetNodevinDataDir() returned error: %v", err)
	}

	want := filepath.Join(tmpHome, "nested", ".nodevin", "data")
	if dir != want {
		t.Errorf("GetNodevinDataDir() = %q, want %q", dir, want)
	}

	if _, err := os.Stat(dir); err != nil {
		t.Errorf("expected directory to exist at %q: %v", dir, err)
	}

	// The bug this guards against: a literal "~" directory created relative
	// to the current working directory instead of the path resolving to the
	// user's home directory.
	if _, err := os.Stat("~"); !os.IsNotExist(err) {
		t.Errorf("a literal '~' directory should not have been created in the working directory")
	}
}

func TestGetNodevinDataDirWithAbsolutePath(t *testing.T) {
	tmpDir := t.TempDir()

	viper.Set("data-dir", tmpDir)
	t.Cleanup(func() { viper.Set("data-dir", "") })

	dir, err := GetNodevinDataDir()
	if err != nil {
		t.Fatalf("GetNodevinDataDir() returned error: %v", err)
	}

	want := filepath.Join(tmpDir, ".nodevin", "data")
	if dir != want {
		t.Errorf("GetNodevinDataDir() = %q, want %q", dir, want)
	}
}
