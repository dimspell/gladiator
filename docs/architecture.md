# Architecture (current code)

This is the **as-implemented** architecture (not a final design doc).

## Components

### Console (`internal/console`)

An HTTP server (h2c/http2) acting as the “control plane”:

- Metadata for clients/backends: `GET /.well-known/console.json`
- Service APIs (ConnectRPC): `/grpc/*`
- WebSocket lobby/signaling: `/lobby`
- Observability: `GET /_health`, `GET /_metrics`

Console also owns:

- Lobby/presence/matchmaking state (`RoomService`)
- Relay integration (`RelayService`) when in `relay-beta` run mode
- Database connection (memory or sqlite)

### Backend (`internal/backend`)

A TCP server that accepts connections from the game’s multiplayer client (`DispelMulti.exe` behavior is being emulated). It:

- Accepts TCP connections (default `127.0.0.1:6112`)
- Performs a handshake and then processes command packets
- Calls console APIs (`/grpc/*`) for user/game/character/ranking operations
- Uses a proxy implementation depending on run mode:
  - `lan`: direct LAN proxy
  - `webrtc-beta`: P2P/WebRTC proxy using console `/lobby` for signaling
  - `relay-beta`: relay proxy (console provides relay info; a relay server may be started by console)

### Launcher / GUI (`internal/app/ui`, `-tags gui`)

Optional Fyne-based app meant to help configure/start/connect the pieces. It’s present but not a complete product yet.

## Typical flows

### Local “all-in-one” (developer mode)

- You run `gladiator serve`
- Console starts (HTTP on `:2137`)
- Backend starts (TCP on `:6112`)
- Game is configured to use `localhost` as the multiplayer server (registry change)

### Backend joining an existing console

- You run `gladiator console` somewhere reachable
- You run `gladiator backend --console-addr=http://...`
- Backend validates it’s using the same run mode as the console (`/.well-known/console.json`)

## Run modes

Run modes are advertised by the console in `/.well-known/console.json` and are currently:

- `lan`
- `webrtc-beta`
- `relay-beta`

The CLI flag `--proxy` controls which proxy backend uses; console enters `relay-beta` if a relay bind address is configured.

