package relayserver

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dimspell/gladiator/internal/backend/proxy/relay/types"
	"github.com/quic-go/quic-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testStream struct {
	reader *io.PipeReader
	writer *io.PipeWriter
}

func newTestStreamPair() (server, client *testStream) {
	sr, sw := io.Pipe() // server reads from client writes
	cr, cw := io.Pipe() // client reads from server writes
	return &testStream{reader: sr, writer: cw}, &testStream{reader: cr, writer: sw}
}

func (s *testStream) Read(p []byte) (int, error)  { return s.reader.Read(p) }
func (s *testStream) Write(p []byte) (int, error) { return s.writer.Write(p) }
func (s *testStream) CancelRead(code quic.StreamErrorCode) { s.reader.Close() }
func (s *testStream) CancelWrite(code quic.StreamErrorCode) { s.writer.CloseWithError(io.ErrClosedPipe) }
func (s *testStream) Close() error {
	s.reader.Close()
	s.writer.Close()
	return nil
}

type testConn struct {
	streamCh chan *testStream
	closed   atomic.Bool
}

func newTestConn() *testConn {
	return &testConn{streamCh: make(chan *testStream, 1)}
}

func (c *testConn) AcceptStream(ctx context.Context) (RelayStream, error) {
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

func (c *testConn) CloseWithError(code quic.ApplicationErrorCode, msg string) error {
	if c.closed.CompareAndSwap(false, true) {
		close(c.streamCh)
	}
	return nil
}

func (c *testConn) RemoteAddr() net.Addr { return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 9999} }

type testListener struct {
	connCh chan *testConn
	closed atomic.Bool
}

func newTestListener() *testListener {
	return &testListener{connCh: make(chan *testConn, 16)}
}

func (l *testListener) Accept(ctx context.Context) (RelayConn, error) {
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

func (l *testListener) Addr() net.Addr { return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 9999} }

func (l *testListener) Close() error {
	if l.closed.CompareAndSwap(false, true) {
		close(l.connCh)
	}
	return nil
}

func (l *testListener) Connect() *testStream {
	serverConn := newTestConn()
	clientStream, serverStream := newTestStreamPair()
	serverConn.streamCh <- serverStream
	l.connCh <- serverConn
	return clientStream
}

func writeJoin(s *testStream, fromID, roomID string) error {
	pkt := types.RelayPacket{Type: "join", RoomID: roomID, FromID: fromID, ToID: ""}
	data, _ := json.Marshal(pkt)
	return types.WriteFramed(s, data)
}

func writeLeave(s *testStream, fromID, roomID string) error {
	pkt := types.RelayPacket{Type: "leave", RoomID: roomID, FromID: fromID, ToID: ""}
	data, _ := json.Marshal(pkt)
	return types.WriteFramed(s, data)
}

