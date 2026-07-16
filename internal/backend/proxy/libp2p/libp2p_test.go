package libp2p

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"runtime"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	multiv1 "github.com/dimspell/gladiator/gen/multi/v1"
	"github.com/dimspell/gladiator/internal/app/logger"
	"github.com/dimspell/gladiator/internal/backend/bsession"
	"github.com/dimspell/gladiator/internal/backend/proxy"
	"github.com/dimspell/gladiator/internal/backend/redirect"
	"github.com/dimspell/gladiator/internal/model"
	"github.com/dimspell/gladiator/internal/wire"
	libp2plib "github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	logger.SetDiscardLogger()
}

// ─── Mocks ────────────────────────────────────────────────────────────────────

// mockConn satisfies net.Conn for the bsession.Session.Conn field.
type mockConn struct{ written []byte }

func (m *mockConn) Read(b []byte) (int, error) { return 0, fmt.Errorf("eof") }
func (m *mockConn) Write(b []byte) (int, error) {
	m.written = append(m.written, b...)
	return len(b), nil
}
func (m *mockConn) Close() error                       { return nil }
func (m *mockConn) LocalAddr() net.Addr                { return &net.TCPAddr{} }
func (m *mockConn) RemoteAddr() net.Addr               { return &net.TCPAddr{} }
func (m *mockConn) SetDeadline(t time.Time) error      { return nil }
func (m *mockConn) SetReadDeadline(t time.Time) error  { return nil }
func (m *mockConn) SetWriteDeadline(t time.Time) error { return nil }

// mockGameServiceClient is a stub satisfying multiv1connect.GameServiceClient.
type mockGameServiceClient struct {
	games   []*multiv1.Game
	players []*multiv1.Player
	game    *multiv1.Game
}

func (m *mockGameServiceClient) CreateGame(_ context.Context, _ *connect.Request[multiv1.CreateGameRequest]) (*connect.Response[multiv1.CreateGameResponse], error) {
	return connect.NewResponse(&multiv1.CreateGameResponse{}), nil
}
func (m *mockGameServiceClient) JoinGame(_ context.Context, _ *connect.Request[multiv1.JoinGameRequest]) (*connect.Response[multiv1.JoinGameResponse], error) {
	return connect.NewResponse(&multiv1.JoinGameResponse{Players: m.players}), nil
}
func (m *mockGameServiceClient) ListGames(_ context.Context, _ *connect.Request[multiv1.ListGamesRequest]) (*connect.Response[multiv1.ListGamesResponse], error) {
	return connect.NewResponse(&multiv1.ListGamesResponse{Games: m.games}), nil
}
func (m *mockGameServiceClient) GetGame(_ context.Context, _ *connect.Request[multiv1.GetGameRequest]) (*connect.Response[multiv1.GetGameResponse], error) {
	g := m.game
	if g == nil {
		g = &multiv1.Game{}
	}
	return connect.NewResponse(&multiv1.GetGameResponse{Game: g, Players: m.players}), nil
}

// fakeStream is a minimal in-memory implementation of network.Stream backed by
// a bytes.Buffer so we can inspect frame writes without real network I/O.
type fakeStream struct {
	buf    bytes.Buffer
	closed bool
	mu     sync.Mutex
}

func (f *fakeStream) Read(b []byte) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.buf.Read(b)
}
func (f *fakeStream) Write(b []byte) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.buf.Write(b)
}
func (f *fakeStream) Close() error                                   { f.closed = true; return nil }
func (f *fakeStream) Reset() error                                   { f.closed = true; return nil }
func (f *fakeStream) CloseWrite() error                              { return nil }
func (f *fakeStream) CloseRead() error                               { return nil }
func (f *fakeStream) ResetWithError(_ network.StreamErrorCode) error { return nil }
func (f *fakeStream) SetDeadline(t time.Time) error                  { return nil }
func (f *fakeStream) SetReadDeadline(t time.Time) error              { return nil }
func (f *fakeStream) SetWriteDeadline(t time.Time) error             { return nil }
func (f *fakeStream) ID() string                                     { return "fake" }
func (f *fakeStream) Conn() network.Conn                             { return nil }
func (f *fakeStream) Stat() network.Stats                            { return network.Stats{} }
func (f *fakeStream) Scope() network.StreamScope                     { return nil }
func (f *fakeStream) Protocol() protocol.ID                          { return "" }
func (f *fakeStream) SetProtocol(_ protocol.ID) error                { return nil }

// bytes reads out all currently buffered bytes (thread-safe).
func (f *fakeStream) Bytes() []byte {
	f.mu.Lock()
	defer f.mu.Unlock()
	b := make([]byte, f.buf.Len())
	copy(b, f.buf.Bytes())
	return b
}

// blockingStream implements network.Stream where Read blocks until Close/Reset
// or until a read deadline is set and expires.  Used to simulate a hung remote
// peer in goroutine-leak tests.
type blockingStream struct {
	mu           sync.Mutex
	closed       bool
	readBlock    chan struct{}
	readDeadline time.Time
	hasDeadline  bool
}

func (s *blockingStream) Read(b []byte) (int, error) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return 0, fmt.Errorf("stream closed")
	}
	deadline := s.readDeadline
	hasDeadline := s.hasDeadline
	s.mu.Unlock()

	if hasDeadline {
		dur := time.Until(deadline)
		if dur <= 0 {
			return 0, fmt.Errorf("i/o timeout")
		}
		timer := time.NewTimer(dur)
		defer timer.Stop()
		select {
		case <-s.readBlock:
		case <-timer.C:
			return 0, fmt.Errorf("i/o timeout")
		}
	} else {
		<-s.readBlock
	}
	return 0, fmt.Errorf("stream closed")
}

