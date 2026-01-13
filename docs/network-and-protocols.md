# Network & protocols

This project exposes multiple network surfaces: TCP (game-facing), HTTP (console), WebSocket (lobby/signaling), and ConnectRPC/gRPC-like APIs.

## Default ports

Defaults come from `internal/app/action/action_defaults.go`:

- **Console HTTP**: `127.0.0.1:2137`
- **Backend TCP**: `127.0.0.1:6112`
- **Relay server** (relay mode): `127.0.0.1:9999`
- **TURN** (for WebRTC): `:3478`

## Console HTTP endpoints

### Health & metrics

- `GET /_health` — checks DB connectivity
- `GET /_metrics` — Prometheus metrics

### Well-known metadata

- `GET /.well-known/console.json`

Returns JSON (`internal/model/well_known.go`) similar to:

- `version`
- `runMode` (`lan`, `webrtc-beta`, `relay-beta`)
- `consoleServerAddr` (field name currently `Addr`)
- `relayServerAddr` (only in relay mode)
- `callerIP` (only in LAN mode; extracted from request remote addr)

### ConnectRPC APIs (`/grpc/*`)

Console mounts Connect handlers under `/grpc/` (see `proto/multi/v1/*.proto`).

Services currently defined:

- `multi.v1.GameService`
  - `GetGame`, `ListGames`, `CreateGame`, `JoinGame`
- `multi.v1.UserService`
  - `CreateUser`, `AuthenticateUser`, `GetUser`
- `multi.v1.CharacterService`
  - `GetCharacter`, `ListCharacters`, `CreateCharacter`, `PutStats`, `PutSpells`, `PutInventoryCharacter`, `DeleteCharacter`
- `multi.v1.RankingService`
  - `GetRanking`

Implementation lives under `internal/console/*` and generated code under `gen/`.

## Lobby WebSocket (`/lobby`)

Console exposes a WebSocket endpoint at:

- `ws://<console-host>:2137/lobby?userID=...&channelName=DISPEL`

### Connection requirements (current)

- Query params:
  - `channelName` must be exactly `DISPEL`
  - `userID` must be a non-zero integer
- Header:
  - `X-Version` must equal `wire.ProtoVersion` (currently `"dev"`)
- WebSocket subprotocol:
  - must negotiate `wire.SupportedRealm` which is `"lobby-" + wire.ProtoVersion` (currently `lobby-dev`)

### Message framing

WebSocket payloads are:

- first byte: `EventType` (`internal/wire/event_types.go`)
- remaining bytes: JSON-encoded `wire.Message` (codec is currently JSON)

Key event types include:

- `Hello`, `Welcome`
- `LobbyUsers`, `JoinLobby`, `JoinedLobby`, `LeaveLobby`
- `Chat`
- `CreateRoom`, `SetRoomReady`, `JoinRoom`, `LeaveRoom`, `HostMigration`
- `RTCOffer`, `RTCAnswer`, `RTCICECandidate` (used for WebRTC signaling)

## WebRTC mode notes

When running with `--proxy=webrtc-beta`, the proxy factory is configured with ICE servers (see `internal/app/action/action_helpers.go`), including:

- STUN: `stun:stun.l.google.com:19302`
- TURN: `turn:127.0.0.1:3478` (with static credentials in code)

For local WebRTC testing you typically run:

- `gladiator turn`
- `gladiator serve --proxy=webrtc-beta`

## Game-facing TCP protocol (backend)

Backend listens on TCP4 (`net.Listen("tcp4", --backend-addr)`) and handles a binary protocol via a set of “command handlers” under `internal/backend/command_*.go`.

This area is actively reverse-engineered; for now, treat it as internal/unstable.

