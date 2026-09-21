# Nodevin CLI Documentation

Nodevin is a command-line interface (CLI) that simplifies the setup, management, and running of blockchain nodes. Below is a comprehensive list of the available commands and their options.

---

## Table of Contents

**Note**: On Windows, commands should be written as `nodevin.exe <command>`, not `nodevin <command>`. For example, `nodevin.exe init`.

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

*Note on ord web ports*: each `ord` instance publishes its web interface on its own host port so several can run at once: `ord` on `80`, `ord` (testnet) on `8081`, `ord-litecoin` on `8082`, and `ord-litecoin` (testnet) on `8083`. Override with `--ports`.

- **`--ipfs-cluster`**

*Description*: Runs `ipfs-cluster` software alongside the IPFS node.
*Default*: `false`
*Usage*: `--ipfs-cluster`

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

*Description*: Starts a node by downloading data from a snapshot.
*Default*: `false`
*Usage*: `--snapshot-sync`

- **`--snapshot-sync-command`**

*Description*: Runs a custom command for snapshot sync before the node starts (e.g., download and setup).
*Usage*: `--snapshot-sync-command="<command>"`

- **`--data-dir`**

*Description*: Specifies the directory where nodevin and blockchain data will be stored.
*Usage*: `--data-dir="<file-path>"`
*Example*: `nodevin --data-dir="~/Desktop" start ipfs`

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
*Example*: `--ports="8332:8332,8333:8333"`

*Note*: `--ports` replaces the node's default mappings entirely, so list every port you want published. For IPFS the defaults are `4001:4001`, `127.0.0.1:5001:5001` (RPC API) and `127.0.0.1:8080:8080` (gateway): the API and gateway are only reachable from the machine running the node. The API has admin-level access, so only publish it on other interfaces (for example `--ports="4001:4001,0.0.0.0:5001:5001"`) if you have put authentication or a firewall in front of it.

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
--ports="8332:8332,8333:8333" \
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

- **Description**: Displays information about currently running blockchain nodes, including version, status, ports, peer count, and latest block (for supported chain software).
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

- **Description**: Deletes local blockchain data associated with a specific network.
- **Simple Example**: `nodevin delete bitcoin`

#### Options:

- **`delete <network>`**

*Description*: Deletes nodevin data for a network.
*Usage*: `nodevin delete bitcoin`

- **`delete <network> --testnet`**

*Description*: Deletes nodevin data for a network's testnet.
*Usage*: `nodevin delete bitcoin --testnet`

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
