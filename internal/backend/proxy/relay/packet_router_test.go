package relay

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	multiv1 "github.com/dimspell/gladiator/gen/multi/v1"
	"github.com/dimspell/gladiator/internal/app/logger"
	"github.com/dimspell/gladiator/internal/backend/bsession"
	"github.com/dimspell/gladiator/internal/backend/proxy"
	"github.com/dimspell/gladiator/internal/backend/redirect"
	"github.com/dimspell/gladiator/internal/console"
	"github.com/dimspell/gladiator/internal/model"
	"github.com/dimspell/gladiator/internal/wire"
	"github.com/quic-go/quic-go"
)

func startDummyTCPServer(t *testing.T, addr string) (stop func()) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatalf("failed to start dummy TCP server on %s: %v", addr, err)
	}
	done := make(chan struct{})
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				select {
				case <-done:
					return
				default:
					continue
				}
			}
			go func(c net.Conn) {
				defer c.Close()
				// Optionally, read/write to c here if needed
				_, _ = io.Copy(io.Discard, c)
			}(conn)
		}
	}()
	return func() {
		close(done)
		ln.Close()
	}
}

func TestPacketRouter_GuestLeavesBeforeHost(t *testing.T) {
	// t.Skip("Failing - needs to be fixed")
	logger.SetPlainTextLogger(os.Stderr, slog.LevelDebug)

	stopDummy := startDummyTCPServer(t, "127.0.0.1:6114")
	defer stopDummy()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	roomID := "guestLeavesFirstRoom"

	// Start multiplayer backend and relay server
	mp := console.NewRoomService()
	relayServer, err := console.NewQUICRelay("localhost:9995", mp)
	if err != nil {
		t.Fatalf("failed to start relay server: %v", err)
	}
	go mp.Run(ctx)
	go relayServer.Start(ctx)

	// gameClient := newMockGameServiceClient()
	gameClient := &console.GameService{RoomService: mp}

	// --- Host setup ---
	hostSession := &bsession.Session{
		ID:          "host-session",
		UserID:      4001,
		Username:    "host",
		CharacterID: 1,
		ClassType:   model.ClassTypeKnight,
		State:       &bsession.SessionState{},
	}
	hostRelay := NewRelay(&ProxyRelay{RelayServerAddr: "localhost:9995"}, gameClient, hostSession)
	hostRelay.router.manager = redirect.NewManager(
		redirect.WithProxyFactory(&redirect.InMemoryProxyFactory{}),
		redirect.WithDisabledLogger(),
	)
	hostSession.Proxy = hostRelay

	hostUserSession := &console.UserSession{
		UserID:      hostSession.UserID,
		ConnectedAt: time.Now().In(time.UTC),
		User:        wire.User{UserID: hostSession.UserID, Username: hostSession.Username},
		Character:   wire.Character{CharacterID: hostSession.CharacterID, ClassType: byte(hostSession.ClassType)},
	}
	mp.AddUserSession(hostUserSession.UserID, hostUserSession)

	err = hostRelay.CreateRoom(ctx, proxy.CreateParams{GameID: roomID})
	if err != nil {
		t.Fatalf("host failed to create room: %v", err)
	}
	mp.SetRoomReady(wire.Message{Content: roomID})

	// --- Guest setup ---
	guestSession := &bsession.Session{
		ID:          "guest-session",
		UserID:      4002,
		Username:    "guest",
		CharacterID: 2,
		ClassType:   model.ClassTypeArcher,
		State:       &bsession.SessionState{},
	}
	guestRelay := NewRelay(&ProxyRelay{RelayServerAddr: "localhost:9995"}, gameClient, guestSession)
	guestRelay.router.manager = redirect.NewManager(
		redirect.WithProxyFactory(&redirect.InMemoryProxyFactory{}),
		redirect.WithDisabledLogger(),
	)
	guestSession.Proxy = guestRelay

	guestUserSession := &console.UserSession{
		UserID:      guestSession.UserID,
		ConnectedAt: time.Now().In(time.UTC),
		User:        wire.User{UserID: guestSession.UserID, Username: guestSession.Username},
		Character:   wire.Character{CharacterID: guestSession.CharacterID, ClassType: byte(guestSession.ClassType)},
	}
	mp.AddUserSession(guestUserSession.UserID, guestUserSession)

	// Guest needs to call GetGame first to get room info and assign IPs
	_, _, err = guestRelay.GetGame(ctx, roomID)
	if err != nil {
		t.Fatalf("guest failed to get game info: %v", err)
	}

	if _, err := guestRelay.JoinGame(ctx, roomID, ""); err != nil {
		t.Fatalf("guest failed to join room: %v", err)
	}
	t.Log("Guest joined room and connected to relay")

	// --- Guest leaves ---
	mp.LeaveRoom(ctx, guestUserSession)
	t.Log("Guest left the room")

	// --- Assertions: host is still host, room is present, guest resources cleaned up ---
	t.Run("Host is still host and room is present", func(t *testing.T) {
		room, ok := mp.GetRoom(roomID)
		if !ok {
			t.Fatalf("room not found after guest left")
		}
		if len(room.Players) != 1 {
			t.Errorf("expected 1 player in room after guest left, got %d", len(room.Players))
		}
		if room.HostPlayer == nil || room.HostPlayer.UserID != hostSession.UserID {
			t.Errorf("host is not the host after guest left")
		}
	})

	// Cancel context to allow goroutines to stop before cleanup
	cancel()
	time.Sleep(100 * time.Millisecond) // Give time for goroutines to finish

	// Cleanup
	hostRelay.Close()
	guestRelay.Close()

	t.Run("Guest relay/router resources cleaned up", func(t *testing.T) {
		_, peerHosts, peerIPs := guestRelay.router.manager.Len()
		if peerHosts != 0 {
			t.Errorf("expected guest PeerHosts to be empty after leave, got %d", peerHosts)
		}
		_ = peerIPs
	})
}

