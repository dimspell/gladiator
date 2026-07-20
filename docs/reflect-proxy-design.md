# Reflect-Proxy Design Contract (Frozen)

> **Authoritative contract** for `PLAN_reflect_proxy_reimplementation`.
> Every implementation task (2–9) builds against this. It is grounded in the
> **actual current code** (read during Task 1), which diverges from the older
> prose docs in `docs/protocol-reference.md` — see §9 "Current-state corrections".

## 1. Goal in one paragraph

Make the legacy Dispel game (written before private networks, assumes one LAN/NAT)
play over the public internet via a composable **reflect proxy**: one host per room
(≤ 4 players) exposes TCP + UDP; guests connect to the host's TCP and expose
their own UDP, reaching other guests via UDP; TCP is the `##nickname` auth +
keepalive control plane; UDP carries RPG movement/actions; the host periodically
pings the server (liveness) and, on leave **or** missing ping, the server
**migrates the host** to the next peer; every *other* player is bound to a
`127.0.0.X` loopback "fake host" so the game client only ever talks to
loopback and the proxy tunnels the real traffic (no VPN/custom network).

## 2. Package map (verified)

| Package | Responsibility |
|---------|-----------------|
| `internal/backend/redirect` | `Redirect` interface, `HostManager`, `FakeHost`, `ListenerTCP`/`UDP`, `DialerTCP`/`UDP`, `ProxyFactory`. The loopback fake-host layer. |
| `internal/backend/proxy` | `Proxy` interface, `ProxyFactory`, `ProxyClient`, `CreateParams` (selection by run mode). |
| `internal/backend/proxy/relay` | `ProxyRelay` (config), `Relay` (per-session client), `transport.go` (`RelayTransport` — QUIC dial + framed loop), `relay/types` (`RelayPacket`, `ReadFramed`/`WriteFramed`), `in_memory_transport.go` (test double). |
| `internal/backend/proxy/transport` | `PacketRouter` (session routing: join/leave/tcp/udp/ping dispatch, `StartHost`/`StartGuest`), `ProxyClient` impls. **Note: `PacketRouter` is HERE, not in `relay/`.** |
| `internal/console` | `relay.go` (`NewRelayService`/`NewQUICRelay` wiring), `relay_server.go` (`NewQUICRelay` QUIC listener + `relayLoop`), `room.go` (`RegisterRelayHooks`, `HandleRelayJoin/Leave/Delete`), `session_state.go` (`UserSession` FSM), `utilities.go` (`sign`/`verifyRelayPacket` NO-OPS + hardcoded `hmacKey`). |
| `internal/model` | `well_known.go` — `RunMode` (`single`, `lan`, `relay-beta`, `webrtc-beta`). |
| `internal/app/action` | `action_helpers.go` — `selectProxy`, `selectConsoleOptions`, `isValidRunMode`; `backend.go`/`serve.go` — CLI flags (`--proxy`, `--relay-addr`, `--run-mode`, …). |
| `cmd/integration-client` | Mock game client (TCP `:6112` wire + UDP/TCP game packets); extended for 4-player + migration scenarios. |
| `internal/integration` | `testcontainers-go` harness (gated `integration` tag). |

## 3. `Redirect` interface (transport-agnostic)

```go
type Redirect interface {
    Run(ctx context.Context) error
    Alive(now time.Time, timeout time.Duration) bool
    io.Writer
    io.Closer
}
```

- No relay/WebRTC specifics may leak into `redirect/`. The relay transport
  (`proxy/relay`) is **one consumer** that plugs into it.
- Concrete implementations: `ListenerTCP`/`ListenerUDP` (bind `assignedIP`),
  `DialerTCP`/`DialerUDP` (dial `127.0.0.1`). `Noop` for the
  `OtherUserIsJoining` no-op case.

## 4. `HostManager` / `FakeHost` lifecycle

