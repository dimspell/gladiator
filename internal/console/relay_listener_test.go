package console

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dimspell/gladiator/internal/backend/proxy/relay/types"
	"github.com/quic-go/quic-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- In-memory relay transport (test double) -------------------------------

// memStream is a bidirectional in-memory stream satisfying RelayStream. It is
// backed by two io.Pipe pairs (one direction each).
type memStream struct {
	reader *io.PipeReader
	writer *io.PipeWriter
}

func newMemStreamPair() (server, client *memStream) {
	cr, cw := io.Pipe() // client writes -> server reads
	sr, sw := io.Pipe() // server writes -> client reads
	server = &memStream{reader: cr, writer: sw}
	client = &memStream{reader: sr, writer: cw}
	return
}

func (s *memStream) Read(p []byte) (int, error)  { return s.reader.Read(p) }
func (s *memStream) Write(p []byte) (int, error) { return s.writer.Write(p) }
func (s *memStream) CancelRead(code quic.StreamErrorCode) {
	_ = s.reader.Close()
}
func (s *memStream) CancelWrite(code quic.StreamErrorCode) {
	_ = s.writer.CloseWithError(io.ErrClosedPipe)
}
func (s *memStream) Close() error {
	_ = s.reader.Close()
	_ = s.writer.Close()
	return nil
}

// memConn is a single in-memory relay connection offered to the server via
// AcceptStream (exactly one stream, mirroring the real QUIC model).
type memConn struct {
	streamCh chan *memStream
	remote   net.Addr
	closed   atomic.Bool
}

func (c *memConn) AcceptStream(ctx context.Context) (RelayStream, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case s, ok := <-c.streamCh:
		if !ok {
			return nil, io.EOF
		}
		return s, nil
	}
}
func (c *memConn) CloseWithError(code quic.ApplicationErrorCode, msg string) error {
	if c.closed.CompareAndSwap(false, true) {
		close(c.streamCh)
	}
	return nil
}
func (c *memConn) RemoteAddr() net.Addr { return c.remote }

// InMemoryRelayListener is a RelayListener that needs no UDP socket. Tests call
// Connect to open a client stream; the server observes the connection via
// Accept.
type InMemoryRelayListener struct {
	connCh chan *memConn
	addr   net.Addr
	closed atomic.Bool
}

func NewInMemoryRelayListener() *InMemoryRelayListener {
	return &InMemoryRelayListener{
		connCh: make(chan *memConn, 16),
		addr:   &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 9999},
	}
}

func (l *InMemoryRelayListener) Accept(ctx context.Context) (RelayConn, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case c, ok := <-l.connCh:
		if !ok {
			return nil, io.EOF
		}
		return c, nil
	}
}
func (l *InMemoryRelayListener) Addr() net.Addr { return l.addr }
func (l *InMemoryRelayListener) Close() error {
	if l.closed.CompareAndSwap(false, true) {
		close(l.connCh)
	}
	return nil
}

// Connect opens a client connection to the listener and returns the
// client-side stream. The server will accept the connection and read the
// paired server-side stream.
func (l *InMemoryRelayListener) Connect() (*memStream, error) {
	serverConn := &memConn{
		streamCh: make(chan *memStream, 1),
		remote:   &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 12345},
	}
	clientStream, serverStream := newMemStreamPair()
	serverConn.streamCh <- serverStream
	l.connCh <- serverConn
	return clientStream, nil
}

// --- Test helpers ----------------------------------------------------------

type mockSessionProvider struct {
	allowed map[int64]bool
}

func (m *mockSessionProvider) GetUserSession(id int64) (*UserSession, bool) {
	if m == nil || m.allowed == nil {
		return nil, false
	}
	if m.allowed[id] {
		return &UserSession{}, true
	}
	return nil, false
}

func writePacket(s *memStream, pkt RelayPacket) error {
	data, err := json.Marshal(pkt)
	if err != nil {
		return err
	}
	return types.WriteFramed(s, data)
}