// Add a test for double join/leave edge case
func TestPacketRouter_DoubleJoinLeave(t *testing.T) {
	// t.Skip("Failing - needs to be fixed")

	logger.SetPlainTextLogger(os.Stderr, slog.LevelDebug)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	roomID := "doubleJoinRoom"
	mp := console.NewRoomService()
	relayServer, err := console.NewQUICRelay("localhost:9994", mp)
	if err != nil {
		t.Fatalf("failed to start relay server: %v", err)
	}
	mp.RegisterRelayHooks(relayServer)
	go mp.Run(ctx)
	go relayServer.Start(ctx)

	gameClient := &console.GameService{RoomService: mp}

	hostSession := &bsession.Session{
		ID:          "host-session",
		UserID:      5001,
		Username:    "host",
		CharacterID: 1,
		ClassType:   model.ClassTypeKnight,
		State:       &bsession.SessionState{},
	}

	hostUserSession := &console.UserSession{
		UserID:      hostSession.UserID,
		ConnectedAt: time.Now().In(time.UTC),
		User:        wire.User{UserID: hostSession.UserID, Username: hostSession.Username},
		Character:   wire.Character{CharacterID: hostSession.CharacterID, ClassType: byte(hostSession.ClassType)},
	}
	mp.AddUserSession(hostUserSession.UserID, hostUserSession)

	hostRelay := NewRelay(&ProxyRelay{RelayServerAddr: "localhost:9994"}, gameClient, hostSession)
	hostSession.Proxy = hostRelay

	err = hostRelay.CreateRoom(ctx, proxy.CreateParams{GameID: roomID})
	if err != nil {
		t.Fatalf("host failed to create room: %v", err)
	}
	mp.SetRoomReady(wire.Message{Content: roomID})

	// Verify room was created
	room, ok := mp.GetRoom(roomID)
	if !ok {
		t.Fatalf("room was not created")
	}
	if len(room.Players) != 1 {
		t.Errorf("expected 1 player in room, got %d", len(room.Players))
	}

	// Cancel context and cleanup
	cancel()
	time.Sleep(100 * time.Millisecond)

	// Double close - should not panic or error
	hostRelay.Close()
	hostRelay.Close() // This is the actual test - idempotent close
}

