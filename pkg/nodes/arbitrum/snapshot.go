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

package arbitrum

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/fiftysixcrypto/nodevin/internal/logger"
	"github.com/fiftysixcrypto/nodevin/internal/utils"
	"github.com/fiftysixcrypto/nodevin/pkg/docker/compose"
	"github.com/spf13/viper"
)

// runDockerCommand is a variable so tests can substitute it without actually
// invoking Docker (mirrors internal/utils.listRunningContainers' own reason
// for being a var).
var runDockerCommand = func(args ...string) error {
	cmd := exec.Command("docker", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// snapshotDownloadPath resolves --snapshot-download-path, defaulting to
// <nodevin data dir>/.snapshot-staging - a single, shared staging area
// (not per-network) so an operator can point it at a large separate volume
// once, matching requirement 4: it must coexist with the extracted datadir
// during extraction, so it is deliberately not nested under either
// network's own data directory.
func snapshotDownloadPath() (string, error) {
	if p := strings.TrimSpace(viper.GetString("snapshot-download-path")); p != "" {
		expanded, err := utils.ExpandHomeDir(p)
		if err != nil {
			return "", err
		}
		return expanded, nil
	}
	nodevinDataDir, err := utils.GetNodevinDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(nodevinDataDir, ".snapshot-staging"), nil
}

// arbitrumDatabaseInitialised reports whether Nitro has already created a
// real chain database under localChainDataPath for chainName - confirmed via
// a real running node's own log line (database=.../<chain>/nitro/l2chaindata)
// - as opposed to just the chain-name directory, which Nitro creates on
// every attempt (including ones that fail before any snapshot download
// begins, confirmed empirically), so checking for that alone would wrongly
// treat a failed attempt as "already initialised".
func arbitrumDatabaseInitialised(localChainDataPath, chainName string) bool {
	info, err := os.Stat(filepath.Join(localChainDataPath, chainName, "nitro", "l2chaindata"))
	return err == nil && info.IsDir()
}

// EnsureSnapshotInitialised runs Nitro's own snapshot-init flow (phase 1)
// against an uninitialised Arbitrum datadir when --snapshot is set, before
// CreateArbitrumComposeFile's normal run (phase 2) starts it for real. It is
// a no-op - returning nil immediately, no behavior change at all - when
// --snapshot is not given, which is the existing, unchanged default.
//
// Phase 1 and phase 2 always use the same fiftysix/nitro image and version:
// version is resolved once here from the same --version flag start_node.go
// already pulled before calling this, so the two can never drift apart.
func EnsureSnapshotInitialised(cwd string) error {
	kind, err := compose.ResolveArbitrumSnapshotKind()
	if err != nil {
		return err
	}
	if kind == "" {
		return nil
	}

	testnet := utils.CheckIfTestnetOrTestnetNetworkFlag()
	network := "arbitrum"
	if testnet {
		network = "arbitrum-testnet"
	}

	chainName := compose.ArbitrumSnapshotChainName(network)

	localChainDataPath, err := compose.ArbitrumLocalChainDataPath(network)
	if err != nil {
		return err
	}

	// Never re-initialise: requirement 3. A populated datadir makes
	// --snapshot a no-op, not a wipe - whatever is there stays untouched and
	// phase 2 starts normally against it.
	if arbitrumDatabaseInitialised(localChainDataPath, chainName) {
		logger.LogInfo(fmt.Sprintf("Arbitrum datadir for %s is already initialised (found %s) - skipping --snapshot init.",
			network, filepath.Join(localChainDataPath, chainName, "nitro", "l2chaindata")))
		return nil
	}

	if kind == "full-path" {
		logger.LogInfo("WARNING: --snapshot=full-path uses Nitro's newer path-scheme database. OffchainLabs/nitro issue #4746 reports a path-scheme snapshot bootstrap that can stall silently. It also requires a Nitro version that supports path-scheme init - not yet true of every pinned fiftysix/nitro tag. Prefer the default (pruned, hash-scheme) unless you specifically need full-path; see docs/cli-commands.md.")
	}

	// Requirement 7: Nitro reads the parent chain during init, before any
	// snapshot download begins (confirmed empirically - it fails fast on an
	// unreachable L1 before touching --init.download-path at all), so this
	// must resolve before phase 1, exactly like phase 2.
	l1, err := compose.ResolveL1Endpoints(testnet)
	if err != nil {
		return err
	}
	if l1.DockerNetwork != "" {
		logger.LogInfo("No --l1-execution-rpc-url/--l1-beacon-url given: attaching to nodevin's own running Ethereum node as this node's L1 for snapshot init too.")
	}

	logger.LogInfo(fmt.Sprintf("Resolving the latest %s snapshot for %s from snapshot.arbitrum.foundation...", kind, chainName))
	meta, err := compose.FetchArbitrumSnapshotMetadata(chainName, kind)
	if err != nil {
		return fmt.Errorf("could not resolve a snapshot to init from: %w", err)
	}

	downloadPath, err := snapshotDownloadPath()
	if err != nil {
		return err
	}

	// Requirement 5: refuse before starting a download that cannot finish.
	// Checked independently at both paths - the archive (in downloadPath)
	// and its extraction (into localChainDataPath) coexist during
	// extraction, per requirement 4, so each needs its own free space
	// comparable to the snapshot's own total size.
	if err := requireFreeSpace(downloadPath, "--snapshot-download-path", meta); err != nil {
		return err
	}
	if err := requireFreeSpace(localChainDataPath, "the Arbitrum data directory", meta); err != nil {
		return err
	}

	chain := compose.ResolveArbitrumChainIDs(network)

	version := strings.TrimSpace(viper.GetString("version"))
	if version == "" {
		version = "latest"
	}
	image := fmt.Sprintf("fiftysix/nitro:%s", version)

	const containerStagingPath = "/node/nitro/snapshot-staging"
	args := []string{
		"run", "--rm",
		"-v", fmt.Sprintf("%s:/node/nitro/data", localChainDataPath),
		"-v", fmt.Sprintf("%s:%s", downloadPath, containerStagingPath),
		image,
		fmt.Sprintf("--chain.id=%s", chain.ChainID),
		fmt.Sprintf("--parent-chain.id=%s", chain.ParentChainID),
		fmt.Sprintf("--parent-chain.connection.url=%s", l1.ExecutionRPCURL),
		fmt.Sprintf("--parent-chain.blob-client.beacon-url=%s", l1.BeaconURL),
		fmt.Sprintf("--init.latest=%s", kind),
		fmt.Sprintf("--init.download-path=%s", containerStagingPath),
		"--init.then-quit",
	}

	logger.LogInfo(fmt.Sprintf(
		"Starting Arbitrum snapshot init (%s, ~%s) - this can take 12-24 hours on mainnet. Streaming Nitro's own output below.",
		kind, meta.Size.TotalHuman))
	logger.LogInfo(fmt.Sprintf(
		"If this is interrupted, staging at %s is left exactly as it is - re-run the same command and Nitro resumes the download instead of restarting it.",
		downloadPath))

	if err := runDockerCommand(args...); err != nil {
		return fmt.Errorf("snapshot init failed (staging left intact at %s - re-run to resume): %w", downloadPath, err)
	}

	logger.LogInfo("Arbitrum snapshot init completed successfully. Starting the node normally.")
	return nil
}

func requireFreeSpace(path, flagDescription string, meta *compose.ArbitrumSnapshotMetadata) error {
	avail, err := compose.AvailableDiskSpace(path)
	if err != nil {
		return fmt.Errorf("failed to check free space at %s (%s): %w", flagDescription, path, err)
	}
	if avail < uint64(meta.Size.TotalBytes) {
		return fmt.Errorf(
			"not enough free space at %s (%s): the %s snapshot needs ~%s, only %s is free there. Free up space, or point %s at a larger volume",
			flagDescription, path, meta.SnapshotKind, meta.Size.TotalHuman, utils.GetSizeDescription(int64(avail)), flagDescription)
	}
	return nil
}
