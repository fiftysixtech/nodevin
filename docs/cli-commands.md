# Nodevin CLI Documentation

Nodevin is a command-line interface (CLI) that simplifies the setup, management, and running of blockchain nodes. Below is a comprehensive list of the available commands and their options.

---

## Table of Contents

**Note**: On Windows, commands should be written as `nodevin.exe <command>`, not `nodevin <command>`. For example, `nodevin.exe init`.

**Note**: every command exits with a non-zero status and prints an error if it fails (for example, an unsupported network, or Docker Compose not being available), so scripts and CI can check `$?` after running nodevin.

### Getting Started
- [nodevin init](#nodevin-init)
- [nodevin list](#nodevin-list)
- [nodevin version](#nodevin-version)

### Running Nodes
- [nodevin start](#nodevin-start)
- [nodevin stop](#nodevin-stop)
- [nodevin update](#nodevin-update)

### Interacting with Nodes
- [nodevin shell](#nodevin-shell)
- [nodevin logs](#nodevin-logs)
- [nodevin request](#nodevin-request)
- [nodevin info](#nodevin-info)
- [nodevin view](#nodevin-view)

### Data Cleanup
- [nodevin delete](#nodevin-delete)
- [nodevin cleanup](#nodevin-cleanup)

### Using a .env File
- [Info](#env-file)

---

## Commands and Detailed Options

### `nodevin init`

- **Description**: Initializes Nodevin and checks system capabilities to ensure compatibility.
- **Simple Example**: `nodevin init`

---

### `nodevin list`

- **Description**: Lists all networks compatible with Nodevin.
- **Simple Example**: `nodevin list`

---

### `nodevin version`

- **Description**: Prints the installed Nodevin version.
- **Simple Example**: `nodevin version`

---

### `nodevin start`

- **Description**: Starts a blockchain node for the specified network (e.g., `nodevin start bitcoin`).
- **Simple Example**: `nodevin start bitcoin`

#### Options:

- **`--ord`**
*Description*: Runs ordinal software `ord` alongside the Bitcoin node.
*Default*: `false`
*Usage*: `--ord`

- **`--ord-image`**

*Description*: Docker image to use for `ord`.
*Default*: `fiftysix/ord`
*Usage*: `--ord-image=<docker-image>`

- **`--ord-version`**

*Description*: Version of the Docker image to use for `ord`.
*Default*: `latest`
*Usage*: `--ord-version=<tag>`

- **`--ord-cookie-auth`**

*Description*: (ord only) Use authentication directly with the Bitcoin node's cookie file, instead of `--ord-rpc-user`/`--ord-rpc-pass`.
*Default*: `false`
*Usage*: `--ord-cookie-auth`

- **`--ord-rpc-user`**

*Description*: (ord only) Username `ord` uses for the Bitcoin JSON-RPC connection. Falls back to `user` if unset.
*Usage*: `--ord-rpc-user=<username>`

- **`--ord-rpc-pass`**

*Description*: (ord only) Password `ord` uses for the Bitcoin JSON-RPC connection. Falls back to `fiftysix` if unset.
*Usage*: `--ord-rpc-pass=<password>`

- **`--ord-litecoin`**

*Description*: Runs ordinal software `ord` alongside the Litecoin node.
*Default*: `false`
*Usage*: `--ord-litecoin`

- **`--ord-litecoin-image`**

*Description*: Docker image to use for `ord-litecoin`.
*Default*: `fiftysix/ord-litecoin`
*Usage*: `--ord-litecoin-image=<docker-image>`

- **`--ord-litecoin-version`**

*Description*: Version of the Docker image to use for `ord-litecoin`.
*Default*: `latest`
*Usage*: `--ord-litecoin-version=<tag>`

- **`--ord-litecoin-cookie-auth`**

*Description*: (ord-litecoin only) Use authentication directly with the Litecoin node's cookie file, instead of `--ord-litecoin-rpc-user`/`--ord-litecoin-rpc-pass`.
*Default*: `false`
*Usage*: `--ord-litecoin-cookie-auth`

- **`--ord-litecoin-rpc-user`**

*Description*: (ord-litecoin only) Username `ord-litecoin` uses for the Litecoin JSON-RPC connection. Falls back to `--ord-rpc-user`, then `user`, if unset.
*Usage*: `--ord-litecoin-rpc-user=<username>`

- **`--ord-litecoin-rpc-pass`**

*Description*: (ord-litecoin only) Password `ord-litecoin` uses for the Litecoin JSON-RPC connection. Falls back to `--ord-rpc-pass`, then `fiftysix`, if unset.
*Usage*: `--ord-litecoin-rpc-pass=<password>`

*Note on ord web ports*: each `ord` instance publishes its web interface on its own host port so several can run at once: `ord` on `80`, `ord` (testnet) on `8081`, `ord-litecoin` on `8082`, and `ord-litecoin` (testnet) on `8083`. Override with `--ports`.

- **`--ipfs-cluster`**

*Description*: Runs `ipfs-cluster` software alongside the IPFS node.
*Default*: `false`
*Usage*: `--ipfs-cluster`

#### Ethereum options

`nodevin start ethereum` runs one Ethereum execution client and one consensus client together in a single Docker Compose stack. The consensus client drives the execution client over the Engine API using a shared JWT secret that the execution client generates and the consensus client reads (read-only).

- **`--execution-client`**

*Description*: Which execution client to run.
*Options*: `reth`, `geth`, `erigon`, `besu`, `nethermind`
*Default*: `reth`
*Usage*: `--execution-client=geth`

- **`--consensus-client`**

*Description*: Which consensus client to run alongside it, or `none` to run the execution client alone.
*Options*: `lighthouse`, `prysm`, `teku`, `nimbus`, `lodestar`, `none`
*Default*: `lighthouse`
*Usage*: `--consensus-client=prysm`

- **`--checkpoint-sync-url`**

*Description*: URL of a checkpoint sync provider the consensus client starts from. **Required** unless `--consensus-client=none`: a consensus client cannot sync mainnet from genesis (Lighthouse and Teku refuse to try). Nodevin has no default endpoint on purpose; choosing whose checkpoint to trust is your decision. See the [public endpoint list](https://eth-clients.github.io/checkpoint-sync-endpoints/). It must be an `http(s)` URL. Pass it on each `start`; once a client has a database it resumes from that instead (checked with Nimbus and Lodestar).
*Usage*: `--checkpoint-sync-url=<url>`

- **`--blob-serving`**

*Description*: Makes the consensus client keep and serve the blob data that L2 nodes (Arbitrum, Base) read from it. Since the Fusaka upgrade (PeerDAS) a default beacon node custodies only a few of the 128 "data columns" and cannot serve whole blobs; a node that custodies half of them can reconstruct every blob (`semi`), and one that custodies all of them serves them directly (`full`, at a higher bandwidth cost). A bare `--blob-serving` means `semi`. It needs a consensus client, so it cannot be combined with `--consensus-client=none`.
*Usage*: `--blob-serving` or `--blob-serving=full`

| Client | `semi` | `full` |
| --- | --- | --- |
| lighthouse, prysm | `--semi-supernode` | `--supernode` |
| lodestar | `--semiSupernode` | `--supernode` |
| nimbus | `--light-supernode` | `--peerdas-supernode` |
| teku | not available (refused) | `--p2p-subscribe-all-custody-subnets-enabled` |

Teku has no half-custody mode, so `--blob-serving` with Teku must be `--blob-serving=full`; nodevin refuses `semi` rather than quietly using the far more bandwidth-hungry `full`. The beacon REST API that serves the blobs stays published on `127.0.0.1` only.

*What was verified*, on Sepolia (nothing here was run on mainnet, and no Nitro or Base node has read from it yet):
- The consensus client reports `custody_group_count` 4 by default, 64 with `semi` (lighthouse, prysm, nimbus, lodestar) and 128 with `full` (lighthouse, teku), on real stacks.
- On Linux, with a fresh checkpoint sync and 30 minutes of running, a default Lighthouse served none of the 20 most recent slots' blobs, while Lighthouse `full`, Prysm `semi` and Teku `full` served 19–20 of 20.
- Nimbus `semi` served none, but it only ever had 0–2 peers, so it is unresolved whether Nimbus cannot do it or just lacked peers. For now prefer Lighthouse or Prysm, or Teku with `full`; `start` prints a notice for Nimbus `semi`.
- Lighthouse `semi` could not be measured: the test runner ran out of disk twice at the same point, about 35 minutes in. Whether Lighthouse `semi` itself is responsible is being investigated.

*Start early.* After a fresh checkpoint sync a blob-serving node only has blobs from about its checkpoint onward: in 30 minutes none of the serving nodes had any block a day back. How long it takes to backfill the roughly 18-day blob window was not measured. An L2 node started from a snapshot needs blobs from the snapshot's date forward, so start the L1 consensus client well ahead of it, or use a rented blob endpoint in the meantime.

*Cost.* On mainnet a semi-supernode is estimated by ethPandaOps at roughly half of a full supernode's 50–100 Mb/s, about 8–16 TB of bandwidth per month; this is an estimate, not something measured here. `start` prints a warning on mainnet.

- **`--consensus-image`** / **`--consensus-version`**

*Description*: Docker image and tag for the consensus client.
*Default*: `fiftysix/<consensus-client>` and `latest`
*Usage*: `--consensus-image=<docker-image> --consensus-version=<tag>`

Example:
```bash
nodevin start ethereum \
--execution-client=geth \
--consensus-client=nimbus \
--checkpoint-sync-url=<provider-url>
```

*Ports*: Ethereum publishes its JSON-RPC on `127.0.0.1:8547`, WebSocket on `127.0.0.1:8548` and peer port `30305` (the canonical 8545/8546/30303/30304 are already used by Ethereum Classic). The Engine API (8551) is never published to the host. The consensus client's beacon REST API is published on `127.0.0.1` only (Lighthouse and Nimbus `5052`, Prysm `3500`, Teku `5051`, Lodestar `9596`); its peer ports stay public (`9000` for Lighthouse, Teku, Nimbus and Lodestar, plus `9001/udp` for Nimbus; `13000` and `12000/udp` for Prysm). Override with `--ports`.

*Sepolia testnet*: add `--testnet` to run the Sepolia testnet instead of mainnet, with the same client flags:

```bash
nodevin start ethereum --testnet --checkpoint-sync-url=<sepolia-provider-url>
```

The checkpoint provider must serve Sepolia (for example `https://checkpoint-sync.sepolia.ethpandaops.io`; the [public list](https://eth-clients.github.io/checkpoint-sync-endpoints/) has more). The Sepolia stack is entirely separate from mainnet: its containers, data directories (`~/.nodevin/data/<client>-testnet`), volumes and Docker network all carry a `-testnet` suffix, so it never touches mainnet data. Its execution client publishes JSON-RPC on `127.0.0.1:8549`, WebSocket on `127.0.0.1:8550` and peer port `30306`; the consensus clients use the same ports as on mainnet. Use `--testnet` with `stop`, `logs`, `shell` and `delete` to target it (`nodevin stop ethereum --testnet`, `nodevin delete ethereum --testnet --execution-client=geth`), and `nodevin request ethereum-testnet --method eth_chainId` to query it. Sepolia is the only Ethereum testnet supported so far.

**Lodestar does not work on Sepolia.** In four 15-minute runs on Linux (including with `--nat`) Lodestar found no Sepolia peers, although the stack itself comes up correctly and Lodestar does find peers on mainnet; the cause is unknown. `nodevin start ethereum --testnet --consensus-client=lodestar` prints a warning. Use Lighthouse, Prysm, Teku or Nimbus for Sepolia.

*Notes*:
- Nimbus starts from your checkpoint provider with its `trustedNodeSync` command, run once when it has no database yet.
- Only one Ethereum stack can run at a time, mainnet or Sepolia: `start` refuses to start over a running execution or consensus client from either. Run `nodevin stop ethereum` (or `nodevin stop ethereum --testnet`) first.
- Compatibility caveat: in testing on Docker Desktop for Mac, Nimbus and Lodestar found no peers while Lighthouse, Prysm and Teku did. This looked like a Docker Desktop UDP port publishing issue: on Linux (verified on GitHub Actions Ubuntu runners) all five consensus clients find peers.
- Mainnet Ethereum needs a lot of disk space and can take days to sync. The reth image defaults to archive mode.

#### Arbitrum and Base options

`nodevin start arbitrum` and `nodevin start base` run an Arbitrum (Nitro) or Base (base-reth + base-consensus) L2 node. Both depend on an Ethereum L1: an execution JSON-RPC endpoint, and a consensus (beacon) endpoint that can serve blob sidecars (a default beacon node cannot, since Fusaka — see `--blob-serving` above).

- **`--l1-execution-rpc-url`** / **`--l1-beacon-url`**

*Description*: An external L1's execution JSON-RPC and beacon REST endpoints. Both must be set together to use an external L1 (a third-party provider, or an L1 not managed by nodevin); setting only one is an error.
*Default*: unset — see auto-attach below.
*Usage*: `--l1-execution-rpc-url=<url> --l1-beacon-url=<url>`

*Auto-attach*: if neither flag is set, nodevin looks for its own already-running Ethereum node instead — the same execution/consensus client `--execution-client`/`--consensus-client` would pick for `nodevin start ethereum` (mainnet, or the Sepolia stack with `--testnet`) — and, if both of those containers are actually running, connects to them directly by container name over their stack's own Docker network. If it isn't running, `start` fails before pulling the L2's multi-GB image, naming the missing container(s) and suggesting `nodevin start ethereum ... --blob-serving` or the two `--l1-*-url` flags instead.

*What to know before relying on this*: a freshly-started (or non-blob-serving) consensus client only has blobs from around when it started — see the "Start early" note under `--blob-serving` above. An L2 node pointed at such an L1 will not find the blobs its batches need until the L1 has run long enough.

Example, mainnet, auto-attached to a local Ethereum node:
```bash
nodevin start ethereum --blob-serving
nodevin start arbitrum
```

Example, testnet, external L1:
```bash
nodevin start base --testnet \
  --l1-execution-rpc-url=https://ethereum-sepolia.example.com \
  --l1-beacon-url=https://beacon-sepolia.example.com
```

*Arbitrum*: `nodevin start arbitrum` runs `fiftysix/nitro`, a single monolithic process — no execution/consensus split, no JWT of its own. `--chain.id`/`--parent-chain.id` are set automatically (`42161`/`1` on mainnet, `421614`/`11155111` with `--testnet`). JSON-RPC publishes on `127.0.0.1:8552`, WebSocket on `127.0.0.1:8553`, and the sequencer feed (relayed to other nodes, not admin-level) on `9642` — Sepolia uses `8556`/`8557`/`9643` so both can run at once.

- **`--snapshot`**

*Description*: Initialises a fresh datadir from the latest official Arbitrum snapshot ([snapshot.arbitrum.foundation](https://snapshot.arbitrum.foundation)) before starting, instead of syncing from genesis (impractical on mainnet). `nodevin` drives Nitro's own snapshot download/verify/extract flow rather than reimplementing it — it never downloads or extracts anything itself.
*Options*: `pruned` (hash-scheme database — recommended), `full-path` (newer path-scheme database — see the warning below)
*Default behavior when unset*: unchanged — syncs from genesis, exactly as before this flag existed.
*Usage*: `--snapshot` (bare, means `pruned`) or `--snapshot=full-path`

This runs as a two-phase start the first time: phase 1 runs `fiftysix/nitro` once, in the foreground, with `--init.latest=<kind> --init.download-path=<staging> --init.then-quit` (plus the same `--chain.id`/`--parent-chain.id`/L1 flags phase 2 uses — Nitro reads the parent chain during init, before any snapshot download begins); phase 2 is the normal, unchanged node start. Nitro's own output streams live — this can take 12–24 hours on mainnet.

**`--snapshot` never re-initialises.** If the datadir already has a real chain database (Nitro has created `<datadir>/<chain>/nitro/l2chaindata`), phase 1 is skipped with an informational log and phase 2 starts normally — `--snapshot` against an already-synced node is a no-op, never a wipe.

**A pre-flight disk check runs before phase 1 starts.** `nodevin` reads the snapshot's real size from its published `metadata.json` and refuses — before downloading anything — if either the download path or the datadir doesn't have comparable free space. It will not start a multi-hundred-GB-to-multi-TB download that cannot finish.

**If phase 1 fails or is interrupted, nothing is deleted.** The downloaded archive is left exactly where it was; re-running the same command resumes the download (Nitro's own behavior) instead of restarting it from zero.

**`--snapshot=full-path` is opt-in for a reason, and does not work yet.** Its snapshot uses Nitro's path-scheme database (confirmed from the snapshot's own `metadata.json`: `"state_scheme": "path"`, vs. `"hash"` for `pruned`), and [OffchainLabs/nitro#4746](https://github.com/OffchainLabs/nitro/issues/4746) reports a path-scheme snapshot bootstrap that can stall silently. Separately, Nitro's `--init.latest` accepts only `archive`, `pruned` and `genesis`: it refuses `full-path` ("invalid value for latest option") on **both** 3.11.4 and 3.12.1, the newest stable release, so a newer Nitro will not fix this by itself (an earlier version of this page wrongly said it would). Giving Nitro the snapshot's directory directly with `--init.url` does work: a real partial download of `full-path.tar.part0000` started and wrote to the staging path on 3.12.1. Supporting `full-path` would therefore mean passing `--init.url=<resolved directory>` instead of `--init.latest`, and starting phase 2 with `--execution.caching.state-scheme=path` so Nitro opens the resulting path-scheme database (its default is `hash`). Neither step is done, and the combination has not been verified end to end. Use the default (`pruned`).

- **`--snapshot-download-path`**

*Description*: Where the snapshot archive is downloaded and verified before extraction. Needs free space comparable to the snapshot itself (multiple TB for Arbitrum One), separate from the datadir it extracts into — the two coexist on disk during extraction. Point this at its own large volume for a real mainnet sync.
*Default*: `<nodevin data dir>/.snapshot-staging` (shared across runs/networks, not per-network, so one override covers both mainnet and Sepolia)
*Usage*: `--snapshot-download-path=/mnt/big-disk/arbitrum-snapshot-staging`

*What was verified*: the disk pre-flight check, real — it fetched Arbitrum Sepolia's actual published snapshot metadata (1.60 TB at the time) and correctly refused on a machine without that much free space, before attempting any download. Skipping an already-initialised datadir was verified with a real populated-looking datadir (the data survived untouched). Resolving L1 endpoints before phase 1 — both auto-attach and an explicit external L1 — was verified the same way `--l1-execution-rpc-url`/`--l1-beacon-url` are verified below. `nodevin start arbitrum` with no `--snapshot` at all was re-verified unchanged.

**Placement and resume were verified with a real partial download**, against the real Sepolia pruned snapshot: `pruned.tar.part0000` landed exactly at `--init.download-path` as expected, was killed partway through at ~40.6 MB, and restarting the identical command resumed from that same file (it was at ~41.0 MB within 6 seconds of restart and kept climbing at the same rate as before — not reset to 0, which a cold restart would have been). Confirms both that staging is correctly placed for a future full sync and that nothing on nodevin's side needs to do anything for resume to work.

**Not verified**: a full snapshot download, extraction, and resulting real sync (`eth_blockNumber` tracking a public endpoint) — this needs multiple TB of free disk and 12+ hours, neither available in the environment this was built in.

*Base*: `nodevin start base` runs `fiftysix/base-reth` (execution) and `fiftysix/base-consensus` (rollup/consensus) together, mirroring Ethereum's execution/consensus pairing — but unlike every Ethereum client here, the Engine API JWT between them is a **shared value**, not a shared file: nodevin generates one 32-byte secret per stack on first use, persists it (`~/.nodevin/data/base[-testnet]/jwt.hex`), and gives the same value to both containers as `BASE_NODE_L2_ENGINE_AUTH_RAW`, each of which writes its own local copy on startup. base-reth's JSON-RPC publishes on `127.0.0.1:8558` (`8563` testnet), base-consensus's on `127.0.0.1:9545` (`9546` testnet). Not yet snapshot-synced by nodevin; see [node-images' Base docs](https://github.com/fiftysixtech/node-images/blob/main/docs/base.md).

*What was verified*: both auto-attach (a real local Sepolia stack, container-name DNS resolution across separate compose files confirmed working) and an external L1 URL. `fiftysix/nitro` and `fiftysix/base-reth` both correctly resolved their L1's chain ID and connected over plain JSON-RPC; `fiftysix/base-consensus` completed the Engine API JWT handshake with `fiftysix/base-reth` (confirmed both containers wrote the identical secret) and began deriving from real L1 head data. Not run on mainnet. A real chain sync was not attempted against either — the local test L1 used didn't have deep-enough history for Nitro's snapshot-init flow to succeed, a data-depth limitation of the test environment, not the wiring.

#### BNB Smart Chain options

`nodevin start bsc` runs `fiftysix/bsc` — a BNB Smart Chain (BSC) node. Unlike Arbitrum and Base, BSC has no L1 dependency: it's a geth fork with the Parlia consensus engine built directly into the same monolithic process, so there is no execution/consensus split and no `--l1-*` flags apply here.

BSC has no built-in chain config nodevin can select with a plain flag the way Ethereum mainnet/Sepolia have, so `fiftysix/bsc` bundles both networks' genesis/config and picks between them via an internal `NETWORK` environment variable, set automatically from `--testnet` — there is no separate CLI flag for it.

JSON-RPC publishes on `127.0.0.1:8565` (`8567` testnet), WebSocket on `127.0.0.1:8566` (`8568` testnet), and P2P (BSC's own default port, `30311`, not Ethereum's `30303`) on `30311` (`30312` testnet) — mainnet and testnet use different host ports so both can run at once, same as Arbitrum/Base.

```bash
nodevin start bsc
nodevin start bsc --testnet
```

Not yet snapshot-synced by nodevin — syncing from genesis is impractical on real BSC mainnet; see [node-images' BNB Smart Chain docs](https://github.com/fiftysixtech/node-images/blob/main/docs/bnb-smart-chain.md) for the manual `bnb-chain/bsc-snapshots` flow.

*What was verified*: `nodevin start bsc` and `nodevin start bsc --testnet` both self-initialise from the bundled genesis on first start (no manual `geth init` step) and serve RPC with the correct chain ID — `0x38` (56) on mainnet, `0x61` (97) on testnet. Not run against a real, already-populated chain; not run on mainnet in the sense of joining real BSC mainnet peers.

- **`--ipfs-cluster-image`**

*Description*: Docker image to use for `ipfs-cluster`.
*Default*: `fiftysix/ipfs-cluster`
*Usage*: `--ipfs-cluster-image=<docker-image>`

- **`--ipfs-cluster-version`**

*Description*: Version of the Docker image to use for `ipfs-cluster`.
*Default*: `latest`
*Usage*: `--ipfs-cluster-version=<tag>`

- **`--ipfs-cluster-peername`**

*Description*: (ipfs-cluster only) The peername(s) to attach to.
*Usage*: `--ipfs-cluster-peername=<name>`
*Example*: `--ipfs-cluster-peername=cluster-peer-1`

- **`--ipfs-cluster-secret`**

*Description*: (ipfs-cluster only) The cluster secret required for connection.
*Usage*: `--ipfs-cluster-secret=<secret>`

- **`--ipfs-cluster-bootstrap`**

*Description*: (ipfs-cluster only) The bootstrap node address.
*Usage*: `--ipfs-cluster-bootstrap=<multiaddr>`
*Example*: `--ipfs-cluster-bootstrap=/ip4/172.20.0.2/tcp/4001/p2p/12D3KooWHUZ36WvuUBmz5aFLJ9PoNKrUJRMSA22i98BkoAaQPRzi`

- **`--rpc-user`**

*Description*: Username passed in via command for JSON RPC.
*Default*: `user`
*Usage*: `--rpc-user=<username>`

- **`--rpc-pass`**

*Description*: Password passed in via command for JSON RPC.
*Default*: `fiftysix`
*Usage*: `--rpc-pass=<password>`

- **`--cookie-auth`**

*Description*: Use authentication directly with the node's cookie file instead of `--rpc-user`/`--rpc-pass`.
*Default*: `false`
*Usage*: `--cookie-auth`

- **`--restart`**

*Description*: Docker restart policy for the container.
*Default*: `no`
*Usage*: `--restart=<policy>`
*Example*: `--restart=always`

- **`--command`**

*Description*: Specifies the node command and its configuration options (e.g., RPC settings).
*Usage*: `--command="<command>"`
*Example*: `--command="--rpcallowip=0.0.0.0/0 -rpcuser=user -rpcpassword=pass"`

- **`--testnet`**

*Description*: Runs the node on a test network.
*Default*: `false`
*Usage*: `--testnet`

- **`--snapshot-sync`**

*Description*: Starts a node by downloading data from a snapshot. **Currently disabled** — the flag is accepted but has no effect; the node syncs from genesis regardless.
*Default*: `false`
*Usage*: `--snapshot-sync`

- **`--snapshot-sync-command`**

*Description*: Runs a custom command for snapshot sync before the node starts (e.g., download and setup). **Currently disabled**, same as `--snapshot-sync` above.
*Usage*: `--snapshot-sync-command="<command>"`

- **`--data-dir`**

*Description*: Specifies the directory where nodevin and blockchain data will be stored.
*Usage*: `--data-dir="<file-path>"`
*Example*: `nodevin --data-dir="~/Desktop" start ipfs`

*Important*: if you use `--data-dir` with `start`, pass the same `--data-dir` to every later `stop`, `delete`, `logs`, `info`, and `view` for that node — nodevin looks for the node's compose file there. Without it, those commands fall back to `~/.nodevin` and will not find a node started elsewhere.

#### Docker & Container Options:

- **`--image`**

*Description*: Specifies the Docker image to use for the node.
*Usage*: `--image=<docker-image>`
*Example*: `--image=fiftysix/bitcoin-core`

- **`--version`**

*Description*: Version of the Docker image to use.
*Usage*: `--version=<tag>`
*Example*: `--version=27.0`

- **`--container-name`**

*Description*: Name of the Docker container.
*Usage*: `--container-name=<name>`
*Example*: `--container-name=bitcoin-node`

- **`--docker-networks`**

*Description*: Networks the Docker container connects to.
*Usage*: `--docker-networks=<network1,network2,...>`
*Example*: `--docker-networks=network1,network2`

- **`--network-driver`**

*Description*: Docker network driver (e.g., bridge).
*Usage*: `--network-driver=<driver>`
*Example*: `--network-driver=bridge`

- **`--ports`**

*Description*: Port mappings for the node container.
*Usage*: `--ports="<port1:port1,port2:port2,...>"`
*Example*: `--ports="127.0.0.1:8332:8332,8333:8333"`

*Note*: `--ports` replaces the node's default mappings entirely, so list every port you want published. For IPFS the defaults are `4001:4001`, `127.0.0.1:5001:5001` (RPC API) and `127.0.0.1:8080:8080` (gateway): the API and gateway are only reachable from the machine running the node. The API has admin-level access, so only publish it on other interfaces (for example `--ports="4001:4001,0.0.0.0:5001:5001"`) if you have put authentication or a firewall in front of it.

*Note on RPC ports*: for every chain the JSON-RPC port is published on `127.0.0.1` only, and peer ports stay public so other nodes can connect: Bitcoin `127.0.0.1:8332` + `8333`, Litecoin `127.0.0.1:9332` + `9333`, Dogecoin `127.0.0.1:22555` + `22556`, Ethereum `127.0.0.1:8547` + `30305` (Sepolia `127.0.0.1:8549` + `30306`), Ethereum Classic `127.0.0.1:8545` + `30303` (testnets use their own ports). To reach a node's RPC from another machine, list the mapping yourself, for example `--ports="0.0.0.0:8332:8332,8333:8333"`, and set your own `--rpc-user`/`--rpc-pass`: the defaults (`user`/`fiftysix`) are public, RPC is plain HTTP, and Ethereum Classic's RPC has no authentication at all. Prefer an SSH tunnel or a firewall rule to publishing RPC on a public interface. The `ord` web interface and the `ipfs-cluster` REST API are unchanged.

- **`--volumes`**

*Description*: Docker volumes to mount.
*Usage*: `--volumes="<volume1:/path1,...>"`
*Example*: `--volumes="bitcoin-core-data-2:/node/bitcoin-core"`

- **`--volume-definitions`**

*Description*: Defines the Docker volumes in the compose file.
*Usage*: `--volume-definitions=<volume-name>`
*Example*: `--volume-definitions="bitcoin-core-data-2"`

- **`--volume-labels`**

*Description*: Custom labels for Docker volumes.
*Usage*: `--volume-labels="<label-key=value,...>"`
*Example*: `--volume-labels="nodevin.blockchain.software=bitcoin-core-testnet"`

#### Resource Management Options:

- **`--cpu-limit`**

*Description*: Maximum CPU limit for the container.
*Usage*: `--cpu-limit=<value>`
*Example*: `--cpu-limit=2.0`

- **`--mem-limit`**

*Description*: Maximum memory limit for the container.
*Usage*: `--mem-limit=<value>`
*Example*: `--mem-limit=1g`

- **`--cpu-reservation`**

*Description*: Reserved CPU resources for the container.
*Usage*: `--cpu-reservation=<value>`
*Example*: `--cpu-reservation=1.0`

- **`--mem-reservation`**

*Description*: Reserved memory resources for the container.
*Usage*: `--mem-reservation=<value>`
*Example*: `--mem-reservation=512m`

 
#### Example Usage:
```bash
nodevin start bitcoin \
--ord \
--command="--rpcallowip=0.0.0.0/0 -rpcuser=user -rpcpassword=pass" \
--testnet \
--image=fiftysix/bitcoin-core \
--version=27.0 \
--container-name=bitcoin-node \
--ports="127.0.0.1:8332:8332,8333:8333" \
--restart=always \
--cpu-limit=2.0 \
--mem-limit=1g \
--cpu-reservation=1.0 \
--mem-reservation=512m
```

---

### `nodevin stop`

- **Description**: Stops a running blockchain node for the specified network.
- **Simple Example**: `nodevin stop bitcoin`

#### Options:

- **`--testnet`**

*Description*: Stops the node running on a test network.
*Usage*: `nodevin stop <network> --testnet`

- **`--network=<network>`**

*Description*: Specify a custom network.
*Usage*: `nodevin stop <network> --network="goerli"`

- **`stop ethereum`**

*Description*: Stops the whole Ethereum stack (execution and consensus client). Nodevin finds the client that is running; if several are, pass `--execution-client=<name>`. The consensus clients cannot be stopped on their own: `nodevin stop lighthouse` (and prysm, teku, nimbus, lodestar) prints an error pointing to `nodevin stop ethereum`.
*Usage*: `nodevin stop ethereum`

---

### `nodevin update`

- **Description**: Checks for and applies updates to the Nodevin binary itself, or updates the Docker images for running nodes.
- **Simple Example**: `nodevin update`

#### Options:

- **`update`**

*Description*: Checks for and applies a Nodevin software update.
*Usage*: `nodevin update`

- **`update docker`**

*Description*: Checks for and applies updates to the Docker images used by running nodes.
*Usage*: `nodevin update docker`

---

### `nodevin shell`

- **Description**: Opens an interactive shell in the running container for the specified blockchain network.
- **Simple Example**: `nodevin shell bitcoin`

#### Options:

- **`--detach`**

*Description*: Runs the shell in detached mode (in the background).
*Usage*: `nodevin shell <network> --detach`

- **`--docker-user=<user>`**

*Description*: Specifies the username or UID to run the shell as inside the container.
*Usage*: `--docker-user=root`

- **`--workdir=<path>`**

*Description*: Sets the working directory inside the container.
*Usage*: `--workdir=/node`

- **`--env=<key=value>`**

*Description*: Sets environment variables.
*Usage*: `--env=KEY=VALUE`

- **`--env-file=<file>`**

*Description*: Reads environment variables from a file.
*Usage*: `--env-file=./env.list`

- **`--privileged`**

*Description*: Runs the shell with extended privileges.
*Usage*: `--privileged`

---

### `nodevin logs`

- **Description**: Fetches logs for a running blockchain node.
- **Simple Example**: `nodevin logs bitcoin --tail 20`

#### Options:

- **`--follow`**

*Description*: Continuously stream the logs in real-time.
*Usage*: `nodevin logs <network> --follow`

- **`--tail=<value>`**

*Description*: Shows a specific number of lines from the end of the logs.
*Usage*: `--tail=100`

---

### `nodevin request`

- **Description**: Makes an RPC request to a specified blockchain network.
- **Simple Example**: `nodevin request bitcoin --method getblockcount`
- **Ethereum Example**: `nodevin request ethereum --method eth_blockNumber` (talks to the execution client's JSON-RPC on `127.0.0.1:8547`)

#### Options:

- **`--method`**

*Description*: Specifies the HTTP method to use for the request (e.g., GET, POST).
*Usage*: `--method=<http-method>`
*Example*: `--method=getblockcount`

- **`--params`**

*Description*: JSON data to send as parameters in the request body.
*Usage*: `--params=<json-data>`
*Example*: `--params='["param1", "param2"]'`

*Example with parameters*:
```bash
nodevin request bitcoin --method getblockheader --params '["00000000c937983704a73af28acdec37b049d214adbda81d7e2a3dd146f6ed09"]'
```

- **`--header`**

*Description*: Adds optional extra headers to the request.

*Usage*: `--header=<key:value>`

- **`--endpoint`**

*Description*: Specifies an optional API endpoint (default is `http://127.0.0.1`).
*Usage*: `--endpoint=<url>`

- **`--port`**

*Description*: Optional port to override the default RPC port for the network.
*Usage*: `--port=<port>`

---

### `nodevin info`

- **Description**: Displays information about currently running blockchain nodes, including version, status, ports, peer count, and latest block (for supported chain software, which includes the Ethereum execution clients; consensus clients are listed without peer and block data).
- **Simple Example**: `nodevin info`

#### Options:

- **`nodevin info <network>`**

*Description*: Filters the output to a specific network.
*Usage*: `nodevin info bitcoin`

---

### `nodevin view`

- **Description**: Displays a fun, artistic ASCII-art view of your currently running Nodevin nodes and their stats.
- **Simple Example**: `nodevin view`

---

### `nodevin delete`

- **Description**: Deletes local blockchain data associated with a specific network. `delete` stops the node first; if it is still running afterward (for example because the node's `--data-dir` was not also passed to `delete`), it refuses to remove the data and exits with an error rather than deleting a running node's files.
- **Simple Example**: `nodevin delete bitcoin`

#### Options:

- **`delete <network>`**

*Description*: Deletes nodevin data for a network.
*Usage*: `nodevin delete bitcoin`

- **`delete <network> --testnet`**

*Description*: Deletes nodevin data for a network's testnet.
*Usage*: `nodevin delete bitcoin --testnet`

- **`delete ethereum --execution-client=<name>`**

*Description*: Deletes the data of one Ethereum execution client. Because several clients can have data on disk and deletion can't be undone, the client must always be named; nodevin never infers it. Consensus client data is left in place.
*Usage*: `nodevin delete ethereum --execution-client=geth`

- **`delete <consensus-client>`**

*Description*: Deletes one consensus client's data (for example to re-sync it from a checkpoint). It refuses while that client is running; run `nodevin stop ethereum` first.
*Usage*: `nodevin delete lighthouse`

- **`delete all`**

*Description*: Deletes all nodevin data.
*Usage*: `nodevin delete all`

---

### `nodevin cleanup`

- **Description**: Deletes all local Docker images with prefix `fiftysix`.
- **Simple Example**: `nodevin cleanup`

---

## Env File

Nodevin supports using an `.env` file for easy configuration. Note that variables set in the `.env` file will be overridden by command-line flags.

### `.env` File Location:

Nodevin will look for the `.env` file in the following locations, in order of priority:

1. **Current Working Directory (CWD)**: The directory where you run the Nodevin command.
2. **User-Specific Directory**: `~/.nodevin/.env` (e.g., `/home/user/.nodevin/.env` on Linux or `C:\Users\YourUser\.nodevin\.env` on Windows).
3. **Global Configuration Directory**: `/etc/nodevin/.env`.
4. **Executable Directory**: The directory containing the Nodevin binary (e.g., `/usr/local/bin/.env`).

Only the first `.env` file found is used; its values are not merged with any of the others. Keys are flag names without the leading `--` (e.g. `rpc-user`, `data-dir`); `RPC_USER` and `rpc_user` are also accepted and treated the same as `rpc-user`. Values may be unquoted, double-quoted, or single-quoted, and an unquoted value may end in a `# comment`. Flags that take a comma-separated list on the command line (`--ports`, `--volumes`, `--volume-definitions`, `--docker-networks`) accept the same comma-separated form here, and `--volume-labels` accepts `key=value` pairs separated by commas. A line that cannot be parsed is skipped with a warning; it does not prevent the rest of the file from loading.

### Example `.env` File

Here’s an example `.env` file for Nodevin:

```bash
# Docker Configuration
image=fiftysix/bitcoin-core
version=27.0
restart=always
cpu-limit=2.0
mem-limit=2g
cpu-reservation=1.0
mem-reservation=1g

# Nodevin Configuration
data-dir=/home/user/.nodevin

# Chain Software Configuration
rpc-user=admin
rpc-pass=securepassword123

# Bitcoin-Specific Configuration
ord-image=fiftysix/ord
ord-version=latest

# Litecoin-Specific Configuration
ord-litecoin-image=fiftysix/ord-litecoin
ord-litecoin-version=latest
```

Be careful setting other configs, as they may interfere with Nodevin's automatic node detection.
