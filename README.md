# marshmallows

Secure mesh networking for IoT — an encrypted peer overlay with 2FA/U2F. A self-hostable, Tailscale-like secure overlay.

`Status: Active` · `Go · Noise Protocol` · part of [TrustSentinel](https://trustsentinel.eu)

## Overview
marshmallows links private clouds and IoT devices into a single secure overlay:
devices on separate networks become mutually reachable through encrypted peer
links, hardened with lightweight cryptography and two-factor (2FA/U2F)
authentication. Recognised at INCIBE's national cybersecurity competition (2019).

Components: `broker/` (coordination), `agent/` (device), `web/` (dashboard), `auth/` (identity).

## Design & vision
See [docs/iot-mesh-architecture.md](docs/iot-mesh-architecture.md) — how this grows
into a Tailscale-class mesh purpose-built for constrained / edge devices.

## License
MIT

<sub>Originally prototyped at bluebycode/marshmallows (2019); continued under TrustSentinel.</sub>