- `AssignIP(remoteID) (string, error)`: sequential `127.0.0.2`–`127.0.0.254`
  (first-free-slot); `IPPrefix` configurable (default `127.0.0.1`). Idempotent
  (returns existing IP). Reverse map `IPToPeerID` prevents reuse while assigned.
- `StartHost(ctx, peerID, assignedIP, tcpPort, udpPort, onRecvTCP, onRecvUDP, onHostDisconnect)`
  → `ListenerTCP`+`ListenerUDP` on `assignedIP:6114/6113`.
- `StartGuest(ctx, peerID, assignedIP, tcpPort, udpPort, onRecvTCP, onRecvUDP, onHostDisconnect)`
  → `DialerTCP`+`DialerUDP` to `127.0.0.1:6114/6113`.
- **Host guard (contract, fixes §8.1 pain point 5):** a peer that is
  **not** the current room host MUST NOT create `StartGuest` dialers for other
  peers. Implemented authoritatively inside `HostManager.StartGuest` via an
  `IsHost func() bool` gate: when set and it returns false, `StartGuest`
  returns `redirect.ErrNotHost` and registers no fake host. The caller
  (`PacketRouter.DynamicJoin`) ALSO fast-paths on `selfID == currentHostID`
  (avoids an unnecessary `AssignIP`), but the redirect-layer guard is the
  authoritative defense so no caller can bypass it. `relay.NewRelay` wires
  `manager.IsHost = func() bool { return router.SelfID() == router.CurrentHostID() }`.
- `Alive(now, timeout)` is reused by host-liveness (Task 5): a host that
  stops answering is detected via the relay heartbeat, not TCP only.
- Leak safety: `CreateFakeHost` runs proxy `Run` under an `errgroup` with
  `ctx`; on `g.Wait()` it `cancel()`s and `StopHost` cleans maps under the lock.

## 5. Unified relay wire protocol (already length-prefixed)

- Transport: QUIC (`quic-go`), self-signed TLS, ALPN `"game-relay"`,
  `MaxIdleTimeout: 30s`, `KeepAlivePeriod: 15s`. One bidirectional stream
  per peer per room.
- **Framing (verified current):** `relay/types.ReadFramed` /
  `WriteFramed` → **4-byte little-endian length + JSON payload**. This
  replaces the older newline-delimited JSON. (The §8.3 "Fix 4" / `bufio.Scanner`
  64 KB concern is already addressed by this framing — Task 3 confirms/keeps it.)
- `RelayPacket` shape (`relay/types`):
  ```go
  type RelayPacket struct {
      Type    string          `json:"type"` // join | leave | tcp | udp | ping
      RoomID string          `json:"room"`
      FromID string          `json:"from"`
      ToID   string          `json:"to"`
      Payload json.RawMessage `json:"payload"`
  }
  ```
- Client side (`RelayTransport` in `proxy/relay/transport.go`):
  `Join(room)` → QUIC dial + open stream + send `join` + `readLoop`
  (framed) → `Recv(ctx)` (channel) / `Send(ctx, pkt)` (tcp/udp/ping) /
  `Leave()` / `Close()`. `write` is mutex-guarded.
- Server side (`relay_server.go`): `handleConn` → `relayLoop` reads framed
  packets, dispatches `join`/`leave`/`tcp`/`udp`/`ping`, routes via
  `RoomService` hooks.

## 6. Host liveness + migration (control plane)

- The host emits a periodic `ping` `RelayPacket` (or uses the `Alive` heartbeat)
  while hosting. The **server** records `lastSeen` per room host.
- On **missing ping beyond timeout** OR **explicit `leave`** for the current
  host, the server:
  1. selects the next host (existing `HostMigration` selection — next in
     join/roster order),
  2. emits the WS `HostMigration` event (val 12) **and** delivers the TCP
     `0x47` (`HostMigration` opcode) frame to each surviving game client,
     carrying the new host's fake-host address,
  3. promotes the new host (it creates `StartHost` listeners + `StartGuest`
     dialers for remaining peers via the Task 2/3 plumbing).
