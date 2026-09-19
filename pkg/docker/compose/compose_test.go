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
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

func TestIsDeploySet(t *testing.T) {
	cases := []struct {
		name   string
		deploy Deploy
		want   bool
	}{
		{"all empty", Deploy{}, false},
		{"limits cpu set", Deploy{Resources: Resources{Limits: ResourceDetails{CPUs: "1.5"}}}, true},
		{"limits memory set", Deploy{Resources: Resources{Limits: ResourceDetails{Memory: "512m"}}}, true},
		{"reservations cpu set", Deploy{Resources: Resources{Reservations: ResourceDetails{CPUs: "1.0"}}}, true},
		{"reservations memory set", Deploy{Resources: Resources{Reservations: ResourceDetails{Memory: "256m"}}}, true},
		{"everything set", Deploy{Resources: Resources{
			Limits:       ResourceDetails{CPUs: "2.0", Memory: "1g"},
			Reservations: ResourceDetails{CPUs: "1.0", Memory: "512m"},
		}}, true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := isDeploySet(c.deploy); got != c.want {
				t.Errorf("isDeploySet(%+v) = %v, want %v", c.deploy, got, c.want)
			}
		})
	}
}

// TestMergeConfigs_ScalarFields covers every string field mergeConfigs handles
// individually: the override wins when non-empty, and the default is
// preserved untouched when the override leaves the field at its zero value.
func TestMergeConfigs_ScalarFields(t *testing.T) {
	fields := []struct {
		name string
		set  func(cfg *NetworkConfig, val string)
		get  func(cfg NetworkConfig) string
	}{
		{"Image", func(c *NetworkConfig, v string) { c.Image = v }, func(c NetworkConfig) string { return c.Image }},
		{"Version", func(c *NetworkConfig, v string) { c.Version = v }, func(c NetworkConfig) string { return c.Version }},
		{"ContainerName", func(c *NetworkConfig, v string) { c.ContainerName = v }, func(c NetworkConfig) string { return c.ContainerName }},
		{"Restart", func(c *NetworkConfig, v string) { c.Restart = v }, func(c NetworkConfig) string { return c.Restart }},
		{"Command", func(c *NetworkConfig, v string) { c.Command = v }, func(c NetworkConfig) string { return c.Command }},
		{"LocalPath", func(c *NetworkConfig, v string) { c.LocalPath = v }, func(c NetworkConfig) string { return c.LocalPath }},
		{"SnapshotSyncCID", func(c *NetworkConfig, v string) { c.SnapshotSyncCID = v }, func(c NetworkConfig) string { return c.SnapshotSyncCID }},
		{"LocalChainDataPath", func(c *NetworkConfig, v string) { c.LocalChainDataPath = v }, func(c NetworkConfig) string { return c.LocalChainDataPath }},
		{"SnapshotDataFilename", func(c *NetworkConfig, v string) { c.SnapshotDataFilename = v }, func(c NetworkConfig) string { return c.SnapshotDataFilename }},
		{"SnapshotSyncCommand", func(c *NetworkConfig, v string) { c.SnapshotSyncCommand = v }, func(c NetworkConfig) string { return c.SnapshotSyncCommand }},
	}

	for _, f := range fields {
		t.Run(f.name+"/override wins when set", func(t *testing.T) {
			def := NetworkConfig{}
			f.set(&def, "default-value")
			over := NetworkConfig{}
			f.set(&over, "override-value")

			got := mergeConfigs(def, over)
			if want := "override-value"; f.get(got) != want {
				t.Errorf("mergeConfigs() %s = %q, want %q", f.name, f.get(got), want)
			}
		})

		t.Run(f.name+"/default preserved when override empty", func(t *testing.T) {
			def := NetworkConfig{}
			f.set(&def, "default-value")
			over := NetworkConfig{}

			got := mergeConfigs(def, over)
			if want := "default-value"; f.get(got) != want {
				t.Errorf("mergeConfigs() %s = %q, want %q", f.name, f.get(got), want)
			}
		})
	}
}

