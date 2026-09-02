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
// backed by two io.Pipe pairs (one direction each). Writes are non-blocking
// via a buffered channel so the relay's broadcast goroutine never stalls on a
// test reader that has not been scheduled yet.
type memStream struct {
	reader   *io.PipeReader
	writer   *io.PipeWriter
	writeCh  chan []byte
	writeWG  sync.WaitGroup
	closed   atomic.Bool
}

func newMemStreamPair() (server, client *memStream) {
	cr, cw := io.Pipe() // client writes -> server reads
	sr, sw := io.Pipe() // server writes -> client reads
	server = newMemStream(cr, sw)
	client = newMemStream(sr, cw)
	return
}

func newMemStream(reader *io.PipeReader, writer *io.PipeWriter) *memStream {
	s := &memStream{
		reader:  reader,
		writer:  writer,
		writeCh: make(chan []byte, 4096),
	}
	s.writeWG.Add(1)
	go func() {
		defer s.writeWG.Done()
		for data := range s.writeCh {
			if _, err := s.writer.Write(data); err != nil {
				return
			}
		}
	}()
	return s
}

func (s *memStream) Read(p []byte) (int, error)  { return s.reader.Read(p) }
func (s *memStream) Write(p []byte) (int, error) {
	if s.closed.Load() {
		return 0, io.ErrClosedPipe
	}
	// Copy because callers reuse their buffers.
	buf := make([]byte, len(p))
	copy(buf, p)
	select {
	case s.writeCh <- buf:
		return len(p), nil
	}
}
func (s *memStream) CancelRead(code quic.StreamErrorCode) {
	_ = s.reader.Close()
}
func (s *memStream) CancelWrite(code quic.StreamErrorCode) {
	_ = s.writer.CloseWithError(io.ErrClosedPipe)
}
func (s *memStream) Close() error {
	if s.closed.CompareAndSwap(false, true) {
		close(s.writeCh)
		_ = s.reader.Close()
		_ = s.writer.Close()
		s.writeWG.Wait()
	}
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
	rs, listener, cancel := newTestServer(t, map[int64]bool{1: true, 2: true})
	defer cancel()

	a, err := listener.Connect()
	require.NoError(t, err)
	defer a.Close()
	readerA := readPackets(context.Background(), a)
	require.NoError(t, writePacket(a, RelayPacket{Type: "join", RoomID: "R", FromID: "1"}))

	// Wait until A is actually in the room. Without this, B's join can
	// happen before A is registered and the broadcast goes nowhere.
	require.Eventually(t, func() bool {
		room, ok := any(rs).(interface {
			PeersInRoom(string) []string
		})
		if !ok {
			return false
		}
		for _, id := range room.PeersInRoom("R") {
			if id == "1" {
				return true
			}
		}
		return false
	}, 2*time.Second, 5*time.Millisecond, "A never joined the room")

	b, err := listener.Connect()
	require.NoError(t, err)
	defer b.Close()
	require.NoError(t, writePacket(b, RelayPacket{Type: "join", RoomID: "R", FromID: "2"}))

	// Wait for the broadcast to be sent to A. We poll the reader on a
	// short tick because a single select+time.After is unreliable when
	// the relay goroutine is preempted under load.
	deadline := time.Now().Add(2 * time.Second)
	tick := time.NewTicker(5 * time.Millisecond)
	defer tick.Stop()
	for time.Now().Before(deadline) {
		select {
		case pkt := <-readerA:
			assert.Equal(t, "join", pkt.Type)
			assert.Equal(t, "2", pkt.FromID)
			assert.Equal(t, "1", pkt.ToID)
			return
		case <-tick.C:
		}
	}
	t.Fatal("timed out waiting for join broadcast to peer A")
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
		room := rs.PeersInRoom("R")
		return len(room) == 1
	}, 2*time.Second, 10*time.Millisecond)

	require.NoError(t, writePacket(a, RelayPacket{Type: "leave", RoomID: "R", FromID: "1"}))

	require.Eventually(t, func() bool {
		return !rs.HasPeer("1")
	}, 2*time.Second, 10*time.Millisecond)

	// A later joiner must not resurrect the departed peer.
	b, err := listener.Connect()
	require.NoError(t, err)
	defer b.Close()
	require.NoError(t, writePacket(b, RelayPacket{Type: "join", RoomID: "R", FromID: "2"}))

	require.Eventually(t, func() bool {
		if rs.HasPeer("1") {
			return false
		}
		room := rs.PeersInRoom("R")
		return len(room) == 1
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
	deadline := time.Now().Add(30 * time.Second)
	for len(received) < 2 && time.Now().Before(deadline) {
		select {
		case pkt := <-readerA:
			if pkt.Type == "tcp" && pkt.ToID == "1" {
				received[pkt.FromID] = true
			}
		case <-time.After(20 * time.Millisecond):
		}
	}
	if len(received) < 2 {
		t.Fatal("timed out waiting for tcp packets to peer A")
	}
	assert.True(t, received["2"], "expected tcp from peer 2")
	assert.True(t, received["3"], "expected tcp from peer 3")
}