func readPackets(ctx context.Context, s *testStream) <-chan types.RelayPacket {
	out := make(chan types.RelayPacket, 32)
	go func() {
		defer close(out)
		for {
			raw, err := types.ReadFramed(s)
			if err != nil {
				return
			}
			var pkt types.RelayPacket
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

func newTestServer(t *testing.T) (*RelayServer, *testListener, context.CancelFunc) {
	t.Helper()
	listener := newTestListener()
	rs, err := NewRelayServer("127.0.0.1:9999", AllowAllSessions(),
		WithListener(listener),
		WithVerifyFunc(func(b []byte) ([]byte, bool) { return b, true }),
	)
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	go rs.Start(ctx)
	return rs, listener, cancel
}

func newTestServerWithTimeout(t *testing.T, timeout time.Duration) (*RelayServer, *testListener, context.CancelFunc) {
	t.Helper()
	listener := newTestListener()
	rs, err := NewRelayServer("127.0.0.1:9999", AllowAllSessions(),
		WithListener(listener),
		WithVerifyFunc(func(b []byte) ([]byte, bool) { return b, true }),
		WithPingTimeout(timeout),
		WithLivenessInterval(timeout/2),
	)
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	go rs.Start(ctx)
	return rs, listener, cancel
}

func writePing(s *testStream, fromID, roomID string) error {
	pkt := types.RelayPacket{Type: "ping", RoomID: roomID, FromID: fromID}
	data, _ := json.Marshal(pkt)
	return types.WriteFramed(s, data)
}

func TestRelayServer_PingPreventsTimeout(t *testing.T) {
	rs, listener, cancel := newTestServerWithTimeout(t, 200*time.Millisecond)
	defer cancel()

	a := listener.Connect()
	defer a.Close()
	require.NoError(t, writeJoin(a, "1", "R"))

	require.Eventually(t, func() bool {
		return len(rs.PeersInRoom("R")) == 1
	}, 2*time.Second, 10*time.Millisecond)

	// Send pings faster than the timeout to keep the peer alive
	for i := 0; i < 5; i++ {
		time.Sleep(50 * time.Millisecond)
		require.NoError(t, writePing(a, "1", "R"))
	}

	// Peer should still be alive since we kept pinging
	assert.Equal(t, 1, len(rs.PeersInRoom("R")), "peer timed out despite periodic pings")
}

func TestRelayServer_TimeoutRemovesPeer(t *testing.T) {
	leaveCh := make(chan string, 1)
	rs, listener, cancel := newTestServerWithTimeout(t, 100*time.Millisecond)
	defer cancel()

	// Hook into OnLeave so we know when the peer is removed
	rs.OnLeave = func(evType, peerID, roomID string) {
		leaveCh <- peerID
	}

	a := listener.Connect()
	defer a.Close()
	require.NoError(t, writeJoin(a, "1", "R"))

	require.Eventually(t, func() bool {
		return len(rs.PeersInRoom("R")) == 1
	}, 2*time.Second, 10*time.Millisecond)

	// Don't send anything; peer should time out
	select {
	case peerID := <-leaveCh:
		assert.Equal(t, "1", peerID)
	case <-time.After(2 * time.Second):
		t.Fatal("peer was not removed within timeout")
	}

	require.False(t, rs.HasPeer("1"), "peer should be removed after timeout")
}

func TestRelayServer_JoinBroadcast(t *testing.T) {
	rs, listener, cancel := newTestServer(t)
	defer cancel()

	a := listener.Connect()
	defer a.Close()
	require.NoError(t, writeJoin(a, "1", "R"))

	require.Eventually(t, func() bool {
		return len(rs.PeersInRoom("R")) == 1
	}, 2*time.Second, 10*time.Millisecond)

	readerA := readPackets(context.Background(), a)
	b := listener.Connect()
	defer b.Close()
	require.NoError(t, writeJoin(b, "2", "R"))

	select {
	case pkt := <-readerA:
		assert.Equal(t, "join", pkt.Type)
		assert.Equal(t, "2", pkt.FromID)
		assert.Equal(t, "1", pkt.ToID)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for join broadcast")
	}
}

func TestRelayServer_LeaveCleansUpPeer(t *testing.T) {
	rs, listener, cancel := newTestServer(t)
	defer cancel()

	a := listener.Connect()
	defer a.Close()
	require.NoError(t, writeJoin(a, "1", "R"))

	require.Eventually(t, func() bool {
		return len(rs.PeersInRoom("R")) == 1
	}, 2*time.Second, 10*time.Millisecond)

	require.NoError(t, writeLeave(a, "1", "R"))

	require.Eventually(t, func() bool {
		return !rs.HasPeer("1")
	}, 2*time.Second, 10*time.Millisecond)
}

func TestRelayServer_ForwardPacket(t *testing.T) {
	rs, listener, cancel := newTestServer(t)
	defer cancel()

	a := listener.Connect()
	defer a.Close()
	require.NoError(t, writeJoin(a, "1", "R"))
	readerA := readPackets(context.Background(), a)

	b := listener.Connect()
	defer b.Close()
	require.NoError(t, writeJoin(b, "2", "R"))
	readerB := readPackets(context.Background(), b)

	// Drain join notifications
	select {
	case <-readerA:
	case <-time.After(time.Second):
	}
	select {
	case <-readerB:
	case <-time.After(time.Second):
	}

	// Send a packet from A to B
	data, _ := json.Marshal(types.RelayPacket{Type: "udp", RoomID: "R", FromID: "1", ToID: "2", Payload: []byte("hello")})
	require.NoError(t, types.WriteFramed(a, data))

	select {
	case pkt := <-readerB:
		assert.Equal(t, "udp", pkt.Type)
		assert.Equal(t, "hello", string(pkt.Payload))
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for forwarded packet")
	}

	require.Equal(t, 2, len(rs.PeersInRoom("R")))
}
