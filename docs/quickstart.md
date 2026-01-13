# Quick start (local dev)

This is a **work-in-progress** project. The instructions below reflect defaults currently hardcoded in the CLI flags and packages.

## Prerequisites

- **Go**: see `go.mod` (`go 1.24.x`)
- Optional tools (only if you work on proto/db codegen):
  - `buf` (protobuf generation)
  - `sqlc` (SQL -> Go)
  - `migrate` (DB migrations)

## Start everything (console + backend)

From repo root:

```bash
make serve
```

This runs (see `Makefile`) the `serve` command with default-ish addresses:

- **console**: `127.0.0.1:2137`
- **backend**: `127.0.0.1:6112`

## Start console only

```bash
make console
```

## Start backend only (pointing at an existing console)

```bash
make backend
```

## Configure the game to use your backend

After installing **Dispel Colosseum**, update the registry key so the game points at your local backend:

- `HKEY_LOCAL_MACHINE\SOFTWARE\WOW6432Node\AbalonStudio\Dispel\Multi`
- set `Server` to `localhost`

See the root `README.md` for the exact `regedit` snippet.

## Common environment variables

Most CLI flags can also be set via env vars:

- `CONSOLE_ADDR`, `CONSOLE_BIND`, `CONSOLE_PUBLIC_ADDR`
- `BACKEND_ADDR`
- `PROXY` (one of `lan`, `webrtc-beta`, `relay-beta`)
- `LAN_MY_IP_ADDR`
- `RELAY_ADDR`, `RELAY_BIND`, `RELAY_PUBLIC_ADDR`
- `DATABASE_TYPE` (`memory` or `sqlite`), `SQLITE_PATH`
- Logging: `LOG_LEVEL`, `LOG_FORMAT`, `LOG_FILE`, `NO_COLOR`

## Troubleshooting

Troubleshooting notes live in the root `README.md`:

- **Windows**: HNS restart may fix “forbidden by access permissions” socket errors.
- **Linux/macOS**: you may need to alias `127.0.0.X` on loopback for relay testing.
- **Linux**: QUIC UDP buffer size warnings can require `sysctl` changes.

