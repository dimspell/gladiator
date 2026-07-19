package p2p

import (
	"context"
	"io"
	"log/slog"
	"sync"

	"github.com/dimspell/gladiator/internal/backend/proxy/transport"
)

// webrtcTransport implements transport.PeerTransport over a mesh of WebRTC peer
// connections. It is session-scoped (one per PeerToPeer instance) and multiplexes
// every peer through a single Send/Recv surface keyed by peer ID — exactly like
// RelayTransport multiplexes peers through the relay server.
//
// Outbound: Send looks up the destination peer by pkt.ToID and writes the payload
// onto its data channel, prefixing it with 'T'/'U' so the receiver can route it to
// the right fake socket (mirroring the legacy p2p wire framing).
//
// Inbound: setupDataChannel registers dc.OnMessage handlers that push a
// transport.TransportPacket (tagged with the sender's peer ID) into recvCh. The
// PacketRouter receive loop drains recvCh and writes each packet to the matching
// FakeHost's ProxyTCP/ProxyUDP.
type webrtcTransport struct {
	logger *slog.Logger

	// lookup returns the live peer for a peer ID. It reads PeerToPeer.peers under
	// its own lock, so no additional synchronization is needed here.
	lookup func(peerID string) (*Peer, bool)

	mu sync.Mutex
	// recvCh aggregates inbound data-channel messages from every peer. It is
	// (re)created on Join and closed on Close so reconnection is leak-free.
	recvCh chan transport.TransportPacket
	closed bool
}

var _ transport.PeerTransport = (*webrtcTransport)(nil)

// Join is a no-op for WebRTC: peers connect via signaling handled in
// PeerToPeer.Handle, not through a central "join". It (re)creates the receive
// channel so a stale receive loop exits and a fresh one can attach.
func (t *webrtcTransport) Join(ctx context.Context, roomID string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.recvCh != nil && !t.closed {
		close(t.recvCh)
		t.recvCh = nil // prevent deliver from sending to the now-closed channel
	}
	t.recvCh = make(chan transport.TransportPacket, 256)
	t.closed = false
	return nil
}

// Send delivers a packet to the peer identified by pkt.ToID over its data channel.
// Join/Leave/Ping are signaling-level concerns handled outside the data channel,
// so they are dropped here (the transport only carries game TCP/UDP payloads).
func (t *webrtcTransport) Send(ctx context.Context, pkt transport.TransportPacket) error {
	peer, ok := t.lookup(pkt.ToID)
	if !ok {
		t.logger.Debug("webrtc transport: dropping outbound packet; no peer", "to", pkt.ToID)
		return nil
	}

	var prefix byte
	switch pkt.Kind {
	case transport.KindTCP:
		prefix = 'T'
	case transport.KindUDP:
		prefix = 'U'
	default:
		return nil
	}

	payload := make([]byte, len(pkt.Data)+1)
	payload[0] = prefix
	copy(payload[1:], pkt.Data)
	return peer.Send(payload)
}

// Recv blocks until an inbound packet arrives, ctx is done, or the channel is closed.
func (t *webrtcTransport) Recv(ctx context.Context) (transport.TransportPacket, error) {
	t.mu.Lock()
	ch := t.recvCh
	t.mu.Unlock()

	if ch == nil {
		return transport.TransportPacket{}, io.EOF
	}

	select {
	case <-ctx.Done():
		return transport.TransportPacket{}, ctx.Err()
	case pkt, ok := <-ch:
		if !ok {
			return transport.TransportPacket{}, io.EOF
		}
		return pkt, nil
	}
}

// Leave is a no-op: peers leave via signaling / connection-state changes, not a
// transport call.
func (t *webrtcTransport) Leave(ctx context.Context) error { return nil }

// Close tears down the receive channel, unblocking any running receive loop.
func (t *webrtcTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return nil
	}
	t.closed = true
	if t.recvCh != nil {
		close(t.recvCh)
		t.recvCh = nil
	}
	return nil
}

// deliver pushes an inbound data-channel message into the receive channel so the
// PacketRouter dispatch loop can route it to the right FakeHost. It is called
// from dc.OnMessage handlers installed in setupDataChannel.
func (t *webrtcTransport) deliver(fromID string, kind transport.PacketKind, data []byte) {
	t.mu.Lock()
	ch := t.recvCh
	t.mu.Unlock()

	if ch == nil {
		return
	}

	select {
	case ch <- transport.TransportPacket{FromID: fromID, Kind: kind, Data: data}:
	default:
		// No receiver draining (e.g. before Connect started the loop). Drop rather
		// than block the pion callback goroutine.
		t.logger.Warn("webrtc transport: dropping inbound packet; recv channel full", "from", fromID)
	}
}