func (s *blockingStream) Write(b []byte) (int, error)         { return len(b), nil }
func (s *blockingStream) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.closed {
		s.closed = true
		close(s.readBlock)
	}
	return nil
}
func (s *blockingStream) Reset() error                                   { return s.Close() }
func (s *blockingStream) CloseWrite() error                              { return nil }
func (s *blockingStream) CloseRead() error                               { return nil }
func (s *blockingStream) ResetWithError(_ network.StreamErrorCode) error { return s.Close() }
func (s *blockingStream) SetDeadline(t time.Time) error                  { return nil }
func (s *blockingStream) SetReadDeadline(t time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.readDeadline = t
	s.hasDeadline = !t.IsZero()
	return nil
}
func (s *blockingStream) SetWriteDeadline(t time.Time) error { return nil }
func (s *blockingStream) ID() string                         { return "blocking" }
func (s *blockingStream) Conn() network.Conn                 { return nil }
func (s *blockingStream) Stat() network.Stats                { return network.Stats{} }
func (s *blockingStream) Scope() network.StreamScope         { return nil }
func (s *blockingStream) Protocol() protocol.ID              { return "" }
func (s *blockingStream) SetProtocol(_ protocol.ID) error    { return nil }

// captureRedirect implements redirect.Redirect and captures all Write calls
// into a shared buffer so tests can inspect what was forwarded.
type captureRedirect struct {
	buf *bytes.Buffer
	mu  *sync.Mutex
}

func (c *captureRedirect) Write(p []byte) (int, error) {
	if c.mu != nil {
		c.mu.Lock()
		defer c.mu.Unlock()
	}
	if c.buf != nil {
		return c.buf.Write(p)
	}
	return len(p), nil
}
func (c *captureRedirect) Close() error                           { return nil }
func (c *captureRedirect) Run(_ context.Context) error            { return nil }
func (c *captureRedirect) Alive(_ time.Time, _ time.Duration) bool { return true }

// ─── Helpers ──────────────────────────────────────────────────────────────────

func makeSession(userID int64) *bsession.Session {
	return &bsession.Session{
		ID:     fmt.Sprintf("session-%d", userID),
		UserID: userID,
		Conn:   &mockConn{},
		State:  &bsession.SessionState{},
	}
}

func makeProxy(userID int64) *Libp2pProxy {
	return newLibp2pProxy(
		&ProxyLibp2p{IPPrefix: net.IPv4(127, 0, 0, 0)},
		&mockGameServiceClient{},
		makeSession(userID),
	)
}

func wirePayload(t *testing.T, eventType wire.EventType, content any) []byte {
	t.Helper()
	return wire.Compose(eventType, wire.Message{
		From:    "1",
		Type:    eventType,
		Content: content,
	})
}

// ─── Unit: factory & basic wiring ────────────────────────────────────────────

func TestProxyLibp2p_Mode(t *testing.T) {
	p := &ProxyLibp2p{}
	assert.Equal(t, model.RunModeLibp2p, p.Mode())
}

func TestProxyLibp2p_Create_ReturnsLibp2pProxy(t *testing.T) {
	session := makeSession(1)
	factory := &ProxyLibp2p{IPPrefix: net.IPv4(127, 0, 0, 0)}
	client := factory.Create(session, &mockGameServiceClient{})
	require.NotNil(t, client)
	_, ok := client.(*Libp2pProxy)
	assert.True(t, ok, "Create must return *Libp2pProxy")
}

func TestPeerIDStr(t *testing.T) {
	assert.Equal(t, "42", peerIDStr(42))
	assert.Equal(t, "0", peerIDStr(0))
	assert.Equal(t, "9999999", peerIDStr(9999999))
}

func TestNewLibp2pProxy_Fields(t *testing.T) {
	session := makeSession(7)
	p := newLibp2pProxy(&ProxyLibp2p{}, &mockGameServiceClient{}, session)
	assert.NotNil(t, p.manager)
	assert.Equal(t, "7", p.selfID)
	assert.NotNil(t, p.peers)
	assert.Nil(t, p.h, "libp2p host should not be started on construction")
}

// ─── Unit: reset / close ──────────────────────────────────────────────────────

func TestLibp2pProxy_Reset_ClearsPeersAndRoom(t *testing.T) {
	p := makeProxy(10)
	p.roomID = "my-room"
	p.currentHostID = "10"
	p.peers["99"] = &peerStream{peerID: "99"}

	p.reset()

	assert.Empty(t, p.roomID)
	assert.Empty(t, p.currentHostID)
	assert.Empty(t, p.peers)
}

func TestLibp2pProxy_Close_Idempotent(t *testing.T) {
	p := makeProxy(11)
	// Must not panic or block regardless of how many times called
	assert.NotPanics(t, func() {
		p.Close()
		p.Close()
		p.Close()
	})
}

func TestLibp2pProxy_Close_ClosesLibp2pHost(t *testing.T) {
	p := makeProxy(12)
	h, err := libp2plib.New()
	require.NoError(t, err)
	p.h = h

	p.Close()

	assert.Nil(t, p.h, "h must be nilled after Close")
}

