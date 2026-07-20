package relay

import (
	"context"
	"io"
	"log/slog"
	"net"
	"os"
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
	hostRelay.router.SetManager(redirect.NewManager(
		redirect.WithProxyFactory(&redirect.InMemoryProxyFactory{}),
		redirect.WithDisabledLogger(),
	))
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
	guestRelay.router.SetManager(redirect.NewManager(
		redirect.WithProxyFactory(&redirect.InMemoryProxyFactory{}),
		redirect.WithDisabledLogger(),
	))
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
		_, peerHosts, peerIPs := guestRelay.router.Manager().Len()
		if peerHosts != 0 {
			t.Errorf("expected guest PeerHosts to be empty after leave, got %d", peerHosts)
		}
		_ = peerIPs
	})
}

func TestPacketRouter_DoubleJoinLeave(t *testing.T) {
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

type mockGameServiceClient struct{}

func newMockGameServiceClient() *mockGameServiceClient {
	return &mockGameServiceClient{}
}

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
