# Introduction to Nodevin

**Nodevin** is an open-source tool that simplifies the process of running blockchain nodes. Whether you're a developer, enthusiast, or institution, Nodevin automates node setup, updates, and management—making it easy for anyone to participate in decentralized blockchain networks.

Nodevin is a **Command-Line Interface (CLI) tool**. This means it doesn't have a graphical user interface with buttons to click or windows to navigate. Instead, you interact with Nodevin by typing commands directly into your **command prompt** (Windows) or **terminal** (Linux/MacOS).

---

## Nodevin Quick Start Guide

1. **Check System Requirements**
   - **OS**: Linux, macOS, Windows (64-bit)
   - **Docker**: Version 20+ ([Get Docker](https://docs.docker.com/get-docker/))
   - **Docker Compose**: Included with latest versions of Docker
   - **CPU**: (Depends on blockchain)
   - **RAM**: (Depends on blockchain)
   - **Storage**: (Depends on blockchain)
   - **Network**: Reliable broadband with unlimited data

The `nodevin init` command can help you get set up and test system requirements.

2. **Install Nodevin**
   - Download the latest version from the [Nodevin GitHub Releases](https://github.com/fiftysixtech/nodevin/releases) or the [Nodevin Website](https://nodevin.xyz).
   - Ensure Docker and Docker Compose are installed (you can test this with `nodevin init`).

3. **Start a Blockchain Node**
   - Run: `nodevin list` to list all supported networks.
   - Run: `nodevin start <network>` (e.g., `nodevin start bitcoin`).

---

### Intro - Using a CLI Tool

1. **Open Your Command Prompt/Terminal**:
   - On **Windows**: Search for `cmd` or `Powershell` from the Start menu.
   - On **Linux/MacOS**: Open the Terminal application (usually found in your Applications menu).

2. **Running Commands**: 
   - Commands are simply instructions you type and execute by pressing "Enter." For example, to start a Bitcoin node, you'd type:
     ```
     nodevin start bitcoin
     ```
   - This command tells Nodevin to start a Bitcoin node, on mainnet, using Docker.

3. **No Mouse Required**: Unlike traditional software, you won't be using your mouse to interact. Every action, from starting a blockchain node to stopping it, is handled through typed commands.

### Why a CLI?

CLI tools like Nodevin are preferred by many for their **precision, efficiency, and flexibility**. They allow you to execute complex tasks quickly without navigating multiple screens. Nodevin leverages this power to make node management as straightforward as possible, with simple commands that abstract complex operations, all in the background.

---

## What is a Node?

A **blockchain node** is a computer that participates in a blockchain network by maintaining the blockchain's ledger and validating transactions. Nodes ensure decentralization by distributing control across multiple participants.

### Types of Nodes:
- **Full Node**: Stores and validates the entire blockchain.
- **Light Node**: Stores only part of the blockchain and queries full nodes.
- **More**: [Read more](https://getblock.io/blog/blockchain-node-types/).

---

## How Nodevin Simplifies Node Setup

Nodevin removes the complexities of running blockchain nodes. Normally, setting up a node requires technical expertise and configuration. Nodevin does all this automatically by:
- Using Docker to isolate the node software.
- Handling updates with [Watchtower](https://containrrr.dev/watchtower/).
- Running nodes with a simple command: `nodevin start <blockchain>`.
- In depth information about each blockchain: `nodevin info`.

---

## Mining, Running a Node, and Staking: What's the Difference?

### Running a Node
- **Purpose**: Validates transactions and maintains the blockchain.
- **Rewards**: Typically no direct rewards, but essential for decentralization.
- **Hardware**: Requirements are not as restrictive, but depend on blockchain.
  
### Mining (Proof of Work)
- **Purpose**: Solves puzzles to create new blocks.
- **Rewards**: Earns block rewards and transaction fees.
- **Hardware**: Requires specialized hardware (e.g., ASICs for Bitcoin).

### Staking (Proof of Stake)
- **Purpose**: Locks cryptocurrency to validate transactions and create new blocks.
- **Rewards**: Earns rewards based on the amount staked.
- **Hardware**: Requires much less power and specialized equipment than mining.

Nodevin currently only supports running nodes to help users contribute to the network in various ways.

---

## What is Docker and Why Does Nodevin Use Docker?

[Docker](https://docs.docker.com/get-started/) is a platform that bundles software into containers, which include everything needed to run the software. Nodevin uses Docker because:
- **Isolation**: Docker keeps your node separate from the rest of your system.
- **Standardization**: It ensures consistent performance across different systems.
- **Accessibility**: The [Docker Hub](https://hub.docker.com/u/fiftysix) allows anyone to access Nodevin built software versions and updates.

By using Docker, Nodevin makes deploying and managing blockchain nodes easy and efficient, no matter your technical background.

---

## Uninstalling Nodevin

To uninstall Nodevin and delete all associated data and docker images, run the following commands:

```sh
nodevin stop all
nodevin cleanup
nodevin delete all
rm nodevin # delete nodevin
```

---

## More Resources
- [Main README](../README.md)
- [Nodevin Commands](./cli-commands.md)
- [Nodevin Windows Installation](./windows-setup.md)