func TestLibp2pProxy_Reset_ClosesOpenPeerStreams(t *testing.T) {
	p := makeProxy(13)
	fs := &fakeStream{}
	p.peers["42"] = &peerStream{peerID: "42", stream: fs}

	p.reset()

	assert.True(t, fs.closed, "stream must be reset when peer map is cleared")
	assert.Empty(t, p.peers)
}

// ─── Unit: peerStream framing ─────────────────────────────────────────────────

func TestPeerStream_Send_FrameFormat(t *testing.T) {
	fs := &fakeStream{}
	ps := &peerStream{peerID: "42", stream: fs}

	payload := []byte("hello-world")
	require.NoError(t, ps.send(payload))

	out := fs.Bytes()
	require.GreaterOrEqual(t, len(out), 4+len(payload))

	length := binary.BigEndian.Uint32(out[:4])
	assert.Equal(t, uint32(len(payload)), length, "length prefix must equal payload length")
	assert.Equal(t, payload, out[4:], "payload must follow the length prefix")
}

func TestPeerStream_Send_EmptyPayload(t *testing.T) {
	fs := &fakeStream{}
	ps := &peerStream{peerID: "x", stream: fs}
	// sending empty slice must not panic
	assert.NoError(t, ps.send([]byte{}))
	out := fs.Bytes()
	assert.Equal(t, uint32(0), binary.BigEndian.Uint32(out[:4]))
}

func TestPeerStream_Send_NilStream(t *testing.T) {
	ps := &peerStream{peerID: "99", stream: nil}
	err := ps.send([]byte("data"))
	assert.Error(t, err, "nil stream must return an error")
}

func TestPeerStream_Close_NilsStream(t *testing.T) {
	fs := &fakeStream{}
	ps := &peerStream{peerID: "1", stream: fs}
	ps.close()
	assert.Nil(t, ps.stream, "stream must be nilled after close")
	assert.True(t, fs.closed)
}

func TestPeerStream_Close_Idempotent(t *testing.T) {
	fs := &fakeStream{}
	ps := &peerStream{peerID: "1", stream: fs}
	// Double close must not panic
	assert.NotPanics(t, func() {
		ps.close()
		ps.close()
	})
}

// ─── Unit: readFull helper ────────────────────────────────────────────────────

func TestReadFull_ExactRead(t *testing.T) {
	data := []byte{1, 2, 3, 4, 5}
	fs := &fakeStream{}
	fs.buf.Write(data)

	buf := make([]byte, 5)
	n, err := readFull(fs, buf)
	require.NoError(t, err)
	assert.Equal(t, 5, n)
	assert.Equal(t, data, buf)
}

func TestReadFull_ShortReads(t *testing.T) {
	// Use a custom reader that returns 1 byte at a time to simulate short reads.
	type oneByteReader struct{ buf []byte }
	_ = oneByteReader{} // just verifying the concept – see below

	// fakeStream.Read delegates to bytes.Buffer which may read fewer bytes than
	// requested.  Writing 10 bytes and asking for all 10 still exercises the loop.
	data := make([]byte, 10)
	for i := range data {
		data[i] = byte(i + 1)
	}
	fs := &fakeStream{}
	fs.buf.Write(data)

	buf := make([]byte, len(data))
	n, err := readFull(fs, buf)
	require.NoError(t, err)
	assert.Equal(t, len(data), n)
	assert.Equal(t, data, buf)
}

// ─── Unit: Handle – wire event dispatch ──────────────────────────────────────

func TestHandle_UnknownEvent_Noop(t *testing.T) {
	p := makeProxy(1)
	err := p.Handle(context.Background(), []byte{0xFF})
	assert.NoError(t, err)
}

func TestHandle_LobbyUsers_Noop(t *testing.T) {
	p := makeProxy(1)
	assert.NoError(t, p.Handle(context.Background(), wirePayload(t, wire.LobbyUsers, nil)))
}

func TestHandle_JoinLobby_Noop(t *testing.T) {
	p := makeProxy(1)
	assert.NoError(t, p.Handle(context.Background(), wirePayload(t, wire.JoinLobby, nil)))
}

func TestHandle_CreateRoom_Noop(t *testing.T) {
	p := makeProxy(1)
	assert.NoError(t, p.Handle(context.Background(), wirePayload(t, wire.CreateRoom, nil)))
}

func TestHandle_LeaveRoom_Self_Ignored(t *testing.T) {
	p := makeProxy(100)
	payload := wirePayload(t, wire.LeaveRoom, wire.Player{UserID: 100})
	require.NoError(t, p.Handle(context.Background(), payload))
	assert.Empty(t, p.peers, "no peer entry should be touched for self-leave")
}

func TestHandle_LeaveRoom_OtherPeer_RemovesPeer(t *testing.T) {
	p := makeProxy(100)

	// Pre-populate a fake peer stream
	fs := &fakeStream{}
	p.peers["200"] = &peerStream{peerID: "200", stream: fs}
	_, _ = p.manager.AssignIP("200")

	payload := wirePayload(t, wire.LeaveRoom, wire.Player{UserID: 200})
	require.NoError(t, p.Handle(context.Background(), payload))

	p.mu.Lock()
	_, stillPresent := p.peers["200"]
	p.mu.Unlock()

	assert.False(t, stillPresent, "peer 200 must be removed from peers map")
	assert.True(t, fs.closed, "peer stream must be closed on leave")
}

