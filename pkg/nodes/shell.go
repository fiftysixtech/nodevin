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
	"fmt"
	"os"
	"os/exec"

	"github.com/fiftysixcrypto/nodevin/internal/utils"

	"github.com/spf13/cobra"
)

var (
	detach     bool
	user       string
	workdir    string
	envVars    []string
	envFile    string
	privileged bool
)

var shellCmd = &cobra.Command{
	Use:   "shell [network]",
	Short: "Run a shell in the specified node container",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true

		network := args[0]
		containerName, exists := utils.GetDefaultLocalMappedContainerName(network)
		if !exists {
			return fmt.Errorf("unsupported blockchain network: %s", network)
		}
		return runShell(containerName)
	},
}

func runShell(containerName string) error {
	args := []string{"exec", "-it"}
	if detach {
		args = append(args, "-d")
	}
	if user != "" {
		args = append(args, "-u", user)
	}
	if workdir != "" {
		args = append(args, "-w", workdir)
	}
	for _, env := range envVars {
		args = append(args, "-e", env)
	}
	if envFile != "" {
		args = append(args, "--env-file", envFile)
	}
	if privileged {
		args = append(args, "--privileged")
	}
	args = append(args, containerName, "/bin/bash")

	cmd := exec.Command("docker", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run shell in container %s: %w", containerName, err)
	}
	return nil
}

func init() {
	shellCmd.Flags().BoolVarP(&detach, "detach", "d", false, "Run command in the background")
	shellCmd.Flags().StringVarP(&user, "docker-user", "u", "", "Username or UID to run the command as")
	shellCmd.Flags().StringVarP(&workdir, "workdir", "w", "", "Working directory inside the container")
	shellCmd.Flags().StringArrayVarP(&envVars, "env", "e", nil, "Set environment variables")
	shellCmd.Flags().StringVar(&envFile, "env-file", "", "Read environment variables from a file")
	shellCmd.Flags().BoolVar(&privileged, "privileged", false, "Give extended privileges to the command")
}
