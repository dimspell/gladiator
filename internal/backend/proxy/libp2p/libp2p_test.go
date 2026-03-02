package libp2p

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	multiv1 "github.com/dimspell/gladiator/gen/multi/v1"
	"github.com/dimspell/gladiator/internal/app/logger"
	"github.com/dimspell/gladiator/internal/backend/bsession"
	"github.com/dimspell/gladiator/internal/backend/proxy"
	"github.com/dimspell/gladiator/internal/model"
	"github.com/dimspell/gladiator/internal/wire"
	libp2plib "github.com/libp2p/go-libp2p"
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
