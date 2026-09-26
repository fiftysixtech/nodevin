# Nodevin

Nodevin allows anyone to run blockchain nodes effortlessly. It simplifies the process of setting up and managing nodes for various blockchains, ensuring they are always up-to-date with the latest software versions. With Nodevin, you can run nodes for Bitcoin, Litecoin, Dogecoin, Ethereum, Ethereum Classic, and IPFS with ease.

Our goal is to facilitate blockchain node standup and maintenance for every chain in the world, in service of a more decentralized internet.

Nodevin is built and maintained by [Fiftysix](https://fiftysix.tech?utm_source=github&utm_medium=readme&utm_campaign=nodevin), in conjunction with [Kaisersolver](https://kaisersolver.com).

## Features

- **Easy Setup:** Quickly set up blockchain nodes with a single command.
- **Automatic Updates:** Nodevin ensures your nodes are always running the latest software versions.
- **Maximum Customization:** Set unique ports, data storage, networking, images, or even run multiple nodes at once.
- **Cross-Platform Support:** Works on Linux, macOS, and Windows.

## Getting Started

### Setup

Stand up a blockchain node in three steps.

1. **Download Nodevin:**

Download the latest version of Nodevin from the [releases page](https://github.com/fiftysixtech/nodevin/releases).

2. **Initialize Nodevin and Docker:**

```sh
nodevin init
```

*For more information setting up Nodevin on Windows, read [these docs](./docs/windows-setup.md).*

3. **Start a Blockchain Node:**

Once Nodevin is initialized, you can start a blockchain node. For example, to start a Bitcoin node, run:

```sh
nodevin start bitcoin
```

*Nodevin stores blockchain data by default in `$HOME/.nodevin`.*

**Ethereum** runs an execution client and a consensus client together. A consensus client can't sync mainnet from genesis, so you choose a checkpoint sync provider you trust ([public list](https://eth-clients.github.io/checkpoint-sync-endpoints/)):

```sh
nodevin start ethereum --checkpoint-sync-url <provider-url>
```

By default this runs reth with Lighthouse. Pick others with `--execution-client` (reth, geth, erigon, besu, nethermind) and `--consensus-client` (lighthouse, prysm, teku, nimbus, lodestar). Add `--testnet` for the Sepolia testnet. See [Ethereum options](./docs/cli-commands.md#ethereum-options).

---

#### **(Optional) - Advanced Features:**

Nodevin allows for full customization in node startup. View the full list of flags and configuration details, including `.env` file setup [here](./docs/cli-commands.md). For example, this command runs a Bitcoin Testnet node with a specified command, docker image and tag (version), unique nodevin data directory, and more:

```sh
nodevin start bitcoin \
  --ord \
  --command="--rpcallowip=0.0.0.0/0 -rpcuser=user -rpcpassword=pass" \
  --testnet \
  --image=fiftysix/bitcoin-core \
  --version=27.0 \
  --ports="127.0.0.1:8332:8332,8333:8333,127.0.0.1:18332:18332,18333:18333" \
  --data-dir="~/Desktop" \
  --restart=always \
  --cpu-limit=2.0 \
  --mem-limit=1g \
  --cpu-reservation=1.0 \
  --mem-reservation=512m
```

#### **(For Linux/MacOS) - Set Nodevin Permissions:**

After downloading Nodevin, you may need to set executable permissions and move it to a directory in `$PATH`.

```sh
chmod +x nodevin
sudo mv nodevin /usr/local/bin/
```

### More Documentation

Continue to learn about Nodevin:
- [Introduction to Nodevin](./docs/nodevin-intro.md) - Learn more about what Nodevin is and how it works.
- [Using Nodevin](./docs/cli-commands.md) - Documentation on how to use Nodevin. 
- [Nodevin Blog](...) - Coming soon... 

## Snapshot Synchronization

**NOTE: Snapshot Synchronization is currently disabled.**

Data snapshots are compressed archives of the state of a blockchain node. Using snapshot synchronization greatly speeds up the process of catching up with the network, as the node starts from downloaded data rather than trying to synchronize from the beginning of the blockchain. Running this command will use snapshot synchronization when starting up your node.

```
nodevin start litecoin --snapshot-sync
```

Snapshot synchronization can save up to **days** of node initialization.

## Integrating Your Blockchain

Want your blockchain supported by Nodevin? Reach out to us directly at [hello@fiftysix.tech](mailto:hello@fiftysix.tech) and we'll work with you to get it added.

### Nodevin Docker Images

Nodevin pulls images by default from [Docker Hub](https://hub.docker.com/u/fiftysix), with code located [here](https://github.com/fiftysixtech/node-images). The node image respository contains helpful resources including blockchain requirements and synchronization times, Docker installation steps, Docker compose files with documentation, and more.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request or open an Issue on GitHub.

## License

This project is licensed under the Apache License 2.0. See the [LICENSE](LICENSE) file for details.

## Contact

- [Website](https://nodevin.xyz)
- [Docker Hub](https://hub.docker.com/u/fiftysix) - All Nodevin images are pulled from here.
- [node-images](https://github.com/fiftysixtech/node-images) - Source for every Docker image Nodevin uses.

This repository is currently maintained by [Fiftysix](https://fiftysix.tech?utm_source=github&utm_medium=readme&utm_campaign=nodevin).

For any questions or suggestions, feel free to contact us at [hello@fiftysix.tech](mailto:hello@fiftysix.tech).

---

Thank you for using Nodevin! Happy node running!