// TestMergeConfigs_SliceFields covers Ports/Volumes/Networks: the override
// slice replaces the default wholesale when non-empty, and the default is
// preserved when the override is nil/empty.
func TestMergeConfigs_SliceFields(t *testing.T) {
	fields := []struct {
		name string
		set  func(cfg *NetworkConfig, val []string)
		get  func(cfg NetworkConfig) []string
	}{
		{"Ports", func(c *NetworkConfig, v []string) { c.Ports = v }, func(c NetworkConfig) []string { return c.Ports }},
		{"Volumes", func(c *NetworkConfig, v []string) { c.Volumes = v }, func(c NetworkConfig) []string { return c.Volumes }},
		{"Networks", func(c *NetworkConfig, v []string) { c.Networks = v }, func(c NetworkConfig) []string { return c.Networks }},
	}

	defaultVal := []string{"default-a", "default-b"}
	overrideVal := []string{"override-a"}

	for _, f := range fields {
		t.Run(f.name+"/override wins when non-empty", func(t *testing.T) {
			def := NetworkConfig{}
			f.set(&def, defaultVal)
			over := NetworkConfig{}
			f.set(&over, overrideVal)

			got := f.get(mergeConfigs(def, over))
			if len(got) != 1 || got[0] != "override-a" {
				t.Errorf("mergeConfigs() %s = %v, want %v", f.name, got, overrideVal)
			}
		})

		t.Run(f.name+"/default preserved when override empty", func(t *testing.T) {
			def := NetworkConfig{}
			f.set(&def, defaultVal)
			over := NetworkConfig{}

			got := f.get(mergeConfigs(def, over))
			if len(got) != 2 || got[0] != "default-a" || got[1] != "default-b" {
				t.Errorf("mergeConfigs() %s = %v, want %v", f.name, got, defaultVal)
			}
		})
	}
}

// TestMergeConfigs_DeployFields covers the 4 leaf Deploy fields, each merged
// independently of the others.
func TestMergeConfigs_DeployFields(t *testing.T) {
	fields := []struct {
		name string
		set  func(d *Deploy, val string)
		get  func(d Deploy) string
	}{
		{"Limits.CPUs", func(d *Deploy, v string) { d.Resources.Limits.CPUs = v }, func(d Deploy) string { return d.Resources.Limits.CPUs }},
		{"Limits.Memory", func(d *Deploy, v string) { d.Resources.Limits.Memory = v }, func(d Deploy) string { return d.Resources.Limits.Memory }},
		{"Reservations.CPUs", func(d *Deploy, v string) { d.Resources.Reservations.CPUs = v }, func(d Deploy) string { return d.Resources.Reservations.CPUs }},
		{"Reservations.Memory", func(d *Deploy, v string) { d.Resources.Reservations.Memory = v }, func(d Deploy) string { return d.Resources.Reservations.Memory }},
	}

	for _, f := range fields {
		t.Run(f.name+"/override wins when set", func(t *testing.T) {
			def := NetworkConfig{}
			f.set(&def.Deploy, "default-value")
			over := NetworkConfig{}
			f.set(&over.Deploy, "override-value")

			got := mergeConfigs(def, over)
			if want := "override-value"; f.get(got.Deploy) != want {
				t.Errorf("mergeConfigs() Deploy.%s = %q, want %q", f.name, f.get(got.Deploy), want)
			}
		})

		t.Run(f.name+"/default preserved when override empty", func(t *testing.T) {
			def := NetworkConfig{}
			f.set(&def.Deploy, "default-value")
			over := NetworkConfig{}

			got := mergeConfigs(def, over)
			if want := "default-value"; f.get(got.Deploy) != want {
				t.Errorf("mergeConfigs() Deploy.%s = %q, want %q", f.name, f.get(got.Deploy), want)
			}
		})
	}
}