- **Stale-player race guard:** a leaving/migrating peer's teardown MUST NOT
  reset survivors' sessions (the `onHostDisconnected(forced)` path must only
  stop that peer's proxies, never a full room reset). Survivors keep exchanging.
- Heartbeat interval + timeout are configurable (flag/constant, documented).

## 7. Run-mode matrix (active vs disabled)

| Mode | Status | Notes |
|------|--------|-------|
| `single` | active | local loopback, no net |
| `lan` | active | `direct.ProxyLAN` — real IP-to-IP TCP/UDP |
| `relay-beta` | active | `relay.ProxyRelay` → QUIC relay (the reflect proxy) |
| `webrtc-beta` | **disabled (commented, not deleted)** | `p2p.ProxyP2P` retained for reference |
| `libp2p-beta` | **disabled (commented, not deleted)** | event-type only, never wired |

- `selectProxy` must return only `direct` (lan) or `relay`; `webrtc-beta`
  selection is commented out / build-tagged, source kept.
- WebRTC ICE/TURN hardcodes (`turn:127.0.0.1:3478`) neutralized.
- The WebRTC integration test is **commented out, not removed**.

## 8. Security model

- Relay frames authenticated via **HMAC-SHA256**. **Key source: the
  `REFLECT_RELAY_HMAC_KEY` environment variable** — never hardcoded, never
  logged, never committed.
  - *Current state (to be replaced in Task 7):* `internal/console/utilities.go`
    has `var hmacKey = []byte("shared-secret-key")` (a hardcoded placeholder)
    and `sign()` / `verifyRelayPacket()` are **NO-OPS** that ignore it. Task 7
    deletes the hardcoded key, reads `REFLECT_RELAY_HMAC_KEY`, and activates
    the HMAC (fail closed if the env var is absent in non-dev mode).
- Boundary validation at the relay server: validate/normalize `room`/`from`/`to`
  identifiers; reject unexpected `type`; bound `payload` size.
- Strict input validation at every external boundary (TCP `:6112`, WS lobby,
  QUIC relay frames). Treat all external input as untrusted.
- No secrets in logs; `slog` with actionable fields only.

## 9. Current-state corrections (read before implementing)

The older `docs/protocol-reference.md` / `docs/Relay Mode Topology Summary.md`
were written against an earlier layout. Confirmed corrections:

1. **`PacketRouter` is in `internal/backend/proxy/transport`** (imported as
   `tport` from `relay.go`), **not** `internal/backend/proxy/relay/packet_router.go`.
2. **Framing is already length-prefixed** via `relay/types.ReadFramed` /
   `WriteFramed` in `proxy/relay/transport.go` — not `bufio.Scanner` /
   newline JSON. The §8.3 "Fix 4" is effectively done; Task 3 must
   preserve it and add the TCP ident+payload coalescing fix + host guard.
3. **`utilities.go` hardcodes `hmacKey`** — Task 7 removes it in favor of
   the env var.
4. The relay server currently starts **inside the Console process**
   (`NewRelayService` in `relay.go`, called from `console.go` when
   `RunMode == relay-beta`). Task 4 extracts it to a standalone component
   that still wires `RegisterRelayHooks(RoomService)`.
5. `docs/protocol-reference.md` §6 still lists WebRTC/libp2p as active; Task 9
   updates it to reflect the disabled status.

## 10. Validation contract

- Per implementation task: `go build ./...` + `go test -race ./...` (untagged
  green) + `go vet ./...` + `golangci-lint run ./...` on touched packages.
- The **standalone relay** (Task 4) gets its own test: start it, connect a
  `RelayTransport` (or minimal QUIC client), assert `join` propagates and
  `leave` tears down the room.
- Integration (Task 8, `integration` build tag, Docker): the 4-player
  relay full lifecycle from `docs/integration-test-scenarios.md`.
- Security (Task 7): `gosec` (if available) + manual grep that
  `REFLECT_RELAY_HMAC_KEY` is read, never logged/committed.
