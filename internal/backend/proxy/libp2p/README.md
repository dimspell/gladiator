## Architecture

```
Player A (game client)
    │ TCP/UDP 127.x.x.x
    ▼
redirect.FakeHost     ◄──── libp2p stream ────►  redirect.FakeHost
    │                                                  │
    ▼                                                  ▼
Libp2pProxy                                       Libp2pProxy
    │                                                  │
    └─── libp2p.Host ──── /gladiator/game/1.0.0 ───── libp2p.Host

```

Address exchange via existing WebSocket signalling — when a player starts their libp2p host (on `CreateRoom` or `JoinGame`) it broadcasts all its multiaddresses using the new `Libp2pAddresses` wire event, exactly like WebRTC uses `RTCOffer`/`RTCAnswer`.

Single stream per peer — one bidirectional libp2p stream carries both TCP and UDP frames, prefixed with 'T' or 'U' (same convention as the WebRTC proxy), length-framed with a 4-byte header.

## Usage

```go
proxyFactory := &libp2p.ProxyLibp2p{
    IPPrefix: net.IPv4(127, 0, 0, 0),
}
sessionManager := backend.NewSessionManager(proxyFactory, gameClient)
```

