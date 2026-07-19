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
	"github.com/dimspell/gladiator/internal/backend/proxy/transport"
	"github.com/quic-go/quic-go"
)

// RelayStream abstracts a QUIC stream for reading and writing relay packets.
type RelayStream interface {
	io.Reader
	io.Writer
	CancelRead(code quic.StreamErrorCode)
	CancelWrite(code quic.StreamErrorCode)
	Close() error
}

// deadlineStream wraps a RelayStream to apply a read deadline before each Read,
// preventing a silent remote peer from blocking the scanner goroutine forever.
type deadlineStream struct {
	stream  RelayStream
	timeout time.Duration
}

func (d *deadlineStream) Read(b []byte) (int, error) {
	// If the underlying stream supports SetReadDeadline, use it.
	if s, ok := d.stream.(interface{ SetReadDeadline(time.Time) error }); ok {
		_ = s.SetReadDeadline(time.Now().Add(d.timeout))
	}
	return d.stream.Read(b)
}

// RelayTransport is the QUIC/relay implementation of transport.PeerTransport. It owns the
// dial, the single bidirectional stream, and the framed read/write loop, and
// translates transport.RelayPacket <-> transport.TransportPacket.
type RelayTransport struct {
	addr   string
	selfID string

	mu     sync.Mutex
	conn   *quic.Conn
	stream RelayStream

	recvCh chan transport.TransportPacket
}

var _ transport.PeerTransport = (*RelayTransport)(nil)

func NewRelayTransport(addr, selfID string) *RelayTransport {
	return &RelayTransport{
		addr:   addr,
		selfID: selfID,
		recvCh: make(chan transport.TransportPacket, 64),
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
	t.recvCh = make(chan transport.TransportPacket, 64)
	t.mu.Unlock()

	if err := t.write(transport.RelayPacket{Type: "join", RoomID: roomID, FromID: t.selfID}); err != nil {
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
		var rp transport.RelayPacket
		if err := json.Unmarshal(data, &rp); err != nil {
			continue
		}
		var kind transport.PacketKind
		switch rp.Type {
		case "join":
			kind = transport.KindJoin
		case "tcp":
			kind = transport.KindTCP
		case "udp":
			kind = transport.KindUDP
		case "leave":
			kind = transport.KindLeave
		case "ping":
			kind = transport.KindPing
		default:
			continue
		}
		t.recvCh <- transport.TransportPacket{
			FromID: rp.FromID,
			ToID:   rp.ToID,
			RoomID: rp.RoomID,
			Kind:   kind,
			Data:   rp.Payload,
		}
	}
}

func (t *RelayTransport) Recv(ctx context.Context) (transport.TransportPacket, error) {
	select {
	case <-ctx.Done():
		return transport.TransportPacket{}, ctx.Err()
	case pkt, ok := <-t.recvCh:
		if !ok {
			return transport.TransportPacket{}, io.EOF
		}
		return pkt, nil
	}
}

func (t *RelayTransport) Send(ctx context.Context, pkt transport.TransportPacket) error {
	rp := transport.RelayPacket{RoomID: pkt.RoomID, ToID: pkt.ToID, FromID: t.selfID}
	switch pkt.Kind {
	case transport.KindTCP:
		rp.Type = "tcp"
	case transport.KindUDP:
		rp.Type = "udp"
	case transport.KindPing:
		rp.Type = "ping"
	default:
		return fmt.Errorf("unsupported send kind %v", pkt.Kind)
	}
	rp.Payload = pkt.Data
	return t.write(rp)
}

func (t *RelayTransport) write(rp transport.RelayPacket) error {
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
	return t.write(transport.RelayPacket{Type: "leave", FromID: t.selfID})
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