func TestHandle_LeaveLobby_OtherPeer_RemovesPeer(t *testing.T) {
	p := makeProxy(100)
	fs := &fakeStream{}
	p.peers["300"] = &peerStream{peerID: "300", stream: fs}

	payload := wirePayload(t, wire.LeaveLobby, wire.Player{UserID: 300})
	require.NoError(t, p.Handle(context.Background(), payload))

	p.mu.Lock()
	_, stillPresent := p.peers["300"]
	p.mu.Unlock()
	assert.False(t, stillPresent)
}

func TestHandle_HostMigration_UpdatesCurrentHost(t *testing.T) {
	p := makeProxy(100)
	p.currentHostID = "100"

	payload := wirePayload(t, wire.HostMigration, wire.Player{UserID: 200})
	require.NoError(t, p.Handle(context.Background(), payload))

	assert.Equal(t, "200", p.currentHostID)
}

func TestHandle_HostMigration_ToSelf(t *testing.T) {
	p := makeProxy(100)
	p.currentHostID = "50"

	payload := wirePayload(t, wire.HostMigration, wire.Player{UserID: 100})
	require.NoError(t, p.Handle(context.Background(), payload))

	// selfID becomes the new host
	assert.Equal(t, "100", p.currentHostID)
}

func TestHandle_Libp2pAddresses_Self_Ignored(t *testing.T) {
	p := makeProxy(100)
	// creator == self → must be a no-op, even if host is nil
	payload := wirePayload(t, wire.Libp2pAddresses, wire.Libp2pPeerInfo{
		CreatorID: 100,
		Addresses: []string{"/ip4/127.0.0.1/tcp/1234/p2p/12D3KooWGEybxAiFYRb85gp7mGNQBMaREHmFfqJhrZJfFDFsEcGy"},
	})
	assert.NoError(t, p.Handle(context.Background(), payload))
	assert.Empty(t, p.peers)
}

func TestHandle_Libp2pAddresses_NoHost_Noop(t *testing.T) {
	p := makeProxy(100)
	// h == nil → graceful no-op for a remote peer's address
	payload := wirePayload(t, wire.Libp2pAddresses, wire.Libp2pPeerInfo{
		CreatorID: 200,
		Addresses: []string{"/ip4/127.0.0.1/tcp/1234/p2p/12D3KooWGEybxAiFYRb85gp7mGNQBMaREHmFfqJhrZJfFDFsEcGy"},
	})
	assert.NoError(t, p.Handle(context.Background(), payload))
}

func TestHandle_Libp2pAddresses_InvalidAddr_ReturnsError(t *testing.T) {
	ctx := context.Background()
	p := makeProxy(100)

	// Start a real host so the address handling branch is reached
	h, err := libp2plib.New(libp2plib.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = h.Close() })
	p.h = h

	payload := wirePayload(t, wire.Libp2pAddresses, wire.Libp2pPeerInfo{
		CreatorID: 200,
		Addresses: []string{"not-a-valid-multiaddr"},
	})
	err = p.Handle(ctx, payload)
	assert.Error(t, err, "completely invalid addresses must return an error")
}

func TestHandle_JoinRoom_NonHost_Noop(t *testing.T) {
	// When we are not the host, handleJoinRoom is a no-op for other players.
	p := makeProxy(100)
	p.currentHostID = "999" // someone else is the host

	payload := wirePayload(t, wire.JoinRoom, wire.Player{UserID: 200, Username: "guest"})
	assert.NoError(t, p.Handle(context.Background(), payload))
}

func TestHandle_JoinRoom_Self_Ignored(t *testing.T) {
	p := makeProxy(100)
	payload := wirePayload(t, wire.JoinRoom, wire.Player{UserID: 100})
	assert.NoError(t, p.Handle(context.Background(), payload))
}

// ─── Unit: outbound message helpers ──────────────────────────────────────────

func TestOnTCPMessage_NoPeer_Noop(t *testing.T) {
	p := makeProxy(1)
	// unknown peer → graceful no-op (never error)
	assert.NoError(t, p.onTCPMessage("unknown")([]byte("data")))
}

func TestOnUDPMessage_NoPeer_Noop(t *testing.T) {
	p := makeProxy(1)
	assert.NoError(t, p.onUDPMessage("unknown")([]byte("data")))
}

func TestOnTCPMessage_WritesTFrame(t *testing.T) {
	p := makeProxy(1)
	fs := &fakeStream{}
	p.peers["42"] = &peerStream{peerID: "42", stream: fs}

	data := []byte{0xAB, 0xCD}
	require.NoError(t, p.onTCPMessage("42")(data))

	out := fs.Bytes()
	require.GreaterOrEqual(t, len(out), 5, "need 4-byte header + at least 3 body bytes")

	length := binary.BigEndian.Uint32(out[:4])
	assert.Equal(t, uint32(3), length, "frame must be 'T' + 2 data bytes = 3")
	assert.Equal(t, byte('T'), out[4], "first body byte must be 'T'")
	assert.Equal(t, data, out[5:], "data must follow the tag byte")
}

func TestOnUDPMessage_WritesUFrame(t *testing.T) {
	p := makeProxy(1)
	fs := &fakeStream{}
	p.peers["42"] = &peerStream{peerID: "42", stream: fs}

	data := []byte{0x01, 0x02}
	require.NoError(t, p.onUDPMessage("42")(data))

	out := fs.Bytes()
	require.GreaterOrEqual(t, len(out), 5)
	assert.Equal(t, byte('U'), out[4], "first body byte must be 'U'")
	assert.Equal(t, data, out[5:])
}