func TestMergeConfigs_NetworkDefsMerge(t *testing.T) {
	def := NetworkConfig{
		NetworkDefs: map[string]NetworkDetails{
			"default-net": {Driver: "bridge"},
			"shared-net":  {Driver: "default-driver"},
		},
	}
	over := NetworkConfig{
		NetworkDefs: map[string]NetworkDetails{
			"override-net": {Driver: "host"},
			"shared-net":   {Driver: "override-driver"},
		},
	}

	got := mergeConfigs(def, over)

	if len(got.NetworkDefs) != 3 {
		t.Fatalf("mergeConfigs() NetworkDefs has %d entries, want 3: %+v", len(got.NetworkDefs), got.NetworkDefs)
	}
	if got.NetworkDefs["default-net"].Driver != "bridge" {
		t.Errorf("default-only key was lost or changed: %+v", got.NetworkDefs["default-net"])
	}
	if got.NetworkDefs["override-net"].Driver != "host" {
		t.Errorf("override-only key was not merged in: %+v", got.NetworkDefs["override-net"])
	}
	if got.NetworkDefs["shared-net"].Driver != "override-driver" {
		t.Errorf("override should win on key collision, got: %+v", got.NetworkDefs["shared-net"])
	}
}

func TestMergeConfigs_VolumeDefsMerge(t *testing.T) {
	def := NetworkConfig{
		VolumeDefs: map[string]VolumeDetails{
			"default-vol": {Labels: map[string]string{"owner": "default"}},
			"shared-vol":  {Labels: map[string]string{"owner": "default"}},
		},
	}
	over := NetworkConfig{
		VolumeDefs: map[string]VolumeDetails{
			"override-vol": {Labels: map[string]string{"owner": "override"}},
			"shared-vol":   {Labels: map[string]string{"owner": "override"}},
		},
	}

	got := mergeConfigs(def, over)

	if len(got.VolumeDefs) != 3 {
		t.Fatalf("mergeConfigs() VolumeDefs has %d entries, want 3: %+v", len(got.VolumeDefs), got.VolumeDefs)
	}
	if got.VolumeDefs["default-vol"].Labels["owner"] != "default" {
		t.Errorf("default-only key was lost or changed: %+v", got.VolumeDefs["default-vol"])
	}
	if got.VolumeDefs["override-vol"].Labels["owner"] != "override" {
		t.Errorf("override-only key was not merged in: %+v", got.VolumeDefs["override-vol"])
	}
	if got.VolumeDefs["shared-vol"].Labels["owner"] != "override" {
		t.Errorf("override should win on key collision, got: %+v", got.VolumeDefs["shared-vol"])
	}
}

// TestMergeConfigs_EnvironmentMerge guards against the bug where
// NetworkConfig.Environment was silently dropped by mergeConfigs: it never
// appeared in the merged result, so it could never reach the generated
// Service/YAML no matter what a builder set it to.
func TestMergeConfigs_EnvironmentMerge(t *testing.T) {
	def := NetworkConfig{
		Environment: map[string]string{
			"DEFAULT_ONLY": "default",
			"SHARED_KEY":   "default",
		},
	}
	over := NetworkConfig{
		Environment: map[string]string{
			"OVERRIDE_ONLY": "override",
			"SHARED_KEY":    "override",
		},
	}

	got := mergeConfigs(def, over)

	if len(got.Environment) != 3 {
		t.Fatalf("mergeConfigs() Environment has %d entries, want 3: %+v", len(got.Environment), got.Environment)
	}
	if got.Environment["DEFAULT_ONLY"] != "default" {
		t.Errorf("default-only key was lost or changed: %+v", got.Environment)
	}
	if got.Environment["OVERRIDE_ONLY"] != "override" {
		t.Errorf("override-only key was not merged in: %+v", got.Environment)
	}
	if got.Environment["SHARED_KEY"] != "override" {
		t.Errorf("override should win on key collision, got: %+v", got.Environment)
	}
}