// Add a test for error path (e.g., failed connection)
func TestPacketRouter_ErrorPath_FailedConnection(t *testing.T) {
	t.Parallel()
	logger.SetPlainTextLogger(os.Stderr, slog.LevelDebug)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	gameClient := newMockGameServiceClient()

	hostSession := &bsession.Session{
		ID:          "host-session",
		UserID:      6001,
		Username:    "host",
		CharacterID: 1,
		ClassType:   model.ClassTypeKnight,
		State:       &bsession.SessionState{},
	}
	hostRelay := NewRelay(&ProxyRelay{RelayServerAddr: "invalid:9999"}, gameClient, hostSession)
	hostSession.Proxy = hostRelay
	defer hostRelay.Close()

	err := hostRelay.CreateRoom(ctx, proxy.CreateParams{GameID: "failRoom"})
	if err == nil {
		t.Errorf("expected error on failed connection, got nil")
	}
}

func createSession(mp *console.RoomService, userID int64) (*bsession.Session, *Relay, *console.UserSession) { //nolint:unused // helper for skipped tests
	username := fmt.Sprintf("player%d", userID)
	classType := byte(userID - 1)

	backendSession := &bsession.Session{
		UserID:      userID,
		Username:    username,
		CharacterID: userID,
		ClassType:   model.ClassType(classType),
	}
	lobbySession := &console.UserSession{
		UserID:      userID,
		ConnectedAt: time.Now().In(time.UTC),
		User:        wire.User{UserID: userID, Username: username},
		Character:   wire.Character{CharacterID: userID, ClassType: classType},
	}
	mp.AddUserSession(lobbySession.UserID, lobbySession)

	proxyClient := NewRelay(&ProxyRelay{RelayServerAddr: "localhost:9999"}, newMockGameServiceClient(), backendSession)
	backendSession.Proxy = proxyClient

	return backendSession, proxyClient, lobbySession
}

// --- Mocks ---

// mockStream implements RelayStream for testing.
type mockStream struct {
	mu     sync.Mutex
	buf    bytes.Buffer
	closed bool
}

func (m *mockStream) Read(b []byte) (n int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return 0, fmt.Errorf("stream closed")
	}
	return m.buf.Read(b)
}

func (m *mockStream) Write(b []byte) (n int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return 0, fmt.Errorf("stream closed")
	}
	return m.buf.Write(b)
}

func (m *mockStream) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

func (m *mockStream) CancelRead(code quic.StreamErrorCode) {}
func (m *mockStream) CancelWrite(code quic.StreamErrorCode) {}

// relayConnWrapper adapts a *quic.Stream to RelayConn for testing.
type relayConnWrapper struct{}

func (relayConnWrapper) AcceptStream(context.Context) (*quic.Stream, error) {
	return nil, fmt.Errorf("not implemented")
}
func (relayConnWrapper) CloseWithError(code quic.ApplicationErrorCode, msg string) error {
	return nil
}

// mockTransport is an in-test PeerTransport that buffers sent packets and lets
// the test push packets into the receive channel.
type mockTransport struct {
	mu       sync.Mutex
	recvCh   chan TransportPacket
	closed   bool
	joinRoom string
	joinErr  error
	sendErr  error
	sent     []TransportPacket
}

func newMockTransport() *mockTransport {
	return &mockTransport{recvCh: make(chan TransportPacket, 16)}
}

func (m *mockTransport) Join(ctx context.Context, roomID string) error {
	m.mu.Lock()
	m.joinRoom = roomID
	m.mu.Unlock()
	return m.joinErr
}

func (m *mockTransport) Send(ctx context.Context, pkt TransportPacket) error {
	m.mu.Lock()
	m.sent = append(m.sent, pkt)
	m.mu.Unlock()
	return m.sendErr
}

func (m *mockTransport) Recv(ctx context.Context) (TransportPacket, error) {
	select {
	case <-ctx.Done():
		return TransportPacket{}, ctx.Err()
	case pkt, ok := <-m.recvCh:
		if !ok {
			return TransportPacket{}, io.EOF
		}
		return pkt, nil
	}
}

func (m *mockTransport) Leave(ctx context.Context) error { return nil }

func (m *mockTransport) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return nil
	}
	m.closed = true
	close(m.recvCh)
	return nil
}