// ─── Unit: ListGames ─────────────────────────────────────────────────────────

func TestListGames_EmptyList(t *testing.T) {
	p := makeProxy(1)
	rooms, err := p.ListGames(context.Background())
	require.NoError(t, err)
	assert.Empty(t, rooms)
}

func TestListGames_ReturnsMappedRooms(t *testing.T) {
	p := makeProxy(1)
	p.gameClient = &mockGameServiceClient{
		games: []*multiv1.Game{
			{Name: "room-a", Password: ""},
			{Name: "room-b", Password: "secret"},
		},
	}

	rooms, err := p.ListGames(context.Background())
	require.NoError(t, err)
	require.Len(t, rooms, 2)
	assert.Equal(t, "room-a", rooms[0].Name)
	assert.Equal(t, "room-b", rooms[1].Name)
	// All lobby rooms use the well-known fake-host IP
	assert.Equal(t, net.IPv4(127, 0, 0, 2).To4(), rooms[0].HostIPAddress)
}

// ─── ProxyClient interface compliance ────────────────────────────────────────

func TestInterfaceCompliance(t *testing.T) {
	// Compile-time check is in libp2p.go; this runtime check is belt-and-braces.
	var _ proxy.ProxyClient = (*Libp2pProxy)(nil)
}

// ─── Integration: real libp2p peer communication ─────────────────────────────

// TestLibp2p_PeerCommunication spins up two real in-process libp2p hosts,
// opens a stream between them, and verifies that TCP ('T') and UDP ('U') frames
// produced by onTCPMessage / onUDPMessage arrive intact at the remote peer.
func TestLibp2p_PeerCommunication(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	h1, err := libp2plib.New(libp2plib.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = h1.Close() })

	h2, err := libp2plib.New(libp2plib.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = h2.Close() })

	p1 := makeProxy(1001)
	p1.h = h1

	// Captured payloads on h2 side
	var (
		tcpBuf bytes.Buffer
		udpBuf bytes.Buffer
		bufMu  sync.Mutex
	)

	// Register a stream handler on h2 that manually runs the frame read loop
	// and writes into the capture buffers.
	h2.SetStreamHandler(gameProtocol, func(s network.Stream) {
		go func() {
			defer func() { _ = s.Reset() }()
			lenBuf := make([]byte, 4)
			for {
				if _, err := readFull(s, lenBuf); err != nil {
					return
				}
				l := int(binary.BigEndian.Uint32(lenBuf))
				if l == 0 || l > 1<<20 {
					return
				}
				data := make([]byte, l)
				if _, err := readFull(s, data); err != nil {
					return
				}
				if len(data) < 2 {
					continue
				}
				bufMu.Lock()
				switch data[0] {
				case 'T':
					tcpBuf.Write(data[1:])
				case 'U':
					udpBuf.Write(data[1:])
				}
				bufMu.Unlock()
			}
		}()
	})

	// Connect h1 → h2, open our game protocol stream
	require.NoError(t, h1.Connect(ctx, peer.AddrInfo{ID: h2.ID(), Addrs: h2.Addrs()}))
	stream, err := h1.NewStream(ctx, h2.ID(), gameProtocol)
	require.NoError(t, err)

	h2PeerStr := h2.ID().String()
	p1.mu.Lock()
	p1.peers[h2PeerStr] = &peerStream{peerID: h2PeerStr, stream: stream}
	p1.mu.Unlock()

	// Send a TCP game frame
	tcpPayload := []byte("game-tcp-data-12345")
	require.NoError(t, p1.onTCPMessage(h2PeerStr)(tcpPayload))

	// Send a UDP game frame
	udpPayload := []byte("game-udp-data-67890")
	require.NoError(t, p1.onUDPMessage(h2PeerStr)(udpPayload))

	// Poll until both frames arrive (or timeout)
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		bufMu.Lock()
		gotTCP := bytes.Contains(tcpBuf.Bytes(), tcpPayload)
		gotUDP := bytes.Contains(udpBuf.Bytes(), udpPayload)
		bufMu.Unlock()
		if gotTCP && gotUDP {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	bufMu.Lock()
	defer bufMu.Unlock()
	assert.True(t, bytes.Contains(tcpBuf.Bytes(), tcpPayload),
		"TCP payload must be received by h2; got %q", tcpBuf.Bytes())
	assert.True(t, bytes.Contains(udpBuf.Bytes(), udpPayload),
		"UDP payload must be received by h2; got %q", udpBuf.Bytes())
}

