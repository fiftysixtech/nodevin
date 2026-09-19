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
	"github.com/fiftysixcrypto/nodevin/internal/logger"
	"github.com/fiftysixcrypto/nodevin/internal/version"
	"github.com/fiftysixcrypto/nodevin/pkg/initialize"
	"github.com/fiftysixcrypto/nodevin/pkg/nodes"
	"github.com/fiftysixcrypto/nodevin/pkg/update"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "nodevin",
	Short: "nodevin CLI",
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of Nodevin",
	Run: func(cmd *cobra.Command, args []string) {
		logger.LogInfo("nodevin CLI v" + version.Version)
	},
}

func init() {
	// Docker flags and configuration settings
	rootCmd.PersistentFlags().String("command", "", "Initial command to run the node")
	rootCmd.PersistentFlags().StringSlice("ports", []string{}, "Ports to bind to the node")
	rootCmd.PersistentFlags().StringSlice("volumes", []string{}, "Docker volumes to mount for compose file")
	rootCmd.PersistentFlags().StringSlice("volume-definitions", []string{}, "Docker volume definitions for compose file")
	rootCmd.PersistentFlags().String("image", "", "Docker image to use for the node (image name -- ex: fiftysix/bitcoin-core)")
	rootCmd.PersistentFlags().String("version", "", "Version of Docker image to use for the node (tag -- ex: latest, 27.0)")
	rootCmd.PersistentFlags().String("restart", "no", "Whether or not to restart software on failure (docker restart parameter -- ex: always, no)")
	rootCmd.PersistentFlags().String("container-name", "", "Docker container name for compose file")
	rootCmd.PersistentFlags().StringSlice("docker-networks", []string{}, "Docker networks to connect to for compose file")
	rootCmd.PersistentFlags().String("network-driver", "", "Docker network driver for compose file")
	rootCmd.PersistentFlags().StringToString("volume-labels", map[string]string{}, "Docker volume labels for compose file")
	rootCmd.PersistentFlags().String("cpu-limit", "", "Maximum CPU limit of use (amount of CPUs -- ex: 1.5)")
	rootCmd.PersistentFlags().String("mem-limit", "", "Maximum memory limit of use (positive integer followed by 'b', 'k', 'm', 'g', to indicate bytes, kilobytes, megabytes, or gigabytes -- ex: 50m)")
	rootCmd.PersistentFlags().String("cpu-reservation", "", "Reserve a set amount of CPU for use (amount of CPUs -- ex: 1.5)")
	rootCmd.PersistentFlags().String("mem-reservation", "", "Reserve a set amount of memory for use (positive integer followed by 'b', 'k', 'm', 'g', to indicate bytes, kilobytes, megabytes, or gigabytes -- ex: 50m)")

	// Nodevin specific flags
	rootCmd.PersistentFlags().String("data-dir", "", "Local data directory to store nodevin chain data (default: ~/.nodevin)")
	rootCmd.PersistentFlags().Bool("snapshot-sync", false, "Download chain data from a snapshot url -- (default: false)")
	rootCmd.PersistentFlags().String("snapshot-sync-command", "", "Init command to be ran to handle snapshot data and place in directory")
	rootCmd.PersistentFlags().Bool("testnet", false, "Run assumed network testnet")
	rootCmd.PersistentFlags().String("network", "", "Run node attached to a specific network (network name -- ex: goerli, testnet3)")

	// Chain software specific flags (bitcoin-core, litecoin-core, etc.)
	rootCmd.PersistentFlags().String("rpc-user", "user", "Username passed in via command for JSON RPC -- (default: user)")
	rootCmd.PersistentFlags().String("rpc-pass", "fiftysix", "Username passed in via command for JSON RPC -- (default: fiftysix)")
	rootCmd.PersistentFlags().Bool("cookie-auth", false, "Use authentication directly with node cookie file -- (default: false)")

	// Bitcoin specific flags
	rootCmd.PersistentFlags().Bool("ord", false, "Run ordinal software ord alongside the Bitcoin/Litecoin node")
	rootCmd.PersistentFlags().String("ord-image", "fiftysix/ord", "Docker image to use for ord (image name -- ex: fiftysix/ord)")
	rootCmd.PersistentFlags().String("ord-version", "latest", "Version of Docker image to use for ord (tag -- ex: latest, 27.0)")
	rootCmd.PersistentFlags().Bool("ord-cookie-auth", false, "(ord only) Use authentication directly with the Bitcoin node cookie file -- (default: false)")
	rootCmd.PersistentFlags().String("ord-rpc-user", "", "(ord only) Username passed in via command for the Bitcoin JSON RPC connection (default: user)")
	rootCmd.PersistentFlags().String("ord-rpc-pass", "", "(ord only) Password passed in via command for the Bitcoin JSON RPC connection (default: fiftysix)")

	// Litecoin specific flags
	rootCmd.PersistentFlags().Bool("ord-litecoin", false, "Run ordinal software ord alongside the Litecoin node")
	rootCmd.PersistentFlags().String("ord-litecoin-image", "fiftysix/ord-litecoin", "Docker image to use for ord (image name -- ex: fiftysix/ord-litecoin)")
	rootCmd.PersistentFlags().String("ord-litecoin-version", "latest", "Version of Docker image to use for ord (tag -- ex: latest, 27.0)")
	rootCmd.PersistentFlags().Bool("ord-litecoin-cookie-auth", false, "(ord-litecoin only) Use authentication directly with the Litecoin node cookie file -- (default: false)")
	rootCmd.PersistentFlags().String("ord-litecoin-rpc-user", "", "(ord-litecoin only) Username passed in via command for the Litecoin JSON RPC connection (default: falls back to --ord-rpc-user, then \"user\")")
	rootCmd.PersistentFlags().String("ord-litecoin-rpc-pass", "", "(ord-litecoin only) Password passed in via command for the Litecoin JSON RPC connection (default: falls back to --ord-rpc-pass, then \"fiftysix\")")

	// IPFS specific flags
	rootCmd.PersistentFlags().Bool("ipfs-cluster", false, "Run ipfs-cluster software ord alongside the IPFS node")
	rootCmd.PersistentFlags().String("ipfs-cluster-image", "fiftysix/ipfs-cluster", "Docker image to use for ipfs-cluster (image name -- ex: fiftysix/ipfs-cluster)")
	rootCmd.PersistentFlags().String("ipfs-cluster-version", "latest", "Version of Docker image to use for ipfs-cluster (tag -- ex: latest, 1.1.1)")
	rootCmd.PersistentFlags().String("ipfs-cluster-peername", "", "(ipfs-cluster only) The peername(s) to attach to (ex: cluster-peer-1)")
	rootCmd.PersistentFlags().String("ipfs-cluster-secret", "", "(ipfs-cluster only) The cluster secret required for connection (ex: ...)")
	rootCmd.PersistentFlags().String("ipfs-cluster-bootstrap", "", "(ipfs-cluster only) The bootstrap node address (ex: /ip4/172.20.0.2/tcp/4001/p2p/12D3KooWHUZ36WvuUBmz5aFLJ9PoNKrUJRMSA22i98BkoAaQPRzi)")

	// Bind flags to viper
	// Docker flags and configs
	viper.BindPFlag("command", rootCmd.PersistentFlags().Lookup("command"))
	viper.BindPFlag("ports", rootCmd.PersistentFlags().Lookup("ports"))
	viper.BindPFlag("volumes", rootCmd.PersistentFlags().Lookup("volumes"))
	viper.BindPFlag("volume-definitions", rootCmd.PersistentFlags().Lookup("volume-definitions"))
	viper.BindPFlag("image", rootCmd.PersistentFlags().Lookup("image"))
	viper.BindPFlag("version", rootCmd.PersistentFlags().Lookup("version"))
	viper.BindPFlag("restart", rootCmd.PersistentFlags().Lookup("restart"))
	viper.BindPFlag("container-name", rootCmd.PersistentFlags().Lookup("container-name"))
	viper.BindPFlag("docker-networks", rootCmd.PersistentFlags().Lookup("docker-networks"))
	viper.BindPFlag("network-driver", rootCmd.PersistentFlags().Lookup("network-driver"))
	viper.BindPFlag("volume-labels", rootCmd.PersistentFlags().Lookup("volume-labels"))
	viper.BindPFlag("cpu-limit", rootCmd.PersistentFlags().Lookup("cpu-limit"))
	viper.BindPFlag("mem-limit", rootCmd.PersistentFlags().Lookup("mem-limit"))
	viper.BindPFlag("cpu-reservation", rootCmd.PersistentFlags().Lookup("cpu-reservation"))
	viper.BindPFlag("mem-reservation", rootCmd.PersistentFlags().Lookup("mem-reservation"))

	// Nodevin specific flags
	viper.BindPFlag("data-dir", rootCmd.PersistentFlags().Lookup("data-dir"))
	viper.BindPFlag("snapshot-sync", rootCmd.PersistentFlags().Lookup("snapshot-sync"))
	viper.BindPFlag("snapshot-sync-command", rootCmd.PersistentFlags().Lookup("snapshot-sync-command"))
	viper.BindPFlag("testnet", rootCmd.PersistentFlags().Lookup("testnet"))
	viper.BindPFlag("network", rootCmd.PersistentFlags().Lookup("network"))

	// Chain software specific flags (bitcoin-core, litecoin-core, etc.)
	viper.BindPFlag("rpc-user", rootCmd.PersistentFlags().Lookup("rpc-user"))
	viper.BindPFlag("rpc-pass", rootCmd.PersistentFlags().Lookup("rpc-pass"))
	viper.BindPFlag("cookie-auth", rootCmd.PersistentFlags().Lookup("cookie-auth"))

	// Bitcoin specific flags
	viper.BindPFlag("ord", rootCmd.PersistentFlags().Lookup("ord"))
	viper.BindPFlag("ord-image", rootCmd.PersistentFlags().Lookup("ord-image"))
	viper.BindPFlag("ord-version", rootCmd.PersistentFlags().Lookup("ord-version"))
	viper.BindPFlag("ord-cookie-auth", rootCmd.PersistentFlags().Lookup("ord-cookie-auth"))
	viper.BindPFlag("ord-rpc-user", rootCmd.PersistentFlags().Lookup("ord-rpc-user"))
	viper.BindPFlag("ord-rpc-pass", rootCmd.PersistentFlags().Lookup("ord-rpc-pass"))

	// Litecoin specific flags
	viper.BindPFlag("ord-litecoin", rootCmd.PersistentFlags().Lookup("ord-litecoin"))
	viper.BindPFlag("ord-litecoin-image", rootCmd.PersistentFlags().Lookup("ord-litecoin-image"))
	viper.BindPFlag("ord-litecoin-version", rootCmd.PersistentFlags().Lookup("ord-litecoin-version"))
	viper.BindPFlag("ord-litecoin-cookie-auth", rootCmd.PersistentFlags().Lookup("ord-litecoin-cookie-auth"))
	viper.BindPFlag("ord-litecoin-rpc-user", rootCmd.PersistentFlags().Lookup("ord-litecoin-rpc-user"))
	viper.BindPFlag("ord-litecoin-rpc-pass", rootCmd.PersistentFlags().Lookup("ord-litecoin-rpc-pass"))

	// IPFS specific flags
	viper.BindPFlag("ipfs-cluster", rootCmd.PersistentFlags().Lookup("ipfs-cluster"))
	viper.BindPFlag("ipfs-cluster-image", rootCmd.PersistentFlags().Lookup("ipfs-cluster-image"))
	viper.BindPFlag("ipfs-cluster-version", rootCmd.PersistentFlags().Lookup("ipfs-cluster-version"))
	viper.BindPFlag("ipfs-cluster-peername", rootCmd.PersistentFlags().Lookup("ipfs-cluster-peername"))
	viper.BindPFlag("ipfs-cluster-secret", rootCmd.PersistentFlags().Lookup("ipfs-cluster-secret"))
	viper.BindPFlag("ipfs-cluster-bootstrap", rootCmd.PersistentFlags().Lookup("ipfs-cluster-bootstrap"))

	// Add node commands
	rootCmd.AddCommand(nodes.RequestCmd)
	rootCmd.AddCommand(nodes.ShellCmd)
	rootCmd.AddCommand(nodes.CleanupCmd)
	rootCmd.AddCommand(nodes.DeleteCmd)
	rootCmd.AddCommand(nodes.StartNodeCmd)
	rootCmd.AddCommand(nodes.StopNodeCmd)
	rootCmd.AddCommand(nodes.LogsCmd)
	rootCmd.AddCommand(nodes.InfoCmd)
	rootCmd.AddCommand(nodes.ListCmd)
	rootCmd.AddCommand(nodes.ViewCmd)

	// Add IPFS support commands
	rootCmd.AddCommand(nodes.IpfsSupportCmd)

	// Add init command
	rootCmd.AddCommand(initialize.InitCmd)

	// Add manual update commands
	rootCmd.AddCommand(update.UpdateCmd)
}

func Execute() error {
	rootCmd.AddCommand(versionCmd)
	return rootCmd.Execute()
}