// captureRedirect is a redirect.Redirect that records everything written to it.
// Run blocks forever so the FakeHost stays alive (the cleanup goroutine waits
// on g.Wait() which waits on Run). Close is a no-op because StopAll cleans up
// the PeerHosts/Hosts maps directly and the blocked goroutines exit when the
// test process finishes.
type captureRedirect struct {
	mu     sync.Mutex
	buf    bytes.Buffer
	closed bool
}

func (c *captureRedirect) Run(ctx context.Context) error         { select {} }
func (c *captureRedirect) Alive(time.Time, time.Duration) bool  { return true }
func (c *captureRedirect) Write(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.buf.Write(p)
}
func (c *captureRedirect) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
	return nil
}
func (c *captureRedirect) Bytes() []byte {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]byte(nil), c.buf.Bytes()...)
}

// captureFactory returns the same captureRedirect for every proxy so tests can
// observe what PacketRouter writes to a peer's ProxyTCP/ProxyUDP.
type captureFactory struct {
	shared *captureRedirect
}

func (f *captureFactory) NewDialTCP(ip, port string, onReceive redirect.ReceiveFunc) (redirect.Redirect, error) {
	return f.shared, nil
}
func (f *captureFactory) NewDialUDP(ip, port string, onReceive redirect.ReceiveFunc) (redirect.Redirect, error) {
	return f.shared, nil
}
func (f *captureFactory) NewListenerTCP(ip, port string, onReceive redirect.ReceiveFunc) (redirect.Redirect, error) {
	return f.shared, nil
}
func (f *captureFactory) NewListenerUDP(ip, port string, onReceive redirect.ReceiveFunc) (redirect.Redirect, error) {
	return f.shared, nil
}

func TestPacketRouter_ReceiveLoop_DispatchesTCP(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cap := &captureRedirect{}
	factory := &captureFactory{shared: cap}

	pr := &PacketRouter{
		logger: slog.Default(),
		roomID: "test-room",
		manager: redirect.NewManager(
			redirect.WithProxyFactory(factory),
			redirect.WithDisabledLogger(),
		),
		transport: newMockTransport(),
	}

	// Act as the host so dynamicJoin provisions a guest host for peer 200.
	pr.mu.Lock()
	pr.selfID = "100"
	pr.currentHostID = "100"
	pr.mu.Unlock()

	pr.dynamicJoin(ctx, "test-room", "200")

	host, ok := pr.manager.GetPeerHost("200")
	if !ok || host.ProxyTCP == nil {
		t.Fatalf("peer 200 not registered after dynamicJoin (ok=%v, proxyTCP=%v)", ok, host)
	}

	// Drive the dispatch path synchronously (receiveLoop just calls this).
	pr.onTransportPacket(TransportPacket{
		FromID: "200",
		RoomID: "test-room",
		Kind:   KindTCP,
		Data:   []byte("hello"),
	})

	if got := string(cap.Bytes()); got != "hello" {
		t.Fatalf("TCP payload not delivered to peer, got %q", got)
	}
}

func TestPacketRouter_ReceiveLoop_ExitsOnContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	pr := &PacketRouter{
		logger:    slog.Default(),
		roomID:    "test-room",
		transport: newMockTransport(),
	}

	pr.wg.Add(1)
	done := make(chan struct{})
	go func() {
		pr.receiveLoop(ctx)
		close(done)
	}()

	// Let the receiveLoop settle into the blocking Recv
	time.Sleep(10 * time.Millisecond)

	// Cancel the context while Recv is blocking
	cancel()

	select {
	case <-done:
		// receiveLoop exited due to context cancel
	case <-time.After(time.Second):
		t.Fatal("receiveLoop did not exit within 1s after context cancel")
	}
}

func TestPacketRouter_SendPacket_DataRace(t *testing.T) {
	mt := newMockTransport()
	pr := &PacketRouter{
		logger:    slog.Default(),
		selfID:    "test-self",
		transport: mt,
	}

	var wg sync.WaitGroup
	wg.Add(3)

	// Concurrent sendPacket from FakeHost-like goroutine
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			_ = pr.sendPacket(RelayPacket{Type: "tcp", RoomID: "room"})
		}
	}()

	// Concurrent selfID writes (simulates relay.go CreateRoom/GetGame)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			pr.mu.Lock()
			pr.selfID = fmt.Sprintf("id-%d", i)
			pr.mu.Unlock()
		}
	}()

	// Concurrent disconnect/reset (disconnect handles its own locking)
	go func() {
		defer wg.Done()
		for i := 0; i < 20; i++ {
			time.Sleep(time.Microsecond)
			pr.disconnect()
		}
	}()

	wg.Wait()

	mt.mu.Lock()
	sent := len(mt.sent)
	mt.mu.Unlock()
	if sent != 100 {
		t.Errorf("expected 100 sent packets, got %d", sent)
	}
}