// readPackets streams every RelayPacket read from s until the stream closes or
// ctx is cancelled.
func readPackets(ctx context.Context, s *memStream) <-chan RelayPacket {
	out := make(chan RelayPacket, 32)
	go func() {
		defer close(out)
		for {
			raw, err := types.ReadFramed(s)
			if err != nil {
				return
			}
			var pkt RelayPacket
			if err := json.Unmarshal(raw, &pkt); err != nil {
				return
			}
			select {
			case out <- pkt:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out
}

func newTestServer(t *testing.T, allowed map[int64]bool) (*RelayServer, *InMemoryRelayListener, context.CancelFunc) {
	t.Helper()
	listener := NewInMemoryRelayListener()
	rs, err := NewQUICRelay("127.0.0.1:9999", &mockSessionProvider{allowed: allowed},
		WithListener(listener),
		WithVerifyFunc(func(b []byte) ([]byte, bool) { return b, true }),
	)
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	go rs.Start(ctx)
	return rs, listener, cancel
}

// --- Tests -----------------------------------------------------------------

func TestRelayServer_JoinBroadcastsToOtherPeers(t *testing.T) {
	_, listener, cancel := newTestServer(t, map[int64]bool{1: true, 2: true})
	defer cancel()

	a, err := listener.Connect()
	require.NoError(t, err)
	defer a.Close()
	readerA := readPackets(context.Background(), a)
	require.NoError(t, writePacket(a, RelayPacket{Type: "join", RoomID: "R", FromID: "1"}))

	b, err := listener.Connect()
	require.NoError(t, err)
	defer b.Close()
	require.NoError(t, writePacket(b, RelayPacket{Type: "join", RoomID: "R", FromID: "2"}))

	select {
	case pkt := <-readerA:
		assert.Equal(t, "join", pkt.Type)
		assert.Equal(t, "2", pkt.FromID)
		assert.Equal(t, "1", pkt.ToID)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for join broadcast to peer A")
	}
}

func TestRelayServer_RejectsUnknownUser(t *testing.T) {
	_, listener, cancel := newTestServer(t, map[int64]bool{1: true})
	defer cancel()

	c, err := listener.Connect()
	require.NoError(t, err)
	defer c.Close()
	require.NoError(t, writePacket(c, RelayPacket{Type: "join", RoomID: "R", FromID: "999"}))

	_, err = types.ReadFramed(c)
	assert.Error(t, err, "handshake for an unknown user must close the stream")
}

func TestRelayServer_LeaveCleansUpPeer(t *testing.T) {
	rs, listener, cancel := newTestServer(t, map[int64]bool{1: true, 2: true})
	defer cancel()

	a, err := listener.Connect()
	require.NoError(t, err)
	defer a.Close()
	require.NoError(t, writePacket(a, RelayPacket{Type: "join", RoomID: "R", FromID: "1"}))

	require.Eventually(t, func() bool {
		rs.mu.Lock()
		defer rs.mu.Unlock()
		room, ok := rs.rooms["R"]
		return ok && len(room.Peers) == 1
	}, 2*time.Second, 10*time.Millisecond)

	require.NoError(t, writePacket(a, RelayPacket{Type: "leave", RoomID: "R", FromID: "1"}))

	require.Eventually(t, func() bool {
		rs.mu.Lock()
		defer rs.mu.Unlock()
		_, ok := rs.peerToRoomIDs["1"]
		return !ok
	}, 2*time.Second, 10*time.Millisecond)

	// A later joiner must not resurrect the departed peer.
	b, err := listener.Connect()
	require.NoError(t, err)
	defer b.Close()
	require.NoError(t, writePacket(b, RelayPacket{Type: "join", RoomID: "R", FromID: "2"}))

	require.Eventually(t, func() bool {
		rs.mu.Lock()
		defer rs.mu.Unlock()
		if _, exists := rs.peerToRoomIDs["1"]; exists {
			return false
		}
		room, ok := rs.rooms["R"]
		return ok && len(room.Peers) == 1
	}, 2*time.Second, 10*time.Millisecond)
}

func TestRelayServer_ConcurrentWritesToSamePeer(t *testing.T) {
	_, listener, cancel := newTestServer(t, map[int64]bool{1: true, 2: true, 3: true})
	defer cancel()

	a, err := listener.Connect()
	require.NoError(t, err)
	defer a.Close()
	readerA := readPackets(context.Background(), a)
	require.NoError(t, writePacket(a, RelayPacket{Type: "join", RoomID: "R", FromID: "1"}))

	b, err := listener.Connect()
	require.NoError(t, err)
	defer b.Close()
	_ = readPackets(context.Background(), b) // peers get join notifications; must be drained
	require.NoError(t, writePacket(b, RelayPacket{Type: "join", RoomID: "R", FromID: "2"}))

	c, err := listener.Connect()
	require.NoError(t, err)
	defer c.Close()
	_ = readPackets(context.Background(), c)
	require.NoError(t, writePacket(c, RelayPacket{Type: "join", RoomID: "R", FromID: "3"}))

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		_ = writePacket(b, RelayPacket{Type: "tcp", RoomID: "R", FromID: "2", ToID: "1", Payload: []byte("fromB")})
	}()
	go func() {
		defer wg.Done()
		_ = writePacket(c, RelayPacket{Type: "tcp", RoomID: "R", FromID: "3", ToID: "1", Payload: []byte("fromC")})
	}()
	wg.Wait()

	received := map[string]bool{}
	timeout := time.After(2 * time.Second)
	for len(received) < 2 {
		select {
		case pkt := <-readerA:
			if pkt.Type == "tcp" && pkt.ToID == "1" {
				received[pkt.FromID] = true
			}
		case <-timeout:
			t.Fatal("timed out waiting for tcp packets to peer A")
		}
	}
	assert.True(t, received["2"], "expected tcp from peer 2")
	assert.True(t, received["3"], "expected tcp from peer 3")
}
