# Project documentation (incomplete / WIP)

This repository (`gladiator`) is a monorepo for a Dispel Multiplayer replacement stack. The project is **incomplete**; this `docs/` folder documents what exists in the code today and calls out known gaps.

## What to read first

- **Quick start**: `docs/quickstart.md`
- **CLI reference**: `docs/cli.md`
- **Architecture** (how pieces talk): `docs/architecture.md`
- **Network & protocols** (ports, endpoints, websocket events, gRPC services): `docs/network-and-protocols.md`
- **Known gaps / TODOs**: `docs/incomplete.md`

## Project map (high level)

- **Console**: HTTP server exposing:
  - `/.well-known/console.json` (metadata for launcher/backend)
  - `/grpc/*` (ConnectRPC/gRPC-ish APIs used by backend)
  - `/lobby` (WebSocket signaling + presence/lobby control plane)
- **Backend**: TCP server that pretends to be `DispelMulti.exe`’s multiplayer backend; it translates game client commands into calls to console services and/or proxy logic.
- **Launcher (GUI)**: optional Fyne-based GUI build (`-tags gui`) used to configure/drive the stack.