func TestMergeConfigs_EnvironmentPreservedWhenOverrideEmpty(t *testing.T) {
	def := NetworkConfig{Environment: map[string]string{"FOO": "bar"}}
	over := NetworkConfig{}

	got := mergeConfigs(def, over)

	if len(got.Environment) != 1 || got.Environment["FOO"] != "bar" {
		t.Errorf("mergeConfigs() Environment = %+v, want map[FOO:bar]", got.Environment)
	}
}

func TestGetDirectorySize(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "a.txt"), make([]byte, 100), 0644); err != nil {
		t.Fatalf("failed to write fixture file: %v", err)
	}

	subDir := filepath.Join(dir, "sub")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("failed to create fixture subdirectory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "b.txt"), make([]byte, 50), 0644); err != nil {
		t.Fatalf("failed to write fixture file: %v", err)
	}

	got, err := getDirectorySize(dir)
	if err != nil {
		t.Fatalf("getDirectorySize() returned error: %v", err)
	}
	if want := int64(150); got != want {
		t.Errorf("getDirectorySize() = %d, want %d", got, want)
	}
}

// TestCreateComposeFile_CarriesEnvironment is the end-to-end regression test
// for the ipfs-cluster bug: GetIpfsClusterNetworkComposeConfig builds a
// NetworkConfig.Environment from --ipfs-cluster-secret/--ipfs-cluster-peername,
// but that map never used to reach the generated docker-compose.yml at all.
func TestCreateComposeFile_CarriesEnvironment(t *testing.T) {
	tmpDir := t.TempDir()
	viper.Set("data-dir", tmpDir)
	t.Cleanup(func() { viper.Set("data-dir", "") })

	viper.Set("ipfs-cluster-secret", "test-secret")
	viper.Set("ipfs-cluster-peername", "test-peer")
	t.Cleanup(func() {
		viper.Set("ipfs-cluster-secret", "")
		viper.Set("ipfs-cluster-peername", "")
	})

	cfg, err := GetIpfsClusterNetworkComposeConfig("ipfs-cluster")
	if err != nil {
		t.Fatalf("GetIpfsClusterNetworkComposeConfig() returned error: %v", err)
	}
	if cfg.Environment["CLUSTER_SECRET"] != "test-secret" || cfg.Environment["CLUSTER_PEERNAME"] != "test-peer" {
		t.Fatalf("builder did not populate Environment as expected: %+v", cfg.Environment)
	}

	path, err := CreateComposeFile("ipfs-cluster", cfg, []string{}, []NetworkConfig{}, tmpDir)
	if err != nil {
		t.Fatalf("CreateComposeFile() returned error: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read generated compose file: %v", err)
	}

	if !strings.Contains(string(raw), "environment:") {
		t.Errorf("generated compose file has no environment: block at all:\n%s", raw)
	}

	var composeFile ComposeFile
	if err := yaml.Unmarshal(raw, &composeFile); err != nil {
		t.Fatalf("failed to unmarshal generated compose file: %v", err)
	}

	service, ok := composeFile.Services["ipfs-cluster"]
	if !ok {
		t.Fatalf("generated compose file has no 'ipfs-cluster' service: %+v", composeFile.Services)
	}
	if service.Environment["CLUSTER_SECRET"] != "test-secret" {
		t.Errorf("service.Environment[CLUSTER_SECRET] = %q, want %q", service.Environment["CLUSTER_SECRET"], "test-secret")
	}
	if service.Environment["CLUSTER_PEERNAME"] != "test-peer" {
		t.Errorf("service.Environment[CLUSTER_PEERNAME] = %q, want %q", service.Environment["CLUSTER_PEERNAME"], "test-peer")
	}
}

func TestGetDockerSocketVolume(t *testing.T) {
	got := getDockerSocketVolume()

	want := "/var/run/docker.sock:/var/run/docker.sock"
	if runtime.GOOS == "windows" {
		want = "//./pipe/docker_engine:/var/run/docker.sock"
	}

	if got != want {
		t.Errorf("getDockerSocketVolume() = %q, want %q (GOOS=%s)", got, want, runtime.GOOS)
	}
}