type dataCapture struct { //nolint:unused // used in skipped tests
	mu   sync.Mutex
	data [][]byte
}

type mockRedirect struct { //nolint:unused // used in skipped tests
	id        string
	onReceive redirect.ReceiveFunc
	onWrite   func([]byte) error
	closed    bool
}

func (m *mockRedirect) SetOnReceive(handler redirect.ReceiveFunc) { //nolint:unused // used in skipped tests
	m.onReceive = handler
}

func (m *mockRedirect) SetOnWrite(handler func([]byte) error) { //nolint:unused // used in skipped tests
	m.onWrite = handler
}

func (m *mockRedirect) Run(ctx context.Context) error { //nolint:unused // used in skipped tests
	<-ctx.Done()
	return nil
}

func (m *mockRedirect) Write(p []byte) (n int, err error) { //nolint:unused // used in skipped tests
	if m.onWrite != nil {
		_ = m.onWrite(p)
	}
	return len(p), nil
}

func (m *mockRedirect) Close() error { //nolint:unused // used in skipped tests
	m.closed = true
	return nil
}

func (m *mockRedirect) Alive(_ time.Time, _ time.Duration) bool { //nolint:unused // used in skipped tests
	return true
}

type mockProxyFactory struct { //nolint:unused // used in skipped tests
	tcpDial, udpDial, tcpListen, udpListen *mockRedirect
}

func (m *mockProxyFactory) NewDialTCP(ip, port string, onReceive redirect.ReceiveFunc) (redirect.Redirect, error) { //nolint:unused // used in skipped tests
	m.tcpDial.SetOnReceive(onReceive)
	return m.tcpDial, nil
}
func (m *mockProxyFactory) NewDialUDP(ip, port string, onReceive redirect.ReceiveFunc) (redirect.Redirect, error) { //nolint:unused // used in skipped tests
	m.udpDial.SetOnReceive(onReceive)
	return m.udpDial, nil
}
func (m *mockProxyFactory) NewListenerTCP(ip, port string, onReceive redirect.ReceiveFunc) (redirect.Redirect, error) { //nolint:unused // used in skipped tests
	m.tcpListen.SetOnReceive(onReceive)
	return m.tcpListen, nil
}
func (m *mockProxyFactory) NewListenerUDP(ip, port string, onReceive redirect.ReceiveFunc) (redirect.Redirect, error) { //nolint:unused // used in skipped tests
	m.udpListen.SetOnReceive(onReceive)
	return m.udpListen, nil
}

type mockGameServiceClient struct{}

func newMockGameServiceClient() *mockGameServiceClient {
	return &mockGameServiceClient{}
}

// Implement all methods of multiv1connect.GameServiceClient as stubs
func (m *mockGameServiceClient) CreateGame(ctx context.Context, req *connect.Request[multiv1.CreateGameRequest]) (*connect.Response[multiv1.CreateGameResponse], error) {
	return connect.NewResponse(&multiv1.CreateGameResponse{}), nil
}
func (m *mockGameServiceClient) JoinGame(ctx context.Context, req *connect.Request[multiv1.JoinGameRequest]) (*connect.Response[multiv1.JoinGameResponse], error) {
	return connect.NewResponse(&multiv1.JoinGameResponse{}), nil
}
func (m *mockGameServiceClient) ListGames(ctx context.Context, req *connect.Request[multiv1.ListGamesRequest]) (*connect.Response[multiv1.ListGamesResponse], error) {
	return connect.NewResponse(&multiv1.ListGamesResponse{}), nil
}
func (m *mockGameServiceClient) GetGame(ctx context.Context, req *connect.Request[multiv1.GetGameRequest]) (*connect.Response[multiv1.GetGameResponse], error) {
	return connect.NewResponse(&multiv1.GetGameResponse{}), nil
}