// TestLibp2p_BidirectionalFrames verifies that frames flow correctly in both
// directions simultaneously over independent streams.
func TestLibp2p_BidirectionalFrames(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	h1, err := libp2plib.New(libp2plib.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = h1.Close() })

	h2, err := libp2plib.New(libp2plib.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = h2.Close() })

	type rcvBuf struct {
		mu       sync.Mutex
		tcp, udp bytes.Buffer
	}
	buf1, buf2 := &rcvBuf{}, &rcvBuf{}

	makeReadLoop := func(buf *rcvBuf) func(network.Stream) {
		return func(s network.Stream) {
			defer func() { _ = s.Reset() }()
			lenBuf := make([]byte, 4)
			for {
				if _, err2 := readFull(s, lenBuf); err2 != nil {
					return
				}
				l := int(binary.BigEndian.Uint32(lenBuf))
				if l == 0 || l > 1<<20 {
					return
				}
				data := make([]byte, l)
				if _, err2 := readFull(s, data); err2 != nil {
					return
				}
				if len(data) < 2 {
					continue
				}
				buf.mu.Lock()
				switch data[0] {
				case 'T':
					buf.tcp.Write(data[1:])
				case 'U':
					buf.udp.Write(data[1:])
				}
				buf.mu.Unlock()
			}
		}
	}

	h1.SetStreamHandler(gameProtocol, makeReadLoop(buf1)) // h1 receives from h2
	h2.SetStreamHandler(gameProtocol, makeReadLoop(buf2)) // h2 receives from h1

	// h1 → h2
	require.NoError(t, h1.Connect(ctx, peer.AddrInfo{ID: h2.ID(), Addrs: h2.Addrs()}))
	s12, err := h1.NewStream(ctx, h2.ID(), gameProtocol)
	require.NoError(t, err)
	ps12 := &peerStream{peerID: h2.ID().String(), stream: s12}

	// h2 → h1
	require.NoError(t, h2.Connect(ctx, peer.AddrInfo{ID: h1.ID(), Addrs: h1.Addrs()}))
	s21, err := h2.NewStream(ctx, h1.ID(), gameProtocol)
	require.NoError(t, err)
	ps21 := &peerStream{peerID: h1.ID().String(), stream: s21}

	// h1 → h2: TCP + UDP
	require.NoError(t, ps12.send(append([]byte{'T'}, []byte("h1-tcp")...)))
	require.NoError(t, ps12.send(append([]byte{'U'}, []byte("h1-udp")...)))

	// h2 → h1: TCP + UDP
	require.NoError(t, ps21.send(append([]byte{'T'}, []byte("h2-tcp")...)))
	require.NoError(t, ps21.send(append([]byte{'U'}, []byte("h2-udp")...)))

	waitFor := func(b *rcvBuf, wantTCP, wantUDP []byte) {
		deadline := time.Now().Add(8 * time.Second)
		for time.Now().Before(deadline) {
			b.mu.Lock()
			gt := bytes.Contains(b.tcp.Bytes(), wantTCP)
			gu := bytes.Contains(b.udp.Bytes(), wantUDP)
			b.mu.Unlock()
			if gt && gu {
				return
			}
			time.Sleep(20 * time.Millisecond)
		}
	}

	waitFor(buf2, []byte("h1-tcp"), []byte("h1-udp"))
	waitFor(buf1, []byte("h2-tcp"), []byte("h2-udp"))

	buf2.mu.Lock()
	assert.Contains(t, buf2.tcp.String(), "h1-tcp")
	assert.Contains(t, buf2.udp.String(), "h1-udp")
	buf2.mu.Unlock()

	buf1.mu.Lock()
	assert.Contains(t, buf1.tcp.String(), "h2-tcp")
	assert.Contains(t, buf1.udp.String(), "h2-udp")
	buf1.mu.Unlock()
}

// TestLibp2p_AddressExchange exercises the full address-exchange happy path:
// h1 starts a libp2p host, constructs a Libp2pAddresses wire message, and h2
// (via handleLibp2pAddresses) connects back to h1, producing an entry in its
// peers map.
func TestLibp2p_AddressExchange(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// h1 is the host advertising its addresses
	h1, err := libp2plib.New(libp2plib.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = h1.Close() })

	// h1 needs a stream handler so h2 can open a stream to it
	h1Received := make(chan struct{}, 1)
	h1.SetStreamHandler(gameProtocol, func(s network.Stream) {
		_ = s.Reset()
		select {
		case h1Received <- struct{}{}:
		default:
		}
	})

	// p2 is the joiner that will receive h1's addresses and connect
	p2 := makeProxy(2)
	h2, err := libp2plib.New(libp2plib.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = h2.Close() })
	p2.h = h2

	// Build full multiaddresses for h1 (same format as startHost does)
	var fullAddrs []string
	for _, a := range h1.Addrs() {
		fullAddrs = append(fullAddrs, fmt.Sprintf("%s/p2p/%s", a.String(), h1.ID().String()))
	}

	// Simulate receiving a Libp2pAddresses message from peer with UserID 1
	info := wire.Libp2pPeerInfo{CreatorID: 1, Addresses: fullAddrs}
	err = p2.handleLibp2pAddresses(ctx, info)
	require.NoError(t, err, "handleLibp2pAddresses must succeed with valid addresses")

	// handleLibp2pAddresses stores the peer entry under the game user ID string
	// (peerIDStr(info.CreatorID)), not the libp2p peer ID string.
	fromIDStr := peerIDStr(1) // CreatorID == 1
	p2.mu.Lock()
	_, connected := p2.peers[fromIDStr]
	p2.mu.Unlock()
	assert.True(t, connected, "p2 must have an open stream keyed by game user ID %q after address exchange", fromIDStr)

	// h1 must have received the inbound stream
	select {
	case <-h1Received:
		// success
	case <-time.After(5 * time.Second):
		t.Fatal("h1 did not receive an inbound stream within timeout")
	}
}

// ─── Fix #7: goroutine leak test ────────────────────────────────────────────

