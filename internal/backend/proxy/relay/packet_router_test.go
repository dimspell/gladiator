package relay

import (
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
				io.Copy(io.Discard, c)
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
	mp := console.NewMultiplayer()
	relayServer, err := console.NewQUICRelay("localhost:9995", mp)
	if err != nil {
		t.Fatalf("failed to start relay server: %v", err)
	}
	go mp.Run(ctx)
	go relayServer.Start(ctx)

	// gameClient := newMockGameServiceClient()
	gameClient := &console.GameServiceServer{Multiplayer: mp}

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
	guestSession.Proxy = guestRelay

	guestUserSession := &console.UserSession{
		UserID:      guestSession.UserID,
		ConnectedAt: time.Now().In(time.UTC),
		User:        wire.User{UserID: guestSession.UserID, Username: guestSession.Username},
		Character:   wire.Character{CharacterID: guestSession.CharacterID, ClassType: byte(guestSession.ClassType)},
	}
	mp.AddUserSession(guestUserSession.UserID, guestUserSession)

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
	t.Run("Guest relay/router resources cleaned up", func(t *testing.T) {
		if len(guestRelay.router.manager.PeerHosts) != 0 {
			t.Errorf("expected guest PeerHosts to be empty after leave, got %d", len(guestRelay.router.manager.PeerHosts))
		}
		if len(guestRelay.router.manager.Hosts) != 0 {
			t.Errorf("expected guest Hosts to be empty after leave, got %d", len(guestRelay.router.manager.Hosts))
		}
	})

	// Cleanup
	hostRelay.Close()
	guestRelay.Close()
	cancel()
}

// Add a test for double join/leave edge case
func TestPacketRouter_DoubleJoinLeave(t *testing.T) {
	t.Skip("Failing - needs to be fixed")

	logger.SetPlainTextLogger(os.Stderr, slog.LevelDebug)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	roomID := "doubleJoinRoom"
	mp := console.NewMultiplayer()
	relayServer, err := console.NewQUICRelay("localhost:9994", mp)
	if err != nil {
		t.Fatalf("failed to start relay server: %v", err)
	}
	mp.RegisterRelayHooks(relayServer)
	go relayServer.Start(ctx)

	gameClient := newMockGameServiceClient()

	hostSession := &bsession.Session{
		ID:          "host-session",
		UserID:      5001,
		Username:    "host",
		CharacterID: 1,
		ClassType:   model.ClassTypeKnight,
		State:       &bsession.SessionState{},
	}
	hostRelay := NewRelay(&ProxyRelay{RelayServerAddr: "localhost:9994"}, gameClient, hostSession)
	hostSession.Proxy = hostRelay
	defer hostRelay.Close()

	err = hostRelay.CreateRoom(ctx, proxy.CreateParams{GameID: roomID})
	if err != nil {
		t.Fatalf("host failed to create room: %v", err)
	}
	mp.SetRoomReady(wire.Message{Content: roomID})

	// Double join
	err = hostRelay.CreateRoom(ctx, proxy.CreateParams{GameID: roomID})
	if err == nil {
		t.Errorf("expected error on double create room, got nil")
	}

	// Double leave
	hostRelay.Close()
	hostRelay.Close() // Should not panic or error
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

func createSession(mp *console.Multiplayer, userID int64) (*bsession.Session, *Relay, *console.UserSession) {
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

type dataCapture struct {
	mu   sync.Mutex
	data [][]byte
}

type mockRedirect struct {
	id        string
	onReceive redirect.ReceiveFunc
	onWrite   func([]byte) error
	closed    bool
}

func (m *mockRedirect) SetOnReceive(handler redirect.ReceiveFunc) {
	m.onReceive = handler
}

func (m *mockRedirect) SetOnWrite(handler func([]byte) error) {
	m.onWrite = handler
}

func (m *mockRedirect) Run(ctx context.Context) error {
	<-ctx.Done()
	return nil
}

func (m *mockRedirect) Write(p []byte) (n int, err error) {
	if m.onWrite != nil {
		_ = m.onWrite(p)
	}
	return len(p), nil
}

func (m *mockRedirect) Close() error {
	m.closed = true
	return nil
}

func (m *mockRedirect) Alive(_ time.Time, _ time.Duration) bool {
	return true
}

type mockProxyFactory struct {
	tcpDial, udpDial, tcpListen, udpListen *mockRedirect
}

func (m *mockProxyFactory) NewDialTCP(ip, port string, onReceive redirect.ReceiveFunc) (redirect.Redirect, error) {
	m.tcpDial.SetOnReceive(onReceive)
	return m.tcpDial, nil
}
func (m *mockProxyFactory) NewDialUDP(ip, port string, onReceive redirect.ReceiveFunc) (redirect.Redirect, error) {
	m.udpDial.SetOnReceive(onReceive)
	return m.udpDial, nil
}
func (m *mockProxyFactory) NewListenerTCP(ip, port string, onReceive redirect.ReceiveFunc) (redirect.Redirect, error) {
	m.tcpListen.SetOnReceive(onReceive)
	return m.tcpListen, nil
}
func (m *mockProxyFactory) NewListenerUDP(ip, port string, onReceive redirect.ReceiveFunc) (redirect.Redirect, error) {
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
