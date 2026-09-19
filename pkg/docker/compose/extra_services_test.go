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
	"strings"
	"testing"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

func useTempDataDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	viper.Set("data-dir", dir)
	t.Cleanup(func() { viper.Set("data-dir", "") })
	return dir
}

func readComposeFile(t *testing.T, path string) ComposeFile {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read compose file: %v", err)
	}
	var cf ComposeFile
	if err := yaml.Unmarshal(raw, &cf); err != nil {
		t.Fatalf("failed to parse compose file: %v", err)
	}
	return cf
}

// makeSparseFile creates a file whose reported size is `size` bytes without
// using real disk space, to exercise the >= 1 GB "data already present"
// threshold that decides whether the init container is skipped.
func makeSparseFile(t *testing.T, dir string, size int64) {
	t.Helper()
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(dir + "/chain.dat")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := f.Truncate(size); err != nil {
		t.Skipf("filesystem does not support sparse files of this size: %v", err)
	}
}

func TestCreateExtraServices_OrdWithInitContainer(t *testing.T) {
	useTempDataDir(t)

	ordCfg, err := GetOrdNetworkComposeConfig("ord")
	if err != nil {
		t.Fatal(err)
	}

	services, networks, volumes := createExtraServices(
		[]string{"ord"}, []NetworkConfig{ordCfg},
		map[string]NetworkDetails{"bitcoin-net": {Driver: "bridge"}},
		map[string]VolumeDetails{},
	)

	svc, ok := services["ord"]
	if !ok {
		t.Fatalf("no 'ord' service in %v", keys(services))
	}
	if svc.Image != "fiftysix/ord:latest" {
		t.Errorf("Image = %q, want fiftysix/ord:latest", svc.Image)
	}
	if !strings.HasSuffix(svc.Command, " server") {
		t.Errorf("Command = %q, want it to end with ' server'", svc.Command)
	}

	initName := "init-config-ord"
	init, ok := services[initName]
	if !ok {
		t.Fatalf("expected %q in %v", initName, keys(services))
	}
	if init.Restart != "no" {
		t.Errorf("init container Restart = %q, want no", init.Restart)
	}
	if svc.DependsOn[initName].Condition != "service_completed_successfully" {
		t.Errorf("ord should wait for its init container, DependsOn = %+v", svc.DependsOn)
	}

	label := volumes["ord-init-volume"].Labels["nodevin.init.volume"]
	if label != "true" {
		t.Errorf("ord-init-volume label nodevin.init.volume = %q, want true", label)
	}
	if _, ok := volumes["ord-data"]; !ok {
		t.Errorf("builder's own volume definition 'ord-data' was not carried through: %v", volumes)
	}
	if _, ok := networks["bitcoin-net"]; !ok {
		t.Errorf("network definitions were not carried through: %v", networks)
	}
}

// The Environment fix touched createExtraServices too; the main-service path
// is covered in compose_test.go, this covers the companion-service path
// (`start bitcoin --ipfs-cluster`-style).
func TestCreateExtraServices_CarriesEnvironment(t *testing.T) {
	useTempDataDir(t)
	viper.Set("ipfs-cluster-secret", "s3cret")
	t.Cleanup(func() { viper.Set("ipfs-cluster-secret", "") })

	cfg, err := GetIpfsClusterNetworkComposeConfig("ipfs-cluster")
	if err != nil {
		t.Fatal(err)
	}

	services, _, _ := createExtraServices([]string{"ipfs-cluster"}, []NetworkConfig{cfg}, nil, nil)

	if got := services["ipfs-cluster"].Environment["CLUSTER_SECRET"]; got != "s3cret" {
		t.Errorf("Environment[CLUSTER_SECRET] = %q, want s3cret (env: %v)", got, services["ipfs-cluster"].Environment)
	}
}

func TestCreateExtraServices_UserVolumesSkipInitContainer(t *testing.T) {
	useTempDataDir(t)

	cfg, err := GetOrdNetworkComposeConfig("ord")
	if err != nil {
		t.Fatal(err)
	}

	// A service name unique to this test: viper.Set can't be undone, and
	// "<name>-volumes" being set is what disables the init container.
	const name = "zz-extra-volumes-test"
	viper.Set(name+"-volumes", []string{"/host/path:/container/path"})

	services, _, _ := createExtraServices([]string{name}, []NetworkConfig{cfg}, nil, nil)

	if _, ok := services["init-config-"+name]; ok {
		t.Errorf("init container should be skipped when %s-volumes is set: %v", name, keys(services))
	}
	if got := services[name].Volumes; len(got) != 1 || got[0] != "/host/path:/container/path" {
		t.Errorf("Volumes = %v, want the user-supplied volume to replace the default", got)
	}
}

func TestCreateExtraServices_ExistingDataSkipsInitContainer(t *testing.T) {
	useTempDataDir(t)

	cfg, err := GetOrdNetworkComposeConfig("ord")
	if err != nil {
		t.Fatal(err)
	}
	makeSparseFile(t, cfg.LocalPath, 1<<30)

	services, _, _ := createExtraServices([]string{"ord"}, []NetworkConfig{cfg}, nil, nil)

	if _, ok := services["init-config-ord"]; ok {
		t.Errorf("init container should be skipped when >= 1 GB already exists: %v", keys(services))
	}
	if len(services["ord"].DependsOn) != 0 {
		t.Errorf("DependsOn = %+v, want none", services["ord"].DependsOn)
	}
}