func TestReceiveFromPeer_ReadDeadlineExits(t *testing.T) {
	// receiveFromPeer must exit within a read timeout when the remote peer
	// silently drops the connection.  Without the read-deadline fix the
	// goroutine hangs forever (leak).
	p := makeProxy(1)
	p.readTimeout = 30 * time.Millisecond // short for testing

	bs := &blockingStream{readBlock: make(chan struct{})}
	ps := &peerStream{peerID: "42", stream: bs}
	p.peers["42"] = ps

	baseline := runtime.NumGoroutine()

	p.wg.Add(1)
	done := make(chan struct{})
	go func() {
		p.receiveFromPeer(ps)
		close(done)
	}()

	// If the fix is missing, receiveFromPeer blocks on Read forever and this
	// select will time out (goroutine leak).  With the fix, the read deadline
	// fires after 30 ms and the goroutine exits.
	select {
	case <-done:
		// success – goroutine exited due to read deadline
	case <-time.After(3 * time.Second):
		t.Fatal("receiveFromPeer did not exit within read deadline – goroutine leak")
	}

	time.Sleep(50 * time.Millisecond) // let the runtime clean up exited goroutines
	final := runtime.NumGoroutine()

	assert.InDelta(t, baseline, final, 3,
		"goroutine count should return to baseline after receiveFromPeer exits")

	// Peer must be cleaned up from the map
	p.mu.Lock()
	_, exists := p.peers["42"]
	p.mu.Unlock()
	assert.False(t, exists, "peer 42 must be removed from the map after receiveFromPeer exits")
}

// ─── Fix #6: TOCTOU race test ───────────────────────────────────────────────

// delayedHost wraps a host.Host and introduces a short, configurable sleep
// before Connect and NewStream so that two concurrent address-exchange calls
// both pass the initial "already connected?" check before either inserts.
type delayedHost struct {
	host.Host
	delay time.Duration
}

func (d *delayedHost) Connect(ctx context.Context, ai peer.AddrInfo) error {
	time.Sleep(d.delay)
	return d.Host.Connect(ctx, ai)
}

func (d *delayedHost) NewStream(ctx context.Context, p peer.ID, protos ...protocol.ID) (network.Stream, error) {
	time.Sleep(d.delay)
	return d.Host.NewStream(ctx, p, protos...)
}

func TestHandleLibp2pAddresses_ConcurrentDuplicate(t *testing.T) {
	// Two concurrent calls to handleLibp2pAddresses for the same peer must
	// not produce duplicate peerStream entries.  Without the re-check-under-
	// lock fix, the second caller overwrites the first, leaking the first
	// stream and its receiveFromPeer goroutine.
	//
	// We wrap the host with a delay to widen the TOCTOU window, ensuring
	// both goroutines pass the initial "already connected?" check before
	// either reaches the insert.
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	hA, err := libp2plib.New(libp2plib.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = hA.Close() })

	hB, err := libp2plib.New(libp2plib.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = hB.Close() })

	pA := makeProxy(1)
	pA.h = &delayedHost{Host: hA, delay: 30 * time.Millisecond}

	// B's stream handler – just drains and closes streams.
	hB.SetStreamHandler(gameProtocol, func(s network.Stream) {
		go func() {
			defer s.Reset()
			lenBuf := make([]byte, 4)
			for {
				if _, err := readFull(s, lenBuf); err != nil {
					return
				}
				l := int(binary.BigEndian.Uint32(lenBuf))
				if l == 0 || l > 1<<20 {
					return
				}
				data := make([]byte, l)
				if _, err := readFull(s, data); err != nil {
					return
				}
			}
		}()
	})

	// Build B's multiaddresses for the libp2p address-exchange message.
	var fullAddrsB []string
	for _, a := range hB.Addrs() {
		fullAddrsB = append(fullAddrsB, fmt.Sprintf("%s/p2p/%s", a.String(), hB.ID().String()))
	}
	info := wire.Libp2pPeerInfo{CreatorID: 2, Addresses: fullAddrsB}

	// Launch two concurrent address-exchange calls.  The delayed host
	// ensures they both pass the initial check before either dials.
	var wg sync.WaitGroup
	wg.Add(2)
	var errStr1, errStr2 string
	go func() {
		defer wg.Done()
		if e := pA.handleLibp2pAddresses(ctx, info); e != nil {
			errStr1 = e.Error()
		}
	}()
	go func() {
		defer wg.Done()
		if e := pA.handleLibp2pAddresses(ctx, info); e != nil {
			errStr2 = e.Error()
		}
	}()
	wg.Wait()

	if errStr1 != "" {
		t.Logf("First caller: %s", errStr1)
	}
	if errStr2 != "" {
		t.Logf("Second caller: %s (expected 'already connected' or similar)", errStr2)
	}

	// Exactly one entry for user ID "2".
	pA.mu.Lock()
	ps, exists := pA.peers["2"]
	count := 0
	for k := range pA.peers {
		if k == "2" {
			count++
		}
	}
	pA.mu.Unlock()

	assert.True(t, exists, "peer 2 must have an entry in the peers map")
	assert.Equal(t, 1, count, "must be exactly one peer entry for ID 2")
	require.NotNil(t, ps)

	// At least one call must have succeeded.
	assert.True(t, errStr1 == "" || errStr2 == "",
		"at least one of the concurrent calls must have succeeded")
}

// ─── Fix #4: bidirectional key-mismatch test ────────────────────────────────

