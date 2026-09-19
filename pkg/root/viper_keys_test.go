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

package root

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"
)

var (
	viperBindRe = regexp.MustCompile(`viper\.BindPFlag\(\s*"([^"]+)"`)
	// Only matches literal-string-argument calls by construction. This
	// deliberately excludes dynamic per-service keys such as
	// fmt.Sprintf("%s-image", serviceName) in compose.go's
	// createExtraServices, which are an intentional, separate mechanism, not
	// a hardcoded key that could have a typo caught here.
	viperGetRe = regexp.MustCompile(`viper\.(?:GetString|GetBool|GetStringSlice|GetInt|GetStringMapString|IsSet)\(\s*"([^"]+)"\s*\)`)
)

// allowlistedInternalKeys are viper keys referenced somewhere in the
// codebase that are intentionally not bound to any CLI flag. Every entry
// must be commented with why it's here, so this doesn't silently grow into
// a place bugs go to hide.
var allowlistedInternalKeys = map[string]string{
	// Read in compose.go's CreateComposeFile when building `override`, but
	// no CLI flag registers them and no builder ever calls viper.Set for
	// them either -- confirmed dead (always the zero value, and
	// mergeConfigs only overwrites on non-empty) when this check was
	// introduced. Tracked as separate cleanup rather than removed here to
	// keep this change test-only in scope.
	"local-path":              "dead: never registered as a flag, never set anywhere",
	"snapshot-sync-cid":       "dead: never registered as a flag, never set anywhere",
	"snapshot-sync-data-dir":  "dead: never registered as a flag, never set anywhere",
	"snapshot-sync-file-name": "dead: never registered as a flag, never set anywhere",
}

func repoRoot(t *testing.T) string {
	t.Helper()

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to determine current file path via runtime.Caller")
	}

	dir := filepath.Dir(thisFile)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not find repo root (go.mod) walking up from %s", thisFile)
		}
		dir = parent
	}
}

func collectNonTestGoFiles(t *testing.T, root string) []string {
	t.Helper()

	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if info.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("failed to walk repo at %s: %v", root, err)
	}
	return files
}

// TestEveryViperKeyReferencedIsRegisteredOrAllowlisted is a repo-wide static
// check: every viper.Get*/IsSet("literal-key") call anywhere in the codebase
// must correspond to a key that's either bound via viper.BindPFlag somewhere
// (not just in this package -- pkg/nodes/request.go binds its own flags in
// its own init()) or explicitly allowlisted above with a reason. This is
// what would have caught ord/ord-litecoin's RPC-auth flags being read but
// never registered, and it generalizes: it catches this entire class of bug
// for any future chain, not just these two.
func TestEveryViperKeyReferencedIsRegisteredOrAllowlisted(t *testing.T) {
	root := repoRoot(t)
	files := collectNonTestGoFiles(t, root)

	registered := make(map[string]bool)
	referencedIn := make(map[string][]string)

	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read %s: %v", path, err)
		}
		content := string(data)

		for _, m := range viperBindRe.FindAllStringSubmatch(content, -1) {
			registered[m[1]] = true
		}
		for _, m := range viperGetRe.FindAllStringSubmatch(content, -1) {
			rel, _ := filepath.Rel(root, path)
			referencedIn[m[1]] = append(referencedIn[m[1]], rel)
		}
	}

	if len(registered) == 0 {
		t.Fatal("found zero viper.BindPFlag calls repo-wide -- the scan itself is almost certainly broken, not the codebase")
	}

	var unregistered []string
	for key := range referencedIn {
		if registered[key] {
			continue
		}
		if _, ok := allowlistedInternalKeys[key]; ok {
			continue
		}
		unregistered = append(unregistered, key)
	}
	sort.Strings(unregistered)

	if len(unregistered) > 0 {
		var b strings.Builder
		for _, key := range unregistered {
			fmt.Fprintf(&b, "\n  %q referenced in: %s", key, strings.Join(referencedIn[key], ", "))
		}
		t.Errorf("found %d viper key(s) read via viper.Get*/IsSet that are neither bound via viper.BindPFlag anywhere nor allowlisted in this file:%s", len(unregistered), b.String())
	}
}
