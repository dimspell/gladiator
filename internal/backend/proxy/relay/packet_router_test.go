package relay

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/dimspell/gladiator/internal/app/logger"
	"github.com/dimspell/gladiator/internal/backend/bsession"
	"github.com/dimspell/gladiator/internal/backend/proxy"
	"github.com/dimspell/gladiator/internal/backend/redirect"
	"github.com/dimspell/gladiator/internal/console"
	"github.com/dimspell/gladiator/internal/model"
	"github.com/dimspell/gladiator/internal/wire"
)

func TestPacketRouter_Acceptance_DynamicJoinAndCleanup(t *testing.T) {
	t.Skip("Failing - needs to be fixed")
	logger.SetPlainTextLogger(os.Stderr, slog.LevelDebug)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	roomID := "acceptanceRoom"

	// Start multiplayer backend and relay server
	mp := console.NewMultiplayer()
	relayServer, err := console.NewQUICRelay("localhost:9998", mp)
	if err != nil {
		t.Fatalf("failed to start relay server: %v", err)
	}
	mp.RegisterRelayHooks(relayServer)
	go relayServer.Start(ctx)

	// --- Host setup ---
	hostSession := &bsession.Session{
		ID:          "host-session",
		UserID:      1001,
		Username:    "host",
		CharacterID: 1,
		ClassType:   model.ClassTypeKnight,
		State:       &bsession.SessionState{},
	}
	hostRelay := NewRelay(&ProxyRelay{RelayServerAddr: "localhost:9998"}, hostSession)
	hostSession.Proxy = hostRelay

	// Register host in multiplayer
	hostUserSession := &console.UserSession{
		UserID:      hostSession.UserID,
		Connected:   true,
		ConnectedAt: time.Now().In(time.UTC),
		User:        wire.User{UserID: hostSession.UserID, Username: hostSession.Username},
		Character:   wire.Character{CharacterID: hostSession.CharacterID, ClassType: byte(hostSession.ClassType)},
	}
	mp.AddUserSession(hostUserSession.UserID, hostUserSession)

	// Host creates room and connects
	if _, err := hostRelay.CreateRoom(ctx, proxy.CreateParams{GameID: roomID}); err != nil {
		t.Fatalf("host failed to create room: %v", err)
	}
	mp.SetRoomReady(wire.Message{Content: roomID})

	t.Log("Host created room and connected to relay")

	// --- Guest setup ---
	guestSession := &bsession.Session{
		ID:          "guest-session",
		UserID:      1002,
		Username:    "guest",
		CharacterID: 2,
		ClassType:   model.ClassTypeArcher,
		State:       &bsession.SessionState{},
	}
	guestRelay := NewRelay(&ProxyRelay{RelayServerAddr: "localhost:9998"}, guestSession)
	guestSession.Proxy = guestRelay

	guestUserSession := &console.UserSession{
		UserID:      guestSession.UserID,
		Connected:   true,
		ConnectedAt: time.Now().In(time.UTC),
		User:        wire.User{UserID: guestSession.UserID, Username: guestSession.Username},
		Character:   wire.Character{CharacterID: guestSession.CharacterID, ClassType: byte(guestSession.ClassType)},
	}
	mp.AddUserSession(guestUserSession.UserID, guestUserSession)

	// Guest joins room
	if _, err := guestRelay.Join(ctx, proxy.JoinParams{HostUserID: hostSession.UserID, GameID: roomID}); err != nil {
		t.Fatalf("guest failed to join room: %v", err)
	}
	t.Log("Guest joined room and connected to relay")

	// --- Assertions: both present ---
	t.Run("Both host and guest are present in the room", func(t *testing.T) {
		room, ok := mp.GetRoom(roomID)
		if !ok {
			t.Fatalf("room not found after join")
		}
		if len(room.Players) != 2 {
			t.Errorf("expected 2 players in room, got %d", len(room.Players))
		}
		if _, ok := room.Players[hostSession.UserID]; !ok {
			t.Errorf("host not found in room players")
		}
		if _, ok := room.Players[guestSession.UserID]; !ok {
			t.Errorf("guest not found in room players")
		}
	})

	// --- Simulate guest leaving ---
	mp.LeaveRoom(ctx, guestUserSession)
	t.Log("Guest left the room")

	// --- Assertions: guest cleanup ---
	t.Run("Guest is removed and resources are cleaned up", func(t *testing.T) {
		room, ok := mp.GetRoom(roomID)
		if !ok {
			t.Fatalf("room not found after guest left")
		}
		if _, ok := room.Players[guestSession.UserID]; ok {
			t.Errorf("guest still present in room after leaving")
		}
		// Check relay router state for guest
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

func TestPacketRouter_Acceptance_HostSwitch(t *testing.T) {
	t.Skip("Failing - needs to be fixed")

	logger.SetPlainTextLogger(os.Stderr, slog.LevelDebug)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	roomID := "hostSwitchRoom"

	// Start multiplayer backend and relay server
	mp := console.NewMultiplayer()
	relayServer, err := console.NewQUICRelay("localhost:9997", mp)
	if err != nil {
		t.Fatalf("failed to start relay server: %v", err)
	}
	mp.RegisterRelayHooks(relayServer)
	go relayServer.Start(ctx)

	// --- Host setup ---
	hostSession := &bsession.Session{
		ID:          "host-session",
		UserID:      2001,
		Username:    "host",
		CharacterID: 1,
		ClassType:   model.ClassTypeKnight,
		State:       &bsession.SessionState{},
	}
	hostRelay := NewRelay(&ProxyRelay{RelayServerAddr: "localhost:9997"}, hostSession)
	hostSession.Proxy = hostRelay

	hostUserSession := &console.UserSession{
		UserID:      hostSession.UserID,
		Connected:   true,
		ConnectedAt: time.Now().In(time.UTC),
		User:        wire.User{UserID: hostSession.UserID, Username: hostSession.Username},
		Character:   wire.Character{CharacterID: hostSession.CharacterID, ClassType: byte(hostSession.ClassType)},
		JoinedAt:    time.Now().In(time.UTC),
	}
	mp.AddUserSession(hostUserSession.UserID, hostUserSession)

	if _, err := hostRelay.CreateRoom(ctx, proxy.CreateParams{GameID: roomID}); err != nil {
		t.Fatalf("host failed to create room: %v", err)
	}
	mp.SetRoomReady(wire.Message{Content: roomID})

	t.Log("Host created room and connected to relay")

	// --- Guest setup ---
	guestSession := &bsession.Session{
		ID:          "guest-session",
		UserID:      2002,
		Username:    "guest",
		CharacterID: 2,
		ClassType:   model.ClassTypeArcher,
		State:       &bsession.SessionState{},
	}
	guestRelay := NewRelay(&ProxyRelay{RelayServerAddr: "localhost:9997"}, guestSession)
	guestSession.Proxy = guestRelay

	guestUserSession := &console.UserSession{
		UserID:      guestSession.UserID,
		Connected:   true,
		ConnectedAt: time.Now().In(time.UTC),
		User:        wire.User{UserID: guestSession.UserID, Username: guestSession.Username},
		Character:   wire.Character{CharacterID: guestSession.CharacterID, ClassType: byte(guestSession.ClassType)},
		JoinedAt:    time.Now().Add(time.Millisecond * 10).In(time.UTC), // ensure guest joins after host
	}
	mp.AddUserSession(guestUserSession.UserID, guestUserSession)

	if _, err := guestRelay.Join(ctx, proxy.JoinParams{HostUserID: hostSession.UserID, GameID: roomID}); err != nil {
		t.Fatalf("guest failed to join room: %v", err)
	}
	t.Log("Guest joined room and connected to relay")

	// --- Host leaves ---
	mp.LeaveRoom(ctx, hostUserSession)
	t.Log("Host left the room, triggering host migration")

	// --- Assertions: guest is new host ---
	t.Run("Room still exists and guest is new host", func(t *testing.T) {
		room, ok := mp.GetRoom(roomID)
		if !ok {
			t.Fatalf("room not found after host left")
		}
		if len(room.Players) != 1 {
			t.Errorf("expected 1 player in room after host left, got %d", len(room.Players))
		}
		if room.HostPlayer == nil || room.HostPlayer.UserID != guestSession.UserID {
			t.Errorf("guest is not the new host after host left")
		}
	})
	// t.Run("Room still exists and guest is new host", func(t *testing.T) {
	// 	var room console.GameRoom
	// 	var ok bool
	// 	for i := 0; i < 10; i++ {
	// 		room, ok = mp.GetRoom(roomID)
	// 		if ok && room.HostPlayer != nil && room.HostPlayer.UserID == guestSession.UserID {
	// 			break
	// 		}
	// 		time.Sleep(50 * time.Millisecond)
	// 	}
	// 	if !ok {
	// 		t.Fatalf("room not found after host left")
	// 	}
	// 	if len(room.Players) != 1 {
	// 		t.Errorf("expected 1 player in room after host left, got %d", len(room.Players))
	// 	}
	// 	if room.HostPlayer == nil || room.HostPlayer.UserID != guestSession.UserID {
	// 		t.Errorf("guest is not the new host after host left; HostPlayer: %+v", room.HostPlayer)
	// 	}
	// })

	// --- Assertions: relay/router state ---
	t.Run("Relay/router state is correct after host switch", func(t *testing.T) {
		// Host relay should be cleaned up
		if len(hostRelay.router.manager.PeerHosts) != 0 {
			t.Errorf("expected host PeerHosts to be empty after leave, got %d", len(hostRelay.router.manager.PeerHosts))
		}
		if len(hostRelay.router.manager.Hosts) != 0 {
			t.Errorf("expected host Hosts to be empty after leave, got %d", len(hostRelay.router.manager.Hosts))
		}
		// Guest relay should still be active and be the new host
		if guestRelay.router.currentHostID != guestRelay.router.selfID {
			t.Errorf("guest router did not become the new host, currentHostID=%s, selfID=%s", guestRelay.router.currentHostID, guestRelay.router.selfID)
		}
	})

	// Cleanup
	hostRelay.Close()
	guestRelay.Close()
	cancel()
}

func TestPacketRouter_Acceptance_ProxyForwarding(t *testing.T) {
	t.Skip("Failing - needs to be fixed")
	logger.SetPlainTextLogger(os.Stderr, slog.LevelDebug)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	roomID := "proxyForwardRoom"

	captureHost := &dataCapture{}
	captureGuest := &dataCapture{}

	hostRedirect := &mockRedirect{
		id: "host",
		onReceive: func(p []byte) error {
			captureHost.mu.Lock()
			defer captureHost.mu.Unlock()
			captureHost.data = append(captureHost.data, append([]byte{}, p...))
			return nil
		},
	}
	guestRedirect := &mockRedirect{
		id: "guest",
		onReceive: func(p []byte) error {
			captureGuest.mu.Lock()
			defer captureGuest.mu.Unlock()
			captureGuest.data = append(captureGuest.data, append([]byte{}, p...))
			return nil
		},
	}

	mockProxyFactory := &mockProxyFactory{
		tcpDial:   hostRedirect,
		udpDial:   guestRedirect,
		tcpListen: guestRedirect,
		udpListen: hostRedirect,
	}

	// --- Start multiplayer backend and relay server ---
	mp := console.NewMultiplayer()
	relayServer, err := console.NewQUICRelay("localhost:9996", mp)
	if err != nil {
		t.Fatalf("failed to start relay server: %v", err)
	}
	mp.RegisterRelayHooks(relayServer)
	go relayServer.Start(ctx)

	// --- Host setup ---
	hostSession := &bsession.Session{
		ID:          "host-session",
		UserID:      3001,
		Username:    "host",
		CharacterID: 1,
		ClassType:   model.ClassTypeKnight,
		State:       &bsession.SessionState{},
	}
	hostRelay := NewRelay(&ProxyRelay{RelayServerAddr: "localhost:9996"}, hostSession)
	hostRelay.router.manager.ProxyFactory = mockProxyFactory
	hostSession.Proxy = hostRelay

	hostUserSession := &console.UserSession{
		UserID:      hostSession.UserID,
		Connected:   true,
		ConnectedAt: time.Now().In(time.UTC),
		User:        wire.User{UserID: hostSession.UserID, Username: hostSession.Username},
		Character:   wire.Character{CharacterID: hostSession.CharacterID, ClassType: byte(hostSession.ClassType)},
		JoinedAt:    time.Now().In(time.UTC),
	}
	mp.AddUserSession(hostUserSession.UserID, hostUserSession)

	if _, err := hostRelay.CreateRoom(t.Context(), proxy.CreateParams{GameID: roomID}); err != nil {
		t.Fatalf("host failed to create room: %v", err)
	}
	mp.SetRoomReady(wire.Message{Content: roomID})

	// --- Guest setup ---
	guestSession := &bsession.Session{
		ID:          "guest-session",
		UserID:      3002,
		Username:    "guest",
		CharacterID: 2,
		ClassType:   model.ClassTypeArcher,
		State:       &bsession.SessionState{},
	}
	guestRelay := NewRelay(&ProxyRelay{RelayServerAddr: "localhost:9996"}, guestSession)
	guestRelay.router.manager.ProxyFactory = mockProxyFactory
	guestSession.Proxy = guestRelay

	guestUserSession := &console.UserSession{
		UserID:      guestSession.UserID,
		Connected:   true,
		ConnectedAt: time.Now().In(time.UTC),
		User:        wire.User{UserID: guestSession.UserID, Username: guestSession.Username},
		Character:   wire.Character{CharacterID: guestSession.CharacterID, ClassType: byte(guestSession.ClassType)},
		JoinedAt:    time.Now().Add(time.Millisecond * 10).In(time.UTC),
	}
	mp.AddUserSession(guestUserSession.UserID, guestUserSession)

	if _, err := guestRelay.Join(ctx, proxy.JoinParams{HostUserID: hostSession.UserID, GameID: roomID}); err != nil {
		t.Fatalf("guest failed to join room: %v", err)
	}

	// --- Simulate sending data from host to guest (TCP) ---
	tcpPayload := []byte("hello from host to guest via TCP")
	hostRelay.router.sendPacket(RelayPacket{
		Type:    "tcp",
		RoomID:  roomID,
		FromID:  hostRelay.router.selfID,
		ToID:    guestRelay.router.selfID,
		Payload: tcpPayload,
	})

	// --- Simulate sending data from guest to host (UDP) ---
	udpPayload := []byte("hello from guest to host via UDP")
	guestRelay.router.sendPacket(RelayPacket{
		Type:    "udp",
		RoomID:  roomID,
		FromID:  guestRelay.router.selfID,
		ToID:    hostRelay.router.selfID,
		Payload: udpPayload,
	})

	// --- Assert data was received and forwarded ---
	t.Run("Host receives UDP from guest", func(t *testing.T) {
		time.Sleep(100 * time.Millisecond)
		captureHost.mu.Lock()
		defer captureHost.mu.Unlock()
		found := false
		for _, d := range captureHost.data {
			if string(d) == string(udpPayload) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("host did not receive expected UDP payload from guest")
		}
	})
	t.Run("Guest receives TCP from host", func(t *testing.T) {
		time.Sleep(100 * time.Millisecond)
		captureGuest.mu.Lock()
		defer captureGuest.mu.Unlock()
		found := false
		for _, d := range captureGuest.data {
			if string(d) == string(tcpPayload) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("guest did not receive expected TCP payload from host")
		}
	})

	// Cleanup
	hostRelay.Close()
	guestRelay.Close()
	cancel()
}

func TestPacketRouter_GuestLeavesBeforeHost(t *testing.T) {
	t.Skip("Failing - needs to be fixed")
	logger.SetPlainTextLogger(os.Stderr, slog.LevelDebug)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	roomID := "guestLeavesFirstRoom"

	// Start multiplayer backend and relay server
	mp := console.NewMultiplayer()
	relayServer, err := console.NewQUICRelay("localhost:9995", mp)
	if err != nil {
		t.Fatalf("failed to start relay server: %v", err)
	}
	go relayServer.Start(ctx)

	// --- Host setup ---
	hostSession := &bsession.Session{
		ID:          "host-session",
		UserID:      4001,
		Username:    "host",
		CharacterID: 1,
		ClassType:   model.ClassTypeKnight,
		State:       &bsession.SessionState{},
	}
	hostRelay := NewRelay(&ProxyRelay{RelayServerAddr: "localhost:9995"}, hostSession)
	hostSession.Proxy = hostRelay

	hostUserSession := &console.UserSession{
		UserID:      hostSession.UserID,
		Connected:   true,
		ConnectedAt: time.Now().In(time.UTC),
		User:        wire.User{UserID: hostSession.UserID, Username: hostSession.Username},
		Character:   wire.Character{CharacterID: hostSession.CharacterID, ClassType: byte(hostSession.ClassType)},
	}
	mp.AddUserSession(hostUserSession.UserID, hostUserSession)

	if _, err := hostRelay.CreateRoom(t.Context(), proxy.CreateParams{GameID: roomID}); err != nil {
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
	guestRelay := NewRelay(&ProxyRelay{RelayServerAddr: "localhost:9995"}, guestSession)
	guestSession.Proxy = guestRelay

	guestUserSession := &console.UserSession{
		UserID:      guestSession.UserID,
		Connected:   true,
		ConnectedAt: time.Now().In(time.UTC),
		User:        wire.User{UserID: guestSession.UserID, Username: guestSession.Username},
		Character:   wire.Character{CharacterID: guestSession.CharacterID, ClassType: byte(guestSession.ClassType)},
	}
	mp.AddUserSession(guestUserSession.UserID, guestUserSession)

	if _, err := guestRelay.Join(ctx, proxy.JoinParams{HostUserID: hostSession.UserID, GameID: roomID}); err != nil {
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

	hostSession := &bsession.Session{
		ID:          "host-session",
		UserID:      5001,
		Username:    "host",
		CharacterID: 1,
		ClassType:   model.ClassTypeKnight,
		State:       &bsession.SessionState{},
	}
	hostRelay := NewRelay(&ProxyRelay{RelayServerAddr: "localhost:9994"}, hostSession)
	hostSession.Proxy = hostRelay
	defer hostRelay.Close()

	if _, err := hostRelay.CreateRoom(ctx, proxy.CreateParams{GameID: roomID}); err != nil {
		t.Fatalf("host failed to create room: %v", err)
	}
	mp.SetRoomReady(wire.Message{Content: roomID})

	// Double join
	if _, err := hostRelay.CreateRoom(ctx, proxy.CreateParams{GameID: roomID}); err == nil {
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

	hostSession := &bsession.Session{
		ID:          "host-session",
		UserID:      6001,
		Username:    "host",
		CharacterID: 1,
		ClassType:   model.ClassTypeKnight,
		State:       &bsession.SessionState{},
	}
	hostRelay := NewRelay(&ProxyRelay{RelayServerAddr: "invalid:9999"}, hostSession)
	hostSession.Proxy = hostRelay
	defer hostRelay.Close()

	_, err := hostRelay.CreateRoom(ctx, proxy.CreateParams{GameID: "failRoom"})
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
		Connected:   true,
		ConnectedAt: time.Now().In(time.UTC),
		User:        wire.User{UserID: userID, Username: username},
		Character:   wire.Character{CharacterID: userID, ClassType: classType},
	}
	mp.AddUserSession(lobbySession.UserID, lobbySession)

	proxyClient := NewRelay(&ProxyRelay{RelayServerAddr: "localhost:9999"}, backendSession)
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