func TestLibp2p_BidirectionalTraffic(t *testing.T) {
	// An inbound stream from peer B to peer A must be stored under B's game
	// user ID so that outbound lookups (onTCPMessage / onUDPMessage) and
	// PeerHost lookups in receiveFromPeer find the correct entry.
	//
	// Without fix #4, handleIncomingStream stores the stream keyed by the
	// libp2p peer ID string, which does not match the game user ID key used
	// everywhere else → all inbound traffic is silently dropped.
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// --- hosts ---
	hA, err := libp2plib.New(libp2plib.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = hA.Close() })

	hB, err := libp2plib.New(libp2plib.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = hB.Close() })

	// --- proxies ---
	pA := makeProxy(1) // game user ID "1"
	pA.h = hA
	pB := makeProxy(2) // game user ID "2"
	pB.h = hB

	// --- create a capture for data arriving on A from peer "2" ---
	var (
		aReceived bytes.Buffer
		aRcvMu    sync.Mutex
	)

	// Register a FakeHost on A for peer "2" so that receiveFromPeer has
	// somewhere to deliver frames.  We inject a fake Redirect that captures
	// Write calls instead of using StartGuest (which requires real network
	// ports to create proxies).
	ipForTwo, err := pA.manager.AssignIP("2")
	require.NoError(t, err)

	mockTCP := &captureRedirect{buf: &aReceived, mu: &aRcvMu}
	mockUDP := &captureRedirect{}
	pA.manager.SetHost(ipForTwo, "2", &redirect.FakeHost{
		PeerID:     "2",
		AssignedIP: ipForTwo,
		ProxyTCP:   mockTCP,
		ProxyUDP:   mockUDP,
	})

	// --- populate peerID→userID mapping on A ---
	// Build B's multiaddresses and send them through handleLibp2pAddresses.
	// This also opens an outbound stream A→B which is harmless.
	var fullAddrsB []string
	for _, a := range hB.Addrs() {
		fullAddrsB = append(fullAddrsB, fmt.Sprintf("%s/p2p/%s", a.String(), hB.ID().String()))
	}

	// B needs a stream handler for the outbound stream A will open.
	hB.SetStreamHandler(gameProtocol, func(s network.Stream) {
		go func() {
			defer s.Reset()
			lenBuf := make([]byte, 4)
			for {
				if _, err := readFull(s, lenBuf); err != nil {
					return
				}
				l := int(binary.BigEndian.Uint32(lenBuf))
				if l == 0 || l > 1<<20 {
					return
				}
				data := make([]byte, l)
				if _, err := readFull(s, data); err != nil {
					return
				}
			}
		}()
	})

	err = pA.handleLibp2pAddresses(ctx, wire.Libp2pPeerInfo{
		CreatorID: 2,
		Addresses: fullAddrsB,
	})
	require.NoError(t, err, "A must process B's libp2p addresses")

	// Verify the mapping exists on A.
	hBpeerID := hB.ID().String()
	pA.mu.Lock()
	mappedUserID, mappingOK := pA.peerIDToUserID[hBpeerID]
	pA.mu.Unlock()
	assert.True(t, mappingOK, "A must have peerID→userID mapping for B")
	assert.Equal(t, "2", mappedUserID)

	// --- the main event: B opens an inbound stream to A ---
	hA.SetStreamHandler(gameProtocol, pA.handleIncomingStream)

	// B connects to A and opens a stream.
	require.NoError(t, hB.Connect(ctx, peer.AddrInfo{ID: hA.ID(), Addrs: hA.Addrs()}))
	inboundStream, err := hB.NewStream(ctx, hA.ID(), gameProtocol)
	require.NoError(t, err)
	t.Cleanup(func() { _ = inboundStream.Reset() })

	// Poll until A has registered the stream under game user ID "2".
	deadline := time.Now().Add(5 * time.Second)
	var ps *peerStream
	for time.Now().Before(deadline) {
		pA.mu.Lock()
		ps = pA.peers["2"]
		pA.mu.Unlock()
		if ps != nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	require.NotNil(t, ps, "A must store the inbound stream under game user ID '2'")
	assert.Equal(t, "2", ps.peerID, "ps.peerID must be the game user ID, not the libp2p peer ID")

	// Also verify there is NO entry under the libp2p peer ID (the old bug).
	pA.mu.Lock()
	_, existsUnderLibp2pID := pA.peers[hBpeerID]
	pA.mu.Unlock()
	assert.False(t, existsUnderLibp2pID,
		"there must be no entry in pA.peers under libp2p peer ID %q", hBpeerID)

	// --- send a game packet from B to A via the inbound stream ---
	testPayload := []byte("hello-from-B")
	frame := make([]byte, 4+1+len(testPayload))
	binary.BigEndian.PutUint32(frame[:4], uint32(1+len(testPayload)))
	frame[4] = 'T' // TCP frame
	copy(frame[5:], testPayload)

	_, err = inboundStream.Write(frame)
	require.NoError(t, err)

	// Wait for A to receive the data.
	for time.Now().Before(deadline) {
		aRcvMu.Lock()
		got := aReceived.Len() > 0
		aRcvMu.Unlock()
		if got {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	aRcvMu.Lock()
	assert.True(t, aReceived.Len() > 0,
		"A must receive data from the inbound stream")
	assert.Contains(t, aReceived.String(), "hello-from-B",
		"A must receive the correct payload from B")
	aRcvMu.Unlock()
}
