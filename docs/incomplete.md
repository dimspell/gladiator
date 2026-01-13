# Known gaps / incomplete areas

This repository is **not finished**. This page is intentionally blunt about what is currently “dev-grade” or stubbed so users don’t assume production readiness.

## Security / auth

- Console has an `authMiddleware` scaffold commented out in `internal/console/console.go`.
- JWT secret defaults to a hardcoded dev value (`dev-secret-key`).
- WebSocket identity is currently based on query param `userID` + a `Hello` message, not a verified auth token.

## Protocol stability

- WebSocket protocol version is currently `wire.ProtoVersion = "dev"`.
- Run modes include `webrtc-beta` and `relay-beta` strings and should be treated as experimental.

## WebRTC / TURN defaults are not production-ready

- WebRTC ICE config includes a public Google STUN server and a local TURN URL (`turn:127.0.0.1:3478`).
- TURN credentials are embedded in code (see `internal/app/action/turn.go` and proxy config in `internal/app/action/action_helpers.go`).

## Relay mode assumptions

- Relay mode requires loopback aliasing (`127.0.0.X`) on some platforms for local testing; see `README.md` troubleshooting.
- Relay service is only created when console run mode is relay (`WithRelayAddr` sets `RunModeRelay`).

## Launcher / GUI

- The GUI exists behind the `gui` build tag and is currently a thin wrapper around internal controller screens, not a full “installer/launcher” experience.

## Docs coverage

The docs in `docs/` cover:

- Current CLI flags and default addresses
- Current exposed endpoints and protocols

They intentionally do **not** promise feature completeness, compatibility, or stability.

