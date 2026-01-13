# CLI reference (`gladiator`)

The repository builds a single CLI binary (`gladiator`) from `main.go`. It uses `urfave/cli/v3` and exposes a small set of subcommands.

## Build

```bash
go build ./...
```

## Commands

### `serve`

Starts **console + backend** in one process.

Key flags (defaults from `internal/app/action/action_defaults.go`):

- `--console-addr` (default `127.0.0.1:2137`)
- `--console-public-addr` (default `http://127.0.0.1:2137`)
- `--backend-addr` (default `127.0.0.1:6112`)
- `--proxy` (default `lan`; supported: `lan`, `webrtc-beta`, `relay-beta`)
- `--lan-my-ip-addr` (default `127.0.0.1`)
- `--relay-addr` / `--relay-public-addr` (used in `relay-beta`)
- `--lobby-addr` (default `ws://127.0.0.1:2137/lobby`)
- `--database-type` (`memory` or `sqlite`)
- `--sqlite-path` (default `dispel-multi.sqlite`)

### `console`

Starts **console** only.

Notable flags:

- `--console-addr`, `--console-public-addr`
- `--relay-addr`, `--relay-public-addr` (if set, console enters `relay-beta` run mode)
- `--database-type`, `--sqlite-path`

### `backend`

Starts **backend** only and points it at an existing console:

- `--console-addr` (**URL** with `http://` or `https://`)
- `--backend-addr`
- `--proxy`, `--lan-my-ip-addr`, `--relay-addr`
- `--lobby-addr` (websocket URL)

The backend fetches `/.well-known/console.json` from the console and will error if `--proxy` run mode doesn’t match the console’s advertised run mode.

### `turn`

Starts a standalone TURN server (used for WebRTC mode).

- `--turn-public-ip` (default `127.0.0.1`)
- `--turn-port` (default `3478`)
- `--turn-realm` (default `dispel-multi`)

### `gui` (optional)

Built only with the `gui` build tag:

```bash
go run -tags gui ./ gui
```

This uses Fyne (`fyne.io/fyne/v2`) and is currently a lightweight UI wrapper around internal controller screens.

## Global flags

Global flags apply to all commands:

- `--log-level` (`debug|info|warn|error`)
- `--log-format` (`text|json|discard`)
- `--log-file`
- `--no-color`

