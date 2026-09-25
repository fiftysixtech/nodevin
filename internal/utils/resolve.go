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
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/viper"
)

// AmbiguousClientError means a network can run as several containers (see
// NetworkInfo.AlternateContainerNames) and more than one of them is present,
// so nodevin will not guess which one the user means.
type AmbiguousClientError struct {
	Network    string
	Flag       string
	Candidates []string
	// Running is true when the candidates are running right now, false when
	// they are only present on disk (data directory or compose file).
	Running bool
}

func (e *AmbiguousClientError) Error() string {
	state := "have data on disk"
	if e.Running {
		state = "are running"
	}
	return fmt.Sprintf("more than one %s client %s (%s); pass --%s=<name> to choose one",
		e.Network, state, strings.Join(e.Candidates, ", "), e.Flag)
}

// listRunningContainers is a variable so tests can substitute it.
var listRunningContainers = func() ([]string, error) {
	out, err := exec.Command("docker", "ps", "--format", "{{.Names}}").Output()
	if err != nil {
		return nil, err
	}
	var names []string
	for _, line := range strings.Split(string(out), "\n") {
		if name := strings.TrimSpace(line); name != "" {
			names = append(names, name)
		}
	}
	return names, nil
}

// RunningContainerNames returns the names of all running Docker containers.
func RunningContainerNames() ([]string, error) {
	return listRunningContainers()
}

// CandidateContainerNames returns every container name network's primary
// service may run as, default first. It is nil for an unknown network.
func CandidateContainerNames(network string) []string {
	info, ok := networkInfoMap[network]
	if !ok {
		return nil
	}
	return append([]string{info.ContainerName}, info.AlternateContainerNames...)
}

// AllContainerNames returns every container name any registered network can
// run as, including alternates. Use it (not NetworkContainerMap) wherever the
// question is "is any nodevin node running or storing data?".
func AllContainerNames() []string {
	seen := make(map[string]bool)
	var names []string
	for _, info := range networkInfoMap {
		for _, name := range append([]string{info.ContainerName}, info.AlternateContainerNames...) {
			if !seen[name] {
				seen[name] = true
				names = append(names, name)
			}
		}
	}
	sort.Strings(names)
	return names
}

// StackOwner returns the network whose compose stack network runs inside
// (e.g. "ethereum" for "lighthouse"), if any.
func StackOwner(network string) (string, bool) {
	info, ok := networkInfoMap[network]
	if !ok || info.PartOf == "" {
		return "", false
	}
	return info.PartOf, true
}

// ComponentNetworks returns the networks that run inside network's stack.
func ComponentNetworks(network string) []string {
	var components []string
	for name, info := range networkInfoMap {
		if info.PartOf == network {
			components = append(components, name)
		}
	}
	sort.Strings(components)
	return components
}

func explicitClient(network string, info NetworkInfo) (string, bool, error) {
	if info.ClientFlag == "" || !viper.IsSet(info.ClientFlag) {
		return "", false, nil
	}
	value := viper.GetString(info.ClientFlag)
	if value == "" {
		return "", false, nil
	}
	for _, candidate := range CandidateContainerNames(network) {
		if candidate == value {
			return value, true, nil
		}
	}
	return "", false, fmt.Errorf("unsupported --%s: %s (supported: %s)",
		info.ClientFlag, value, strings.Join(CandidateContainerNames(network), ", "))
}

// hasFootprint reports whether a container has left anything behind: a data
// directory or a generated compose file.
func hasFootprint(container string) bool {
	if dataDir, err := GetNodevinDataDir(); err == nil {
		if stat, err := os.Stat(filepath.Join(dataDir, container)); err == nil && stat.IsDir() {
			return true
		}
	}
	_, err := FindComposeFile(container)
	return err == nil
}

// ResolveContainerName returns the container that currently represents
// network. For most networks that is just the registry's ContainerName. For a
// network whose primary service can run as one of several clients it discovers
// which one, in order: the explicit --<ClientFlag> value, the one running
// right now, the one that has a data directory or compose file, and finally
// the default. If several match at a step it returns *AmbiguousClientError
// rather than guessing.
func ResolveContainerName(network string) (string, error) {
	info, ok := networkInfoMap[network]
	if !ok {
		return "", fmt.Errorf("unsupported blockchain network: %s", network)
	}
	candidates := CandidateContainerNames(network)
	if len(candidates) == 1 {
		return info.ContainerName, nil
	}

	if name, set, err := explicitClient(network, info); err != nil {
		return "", err
	} else if set {
		return name, nil
	}

	if running, err := listRunningContainers(); err == nil {
		var matches []string
		for _, candidate := range candidates {
			for _, name := range running {
				if name == candidate {
					matches = append(matches, candidate)
				}
			}
		}
		switch len(matches) {
		case 0:
		case 1:
			return matches[0], nil
		default:
			return "", &AmbiguousClientError{Network: network, Flag: info.ClientFlag, Candidates: matches, Running: true}
		}
	}

	var present []string
	for _, candidate := range candidates {
		if hasFootprint(candidate) {
			present = append(present, candidate)
		}
	}
	switch len(present) {
	case 0:
		return info.ContainerName, nil
	case 1:
		return present[0], nil
	default:
		return "", &AmbiguousClientError{Network: network, Flag: info.ClientFlag, Candidates: present}
	}
}

// ExplicitContainerName is ResolveContainerName for destructive commands: for
// a network with selectable clients it requires the client to be named with
// its flag, and never infers it. Other networks resolve as usual.
func ExplicitContainerName(network string) (string, error) {
	info, ok := networkInfoMap[network]
	if !ok {
		return "", fmt.Errorf("unsupported blockchain network: %s", network)
	}
	if len(CandidateContainerNames(network)) == 1 {
		return info.ContainerName, nil
	}

	name, set, err := explicitClient(network, info)
	if err != nil {
		return "", err
	}
	if !set {
		var present []string
		for _, candidate := range CandidateContainerNames(network) {
			if hasFootprint(candidate) {
				present = append(present, candidate)
			}
		}
		hint := ""
		if len(present) > 0 {
			hint = fmt.Sprintf(" (found data for: %s)", strings.Join(present, ", "))
		}
		return "", fmt.Errorf("refusing to guess which %s client's data to delete: pass --%s=<%s>%s",
			network, info.ClientFlag, strings.Join(CandidateContainerNames(network), "|"), hint)
	}
	return name, nil
}

// FindComposeFileForContainer locates the compose file that manages
// container. A container that runs inside another network's stack (a
// consensus client) is managed by that stack's compose file.
func FindComposeFileForContainer(container string) (string, error) {
	if path, err := FindComposeFile(container); err == nil {
		return path, nil
	}
	for _, info := range networkInfoMap {
		if info.ContainerName != container || info.PartOf == "" {
			continue
		}
		stack, err := ResolveContainerName(info.PartOf)
		if err != nil {
			return "", err
		}
		return FindComposeFile(stack)
	}
	return FindComposeFile(container)
}
