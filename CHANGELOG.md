# Changelog

All notable changes to this project are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres
to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2026-09-12
Modernized release: a Go-only backend, a new dashboard, and one runnable stack.

### Changed
- Replaced the EOL Rails auth app (Ruby 2.6 / Rails 6.0) with a Go WebAuthn +
  TOTP identity service (`mm-auth`).
- Rebuilt the broker and device agent on `flynn/noise`, off the unmaintained
  `gopkg.in/noisesocket.v0`. The agent↔broker shell channel is now
  Noise-encrypted (it was plaintext in the prototype).
- Rebuilt the dashboard as a lean React 18 + Vite app (light violet theme),
  replacing the 2019 Create-React-App / Argon template.

### Added
- One Go module: `cmd/{mm-broker,mm-agent,mm-auth}` + `internal/*`.
- Persistent device identity (Noise IK) and single-use token enrollment.
- A functional Docker Compose demo (web + broker + agent + auth) and an
  end-to-end test (agent registers → browser session → real PTY shell).
- Governance baseline: CI (Go 1.26 + web build + compose e2e), CodeQL,
  Dependabot, SECURITY policy, and this changelog.

## [0.1.0] - 2026-09-12
Imported legacy baseline (Rails auth + CRA/Argon web + GOPATH Go broker/agent),
preserved under `_legacy/`.

[1.0.0]: https://github.com/trustsentinel/marshmallows/releases/tag/v1.0.0
[0.1.0]: https://github.com/trustsentinel/marshmallows/releases/tag/v0.1.0
