# marshmallows — local demo

A full, working marshmallows platform on your machine: the **dashboard**, the
**broker**, a live **agent** (a real device shell), and the **identity service**.

## Run it
```bash
docker compose -f deploy/compose/compose.yml up --build
```
Then open **http://localhost:8080**.

You land on the **Overview** (login is bypassed for the demo). You'll see one
live device — **`valc31.pi01` (Greenhouse Pi)** — served by the `agent`
container. Click **Connect** to open a real shell into it: the keystrokes travel
browser → broker → agent over the Noise-brokered channel, and a `/bin/sh` runs
in the agent container. Try `uname -a`, `ls /`, `ps`.

Tear down:
```bash
docker compose -f deploy/compose/compose.yml down -v
```

## What's running
| Service | Port | Role |
|---|---|---|
| `web` | 8080 | React dashboard (nginx) |
| `broker` | 8081 | coordinator: device registry + terminal bridging (Noise to agents) |
| `agent` | — | a device: pins the broker key, enrolls, serves a PTY shell (no inbound port) |
| `auth` | 8090 | identity service (WebAuthn + TOTP) — used for real login outside the demo |

## Notes
- The demo runs the broker in **dev mode** (any enrollment token, open browser
  access) so it works with no setup. In production, set the broker's `-auth-url`
  to mm-auth (enrollment tokens are then single-use and verified) and serve the
  dashboard + broker behind one origin so the session cookie gates terminals.
- The agent pins the broker's public key: the broker writes it to a shared
  volume (`-pubkey-out`) and the agent reads it before connecting (Noise IK).
- Login is bypassed in the demo via `VITE_DEMO_AUTH=1`. Build the web with
  `VITE_DEMO_AUTH=0` to require WebAuthn sign-in against `auth`.
