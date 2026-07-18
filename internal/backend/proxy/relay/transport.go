package relay

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/dimspell/gladiator/internal/backend/proxy/relay/types"
	"github.com/quic-go/quic-go"
)

// PacketKind classifies the payload of a TransportPacket independent of the
// underlying transport (relay/QUIC, WebRTC, or an in-memory test double).
type PacketKind int

const (
	KindJoin PacketKind = iota
	KindTCP
	KindUDP
	KindLeave
	KindPing
)

// TransportPacket is the transport-agnostic unit delivered between two peers.
// Adapters (e.g. RelayTransport) translate their wire format into this shape so
// the PacketRouter dispatch logic never depends on QUIC or on the RelayPacket
// envelope.
type TransportPacket struct {
	FromID string
	ToID   string
	RoomID string
	Kind   PacketKind
	Data   []byte
}

// PeerTransport is the hexagon's outer port on the peer-network side. The
// PacketRouter depends only on this interface, so the relay (QUIC), WebRTC, or
// an in-memory test double can be swapped without touching the dispatch logic.
type PeerTransport interface {
	// Join connects to the relay infrastructure and announces presence in roomID.
	Join(ctx context.Context, roomID string) error
	// Send delivers a packet to the peer identified by pkt.ToID.
	Send(ctx context.Context, pkt TransportPacket) error
	// Recv blocks until a packet arrives or ctx is done/cancelled.
	Recv(ctx context.Context) (TransportPacket, error)
	// Leave notifies the infrastructure this peer is departing.
	Leave(ctx context.Context) error
	// Close tears down the transport.
	Close() error
}

// RelayTransport is the QUIC/relay implementation of PeerTransport. It owns the
// dial, the single bidirectional stream, and the framed read/write loop, and
// translates RelayPacket <-> TransportPacket.
type RelayTransport struct {
	addr   string
	selfID string

	mu     sync.Mutex
	conn   *quic.Conn
	stream RelayStream

	recvCh chan TransportPacket
}

func NewRelayTransport(addr, selfID string) *RelayTransport {
	return &RelayTransport{
		addr:   addr,
		selfID: selfID,
		recvCh: make(chan TransportPacket, 64),
	}
}

func (t *RelayTransport) Join(ctx context.Context, roomID string) error {
	tlsConf := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"game-relay"},
	}
	conn, err := quic.DialAddr(ctx, t.addr, tlsConf, &quic.Config{
		MaxIdleTimeout:  30 * time.Second,
		KeepAlivePeriod: 15 * time.Second,
	})
	if err != nil {
		return fmt.Errorf("quic dial failed: %w", err)
	}
	stream, err := conn.OpenStreamSync(ctx)
	if err != nil {
		_ = conn.CloseWithError(0xDEAD, "failed to open stream")
		return fmt.Errorf("quic open stream failed: %w", err)
	}

	t.mu.Lock()
	t.conn = conn
	t.stream = stream
	t.recvCh = make(chan TransportPacket, 64)
	t.mu.Unlock()

	if err := t.write(RelayPacket{Type: "join", RoomID: roomID, FromID: t.selfID}); err != nil {
		_ = stream.Close()
		_ = conn.CloseWithError(0xDEAD, "send join failed")
		return fmt.Errorf("send join packet failed: %w", err)
	}

	// Make sure QUIC sent the join packet on its own, not coalesced with others.
	time.Sleep(100 * time.Millisecond)

	go t.readLoop(stream)
	return nil
}

func (t *RelayTransport) readLoop(stream RelayStream) {
	defer close(t.recvCh)
	deadlineReader := &deadlineStream{stream: stream, timeout: 30 * time.Second}
	for {
		data, err := types.ReadFramed(deadlineReader)
		if err != nil {
			return
		}
		var rp RelayPacket
		if err := json.Unmarshal(data, &rp); err != nil {
			continue
		}
		var kind PacketKind
		switch rp.Type {
		case "join":
			kind = KindJoin
		case "tcp":
			kind = KindTCP
		case "udp":
			kind = KindUDP
		case "leave":
			kind = KindLeave
		case "ping":
			kind = KindPing
		default:
			continue
		}
		t.recvCh <- TransportPacket{
			FromID: rp.FromID,
			ToID:   rp.ToID,
			RoomID: rp.RoomID,
			Kind:   kind,
			Data:   rp.Payload,
		}
	}
}

func (t *RelayTransport) Recv(ctx context.Context) (TransportPacket, error) {
	select {
	case <-ctx.Done():
		return TransportPacket{}, ctx.Err()
	case pkt, ok := <-t.recvCh:
		if !ok {
			return TransportPacket{}, io.EOF
		}
		return pkt, nil
	}
}

func (t *RelayTransport) Send(ctx context.Context, pkt TransportPacket) error {
	rp := RelayPacket{RoomID: pkt.RoomID, ToID: pkt.ToID, FromID: t.selfID}
	switch pkt.Kind {
	case KindTCP:
		rp.Type = "tcp"
	case KindUDP:
		rp.Type = "udp"
	case KindPing:
		rp.Type = "ping"
	default:
		return fmt.Errorf("unsupported send kind %v", pkt.Kind)
	}
	rp.Payload = pkt.Data
	return t.write(rp)
}

func (t *RelayTransport) write(rp RelayPacket) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.stream == nil {
		return fmt.Errorf("stream is nil")
	}
	data, err := json.Marshal(rp)
	if err != nil {
		return fmt.Errorf("marshal packet failed: %w", err)
	}
	if err := types.WriteFramed(t.stream, data); err != nil {
		return fmt.Errorf("write packet failed: %w", err)
	}
	return nil
}

func (t *RelayTransport) Leave(ctx context.Context) error {
	return t.write(RelayPacket{Type: "leave", FromID: t.selfID})
}

func (t *RelayTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.stream != nil {
		t.stream.CancelRead(0xDEAD)
		t.stream.CancelWrite(0xDEAD)
		_ = t.stream.Close()
	}
	if t.conn != nil {
		_ = t.conn.CloseWithError(0xDEAD, "done")
	}
	return nil
}
