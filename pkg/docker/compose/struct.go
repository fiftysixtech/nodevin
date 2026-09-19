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

// ResourceDetails defines CPU and memory limits and reservations.
type ResourceDetails struct {
	CPUs   string `yaml:"cpus,omitempty"`
	Memory string `yaml:"memory,omitempty"`
}

// Resources defines resource limits and reservations for a service.
type Resources struct {
	Limits       ResourceDetails `yaml:"limits,omitempty"`
	Reservations ResourceDetails `yaml:"reservations,omitempty"`
}

// Deploy holds the resource deployment configuration for a service.
type Deploy struct {
	Resources Resources `yaml:"resources,omitempty"`
}

// Healthcheck defines the parameters for a health check.
type Healthcheck struct {
	Test        []string `yaml:"test"`
	Interval    string   `yaml:"interval"`
	Timeout     string   `yaml:"timeout"`
	Retries     int      `yaml:"retries"`
	StartPeriod string   `yaml:"start_period,omitempty"`
}

// ServiceDependsOnCondition defines the health condition for depends_on.
type ServiceDependsOnCondition struct {
	Condition string `yaml:"condition"`
}

// Service defines the configuration of a service in the Docker Compose file.
type Service struct {
	Image         string                               `yaml:"image"`
	ContainerName string                               `yaml:"container_name"`
	User          string                               `yaml:"user,omitempty"`
	Restart       string                               `yaml:"restart"`
	Command       string                               `yaml:"command"`
	Entrypoint    string                               `yaml:"entrypoint,omitempty"`
	Ports         []string                             `yaml:"ports"`
	Volumes       []string                             `yaml:"volumes"`
	Networks      []string                             `yaml:"networks"`
	Healthcheck   *Healthcheck                         `yaml:"healthcheck,omitempty"`
	Environment   map[string]string                    `yaml:"environment,omitempty"`
	DependsOn     map[string]ServiceDependsOnCondition `yaml:"depends_on,omitempty"`
	Deploy        *Deploy                              `yaml:"deploy,omitempty"`
}

// NetworkDetails defines the network configuration for a service.
type NetworkDetails struct {
	Driver string `yaml:"driver"`
}

// VolumeDetails defines the volume configuration for a service.
type VolumeDetails struct {
	Labels map[string]string `yaml:"labels"`
}

// ComposeFile defines the top-level structure of the Docker Compose file.
type ComposeFile struct {
	//Version  string                    `yaml:"version"`
	Services map[string]Service        `yaml:"services"`
	Networks map[string]NetworkDetails `yaml:"networks"`
	Volumes  map[string]VolumeDetails  `yaml:"volumes"`
}

// NetworkConfig holds the configuration used to override or define services.
type NetworkConfig struct {
	Image         string
	Version       string
	ContainerName string
	Command       string
	// CommandIsIntentionallyEmpty marks that this network's Command is meant
	// to be empty (e.g. ipfs/ipfs-cluster rely on the image's own baked-in
	// entrypoint), so a generic "Command must be non-empty" check doesn't
	// need a hardcoded per-network allowlist to know that's expected.
	CommandIsIntentionallyEmpty bool
	Restart                     string
	Ports                       []string
	Volumes                     []string
	Networks                    []string
	Deploy                      Deploy
	Environment                 map[string]string
	NetworkDefs                 map[string]NetworkDetails
	VolumeDefs                  map[string]VolumeDetails
	LocalPath                   string
	SnapshotSyncCID             string
	LocalChainDataPath          string
	SnapshotDataFilename        string
	SnapshotSyncCommand         string
}