func TestCreateComposeFile_ExistingDataSkipsInitContainer(t *testing.T) {
	dir := useTempDataDir(t)

	cfg, err := GetBitcoinNetworkComposeConfig("bitcoin")
	if err != nil {
		t.Fatal(err)
	}

	t.Run("just under 1 GB still gets an init container", func(t *testing.T) {
		makeSparseFile(t, cfg.LocalPath, 1<<30-1)
		path, err := CreateComposeFile("bitcoin-core", cfg, nil, nil, dir)
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := readComposeFile(t, path).Services["init-config-bitcoin-core"]; !ok {
			t.Error("expected an init container below the 1 GB threshold")
		}
	})

	t.Run("exactly 1 GB skips the init container", func(t *testing.T) {
		makeSparseFile(t, cfg.LocalPath, 1<<30)
		path, err := CreateComposeFile("bitcoin-core", cfg, nil, nil, dir)
		if err != nil {
			t.Fatal(err)
		}
		cf := readComposeFile(t, path)
		if _, ok := cf.Services["init-config-bitcoin-core"]; ok {
			t.Errorf("init container should be skipped at >= 1 GB: %v", keys(cf.Services))
		}
		if len(cf.Services["bitcoin-core"].DependsOn) != 0 {
			t.Errorf("DependsOn = %+v, want none", cf.Services["bitcoin-core"].DependsOn)
		}
	})
}

// TestCreateComposeFile_WithExtraServices exercises the real `start bitcoin
// --ord` composition: main + companion + both init containers + watchtower.
func TestCreateComposeFile_WithExtraServices(t *testing.T) {
	dir := useTempDataDir(t)

	btc, err := GetBitcoinNetworkComposeConfig("bitcoin")
	if err != nil {
		t.Fatal(err)
	}
	ord, err := GetOrdNetworkComposeConfig("ord")
	if err != nil {
		t.Fatal(err)
	}

	path, err := CreateComposeFile("bitcoin-core", btc, []string{"ord"}, []NetworkConfig{ord}, dir)
	if err != nil {
		t.Fatal(err)
	}
	cf := readComposeFile(t, path)

	for _, want := range []string{"bitcoin-core", "ord", "init-config-bitcoin-core", "init-config-ord", "watchtower"} {
		if _, ok := cf.Services[want]; !ok {
			t.Errorf("missing service %q in %v", want, keys(cf.Services))
		}
	}

	// Watchtower must watch the real services but never the one-shot init
	// containers, and its argument order must be stable between runs.
	if got, want := cf.Services["watchtower"].Command, "bitcoin-core ord --interval 7200"; got != want {
		t.Errorf("watchtower command = %q, want %q", got, want)
	}
}

// TestCreateComposeFile_UserFlagOverrides proves the --image/--version/
// --restart/--ports/--container-name flags win over the builder's defaults,
// and that the resource flags (--cpu-limit/--mem-limit/--cpu-reservation/
// --mem-reservation) actually reach the main service. They used to be merged
// and then silently dropped.
func TestCreateComposeFile_UserFlagOverrides(t *testing.T) {
	dir := useTempDataDir(t)

	overrides := map[string]interface{}{
		"image":           "myorg/custom-bitcoin",
		"version":         "27.0",
		"restart":         "always",
		"container-name":  "my-btc",
		"ports":           []string{"1234:1234"},
		"cpu-limit":       "2.0",
		"mem-limit":       "1g",
		"cpu-reservation": "1.0",
		"mem-reservation": "512m",
	}
	for k, v := range overrides {
		viper.Set(k, v)
	}
	t.Cleanup(func() {
		for k, v := range overrides {
			switch v.(type) {
			case []string:
				viper.Set(k, []string{})
			default:
				viper.Set(k, "")
			}
		}
	})

	cfg, err := GetBitcoinNetworkComposeConfig("bitcoin")
	if err != nil {
		t.Fatal(err)
	}
	path, err := CreateComposeFile("bitcoin-core", cfg, nil, nil, dir)
	if err != nil {
		t.Fatal(err)
	}

	svc := readComposeFile(t, path).Services["bitcoin-core"]
	if svc.Image != "myorg/custom-bitcoin:27.0" {
		t.Errorf("Image = %q", svc.Image)
	}
	if svc.Restart != "always" {
		t.Errorf("Restart = %q", svc.Restart)
	}
	if svc.ContainerName != "my-btc" {
		t.Errorf("ContainerName = %q", svc.ContainerName)
	}
	if len(svc.Ports) != 1 || svc.Ports[0] != "1234:1234" {
		t.Errorf("Ports = %v", svc.Ports)
	}
	if svc.Deploy == nil {
		t.Fatal("Deploy is nil: resource flags were dropped")
	}
	r := svc.Deploy.Resources
	if r.Limits.CPUs != "2.0" || r.Limits.Memory != "1g" || r.Reservations.CPUs != "1.0" || r.Reservations.Memory != "512m" {
		t.Errorf("Deploy.Resources = %+v, want limits 2.0/1g and reservations 1.0/512m", r)
	}
}

func keys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
