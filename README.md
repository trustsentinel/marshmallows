# marshmallows

<p align="center">
  <img width="600" src="docs/assets/img/mm_brand.png" alt="marshmallows">
</p>

<p align="center">Secure mesh networking for IoT — an encrypted peer overlay that makes separate private clouds mutually reachable, hardened with lightweight crypto and 2FA/U2F.</p>

<p align="center">
  <code>Status: Active</code> · <code>Go · React · Noise Protocol</code> · part of <a href="https://trustsentinel.eu">TrustSentinel</a>
</p>

<p align="center">
  <a href="https://www.incibe.es/"><img src="docs/assets/img/incibe.png" alt="INCIBE — Instituto Nacional de Ciberseguridad" height="64"></a>
</p>
<p align="center">
  <sub>🏆 Recognized at the <b>INCIBE National Cybersecurity Competition 2019</b> — a lightweight-cryptography secure-mesh architecture for IoT.</sub>
</p>

<br/>

<p align="center">
  <img src="docs/assets/img/dashboard.png" width="880" alt="marshmallows dashboard — mini-cloud topology, device management, and a browser terminal" />
</p>

<p align="center">
  <sub><b>The marshmallows dashboard</b> &nbsp;·&nbsp; mini-cloud topology &nbsp;·&nbsp; device management &nbsp;·&nbsp; a Noise-encrypted browser terminal</sub>
</p>

<br/>

## Overview

marshmallows is a Platform-as-a-Service for managing and securely connecting IoT
devices. Each deployment runs as a self-contained **mini-cloud** that IoT devices
attach to; separate mini-clouds — on different networks or infrastructures — can
associate with one another so their devices become mutually visible and reachable,
according to the privileges each administrator defines.

Its goal is **SSH access to any connected device from anywhere** in the
infrastructure — from a classic terminal or from the web dashboard — without users
needing to know IP addresses that may change over time. Every device is catalogued
with an alphanumeric identifier; its address is registered with an intermediary
**broker** that verifies access permissions and brokers the connection between
machines.

As a proof of concept, each platform ships a customized Debian image for Raspberry
Pi, and could target low-power *Thingy*-class devices. It's an ideal way to manage
and segment devices into domains by function or by whatever criteria the operators
choose.

<p align="center">
  <img height="420" src="docs/assets/img/infr_schema.png" alt="marshmallows infrastructure">
</p>

## Components
One Go module plus the dashboard (the 2019 prototype is preserved under `_legacy/`):
- **[`cmd/mm-broker`](cmd/mm-broker)** — the coordinator: agents attach over Noise, a registry drives `/devices.json`, and browser terminals are bridged to agents.
- **[`cmd/mm-agent`](cmd/mm-agent)** — the device-side agent (Go, Noise transport, PTY shell): pins the broker, enrolls with a one-time token, opens no inbound port.
- **[`cmd/mm-auth`](cmd/mm-auth)** — the identity service: WebAuthn (security keys / U2F) + TOTP.
- **[`web/`](web)** — the React dashboard: device management + a browser terminal.

## Run the demo
The whole platform on your machine — dashboard, broker, a live device, and identity:
```bash
docker compose -f deploy/compose/compose.yml up --build
```
Then open **http://localhost:8080**, pick the device, and **Connect** for a real
brokered shell. See **[deploy/compose/](deploy/compose)**.

## Objectives
1. **Manage devices across separate deployments** without them having to be
   individually identified on the network.
2. **SSH management between devices, or from the web portal** — which requires a
   broker that provides a protected end-to-end link.
3. **Interconnect several marshmallows nodes** so their devices are visible and
   reachable across mini-clouds, linked via a web-hook mechanism between brokers.

## Advantages
- **Cost** — lightweight provisioning, no added overhead.
- **Scalable & flexible** — new devices are recognised automatically, named, and
  offered for connection.
- **Location-independent** — reach a device without knowing where it sits on the
  network, so devices stay flexible and mobile.
- **Performance** — communication is lightweight *and* secure, leaving the device
  free to spend its resources on its actual job.
- **Security** — traffic is brokered over lightweight, secure channels; protected
  by the **Noise Protocol Framework** with end-to-end Diffie-Hellman encryption.
- **Maintenance** — devices can move from one cloud to another and be reassigned a
  different IP without the end user noticing.

## What makes it different
- **Reinforced authentication** — physical-key login via Yubico/FIDO keys (U2F),
  or OTPs.
- **Runs on any infrastructure** — physical networks, local networks, or Docker
  containers.
- **Secure comms via NPF** ([Noise Protocol Framework](https://noiseprotocol.org/)),
  avoiding the overhead and heavy keys of TLS.
- **Human-friendly identifiers** — each element gets an alias scoped by cloud and
  device (`cloud_id.device_id[.service_id]`), e.g. `d1d00a.5g0a` or `d1d00a.5g0a.ssh`.

## Internals
End-to-end secure channel and agent flow:

<p align="center">
  <img width="480" src="docs/diagrams/e2e.jpg" alt="end-to-end channel">
  &nbsp;
  <img width="300" src="docs/diagrams/agent-diagram.jpg" alt="agent">
</p>

## Tech stack
- **Back end** — Go, one module (`mm-broker`, `mm-agent`, `mm-auth`)
- **Transport** — Noise Protocol (Curve25519 · ChaCha20-Poly1305 · BLAKE2b)
- **Front end** — React 18 + Vite + xterm.js
- **Identity** — WebAuthn (security keys / U2F) + TOTP
- **Deploy** — Docker Compose (single-command demo)

## Design & vision
See the **[whitepaper](docs/whitepaper.md)** — architecture, the security model,
identity, and how marshmallows grows into a Tailscale-class mesh purpose-built
for constrained/edge devices. It sits alongside the wider TrustSentinel estate:
it seeds the Go Noise agent used by [stk](https://github.com/trustsentinel/stk),
and its mesh model feeds the [netso](https://github.com/trustsentinel/netso)
connectivity platform.

## TrustSentinel
Part of [TrustSentinel](https://trustsentinel.eu) — secure connectivity and
network-intelligence tooling by Álvaro López.

- **[netso](https://github.com/trustsentinel/netso)** — secure-networking platform (SSI + end-to-end encryption)
- **[stk](https://github.com/trustsentinel/stk)** — browser-based remote shell broker
- **[stuk](https://github.com/trustsentinel/stuk)** — SSH access gating (port-knock + MFA)
- **[marshmallows](https://github.com/trustsentinel/marshmallows)** — secure mesh for IoT  ·  _this repo_
- **[argos](https://github.com/trustsentinel/argos)** — P2P blockchain network scanning
- **[eth-rlp](https://github.com/trustsentinel/eth-rlp)** — RLP codec for Ethereum discv4

## License
MIT — see [LICENSE.md](LICENSE.md).

<sub>Originally prototyped at bluebycode/marshmallows (CyberCamp / INCIBE 2019);
continued under TrustSentinel. Translated from the original Spanish and updated.</sub>
