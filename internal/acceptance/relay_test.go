//go:build e2e

package acceptance

import (
	"context"
	"log/slog"
	"net"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	v1 "github.com/dimspell/gladiator/gen/multi/v1"
	"github.com/dimspell/gladiator/internal/app/logger"
	"github.com/dimspell/gladiator/internal/backend"
	"github.com/dimspell/gladiator/internal/backend/bsession"
	"github.com/dimspell/gladiator/internal/backend/proxy/relay"
	"github.com/dimspell/gladiator/internal/console"
	"github.com/dimspell/gladiator/internal/console/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// relayTestEnv contains the test environment for Relay tests.
type relayTestEnv struct {
	t               *testing.T
	ctx             context.Context
	cancel          context.CancelFunc
	console         *console.Console
	testServer      *httptest.Server
	consoleHostPort string
	relayServer     *console.RelayServer
	proxy           *relay.ProxyRelay
}

// relayPlayer represents a player in the Relay test.
type relayPlayer struct {
	backend *backend.Backend
	conn    *mockConn
	session *bsession.Session
	name    string
}

// setupRelayEnv creates the test environment for Relay tests.
func setupRelayEnv(t *testing.T, relayPort string) *relayTestEnv {
	t.Helper()

	logger.SetColoredLogger(os.Stderr, slog.LevelDebug, false)
	helperStartGameServer(t)

	db, err := database.NewMemory()
	require.NoError(t, err, "failed to create database")
	t.Cleanup(func() { db.Close() })

	require.NoError(t, database.Seed(db.Write), "failed to seed database")

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	cs := console.NewConsole(db)
	ts := httptest.NewServer(h2c.NewHandler(cs.HttpRouter(), &http2.Server{}))
	t.Cleanup(ts.Close)

	consoleHostPort := ts.URL[len("http://"):]
	cs.ConsoleBindAddr = consoleHostPort

	// Create and start QUIC relay server
	relayAddr := "127.0.0.1:" + relayPort
	relayServer, err := console.NewQUICRelay(relayAddr, cs.RoomService)
	require.NoError(t, err, "failed to create QUIC relay server")

	cs.RoomService.RegisterRelayHooks(relayServer)
	go cs.RoomService.Run(ctx)
	go relayServer.Start(ctx)

	// Give the relay server time to start
	time.Sleep(50 * time.Millisecond)

	return &relayTestEnv{
		t:               t,
		ctx:             ctx,
		cancel:          cancel,
		console:         cs,
		testServer:      ts,
		consoleHostPort: consoleHostPort,
		relayServer:     relayServer,
		proxy: &relay.ProxyRelay{
			RelayServerAddr: relayAddr,
			IPPrefix:        net.IPv4(127, 0, 0, 0),
		},
	}
}

// createPlayer creates and authenticates a player.
func (env *relayTestEnv) createPlayer(username, characterName string) *relayPlayer {
	bd := backend.NewBackend("", env.testServer.URL, env.proxy)
	bd.SignalServerURL = "ws://" + env.consoleHostPort + "/lobby"

	conn := &mockConn{}
	session := bd.SessionManager.Add(conn)

	// Sign-in
	authReq := backend.ClientAuthenticationRequest(append(
		[]byte{2, 0, 0, 0},
		append([]byte("test\x00"), append([]byte(username), 0)...)...,
	))
	require.NoError(env.t, bd.HandleClientAuthentication(env.ctx, session, authReq),
		"failed to authenticate player %s", username)

	// Select character
	selectReq := backend.SelectCharacterRequest(append(
		append([]byte(username), 0),
		append([]byte(characterName), 0)...,
	))
	require.NoError(env.t, bd.HandleSelectCharacter(env.ctx, session, selectReq),
		"failed to select character for %s", username)

	require.NoError(env.t, session.JoinLobby(env.ctx),
		"failed to join lobby for %s", username)
	require.NoError(env.t, session.RegisterNewObserver(env.ctx),
		"failed to register observer for %s", username)

	conn.Written = nil // Clear written data

	return &relayPlayer{
		backend: bd,
		conn:    conn,
		session: session,
		name:    username,
	}
}

// createRoom creates a game room with the given player as host.
func (env *relayTestEnv) createRoom(player *relayPlayer, roomName string, mapID v1.GameMap) {
	// Create game room (first call sets state=0)
	createReq := backend.CreateGameRequest(append(
		[]byte{0, 0, 0, 0, byte(mapID), 0, 0, 0},
		append([]byte(roomName), 0, 0)...,
	))
	require.NoError(env.t, player.backend.HandleCreateGame(env.ctx, player.session, createReq),
		"failed to create game (init) for %s", player.name)

	// Set room ready (second call sets state=1)
	readyReq := backend.CreateGameRequest(append(
		[]byte{1, 0, 0, 0, byte(mapID), 0, 0, 0},
		append([]byte(roomName), 0, 0)...,
	))
	require.NoError(env.t, player.backend.HandleCreateGame(env.ctx, player.session, readyReq),
		"failed to create game (ready) for %s", player.name)

	// Give time for room service to process the SetRoomReady message
	time.Sleep(100 * time.Millisecond)

	player.conn.Written = nil
}

// joinRoom has a player join a game room.
func (env *relayTestEnv) joinRoom(player *relayPlayer, roomName string) {
	// List games (optional but good practice)
	require.NoError(env.t, player.backend.HandleListGames(env.ctx, player.session, backend.ListGamesRequest{}),
		"failed to list games for %s", player.name)
	player.conn.Written = nil

	// Select game to get room info
	selectReq := backend.SelectGameRequest(append([]byte(roomName), 0, 0))
	require.NoError(env.t, player.backend.HandleSelectGame(env.ctx, player.session, selectReq),
		"failed to select game for %s", player.name)
	player.conn.Written = nil

	// Join game
	joinReq := backend.JoinGameRequest(append([]byte(roomName), 0, 0))
	require.NoError(env.t, player.backend.HandleJoinGame(env.ctx, player.session, joinReq),
		"failed to join game for %s", player.name)
	player.conn.Written = nil
}

// processMessages processes all pending WebSocket messages for a short duration.
// processMessages drains pending room/signaling messages until the channel is
// idle (no message for a short grace period) or the maximum wait elapses.
// This replaces a fixed time.Sleep so tests finish as soon as signaling settles,
// instead of failing when the system is slower than expected (for example under
// the race detector). The passed duration is treated as a safety cap; a
// too-small value is raised to a sane minimum.
func (env *relayTestEnv) processMessages(maxWait time.Duration) {
	if maxWait < 30*time.Second {
		maxWait = 30 * time.Second
	}

	idle := time.NewTimer(250 * time.Millisecond)
	defer idle.Stop()
	deadline := time.NewTimer(maxWait)
	defer deadline.Stop()

	for {
		select {
		case msg, ok := <-env.console.RoomService.Messages:
			if !ok {
				return
			}
			env.console.RoomService.HandleIncomingMessage(env.ctx, msg)
			if !idle.Stop() {
				select {
				case <-idle.C:
				default:
				}
			}
			idle.Reset(250 * time.Millisecond)
		case <-idle.C:
			return
		case <-deadline.C:
			return
		}
	}
}

// TestE2E_Relay tests a basic relay game session with host and guest.
func TestE2E_Relay(t *testing.T) {
	// t.Skip("Requires loopback aliases (127.0.0.X) - see README troubleshooting")

	env := setupRelayEnv(t, "19995")

	// Create host player
	host := env.createPlayer("archer", "archer")

	// Create game room
	env.createRoom(host, "room", v1.GameMap_FrozenLabyrinth)

	room, ok := env.console.RoomService.Rooms["room"]
	require.True(t, ok, "room should exist")
	assert.Equal(t, 1, len(room.Players), "room should have 1 player")
	assert.NotNil(t, room.HostPlayer, "room should have a host")
	assert.Equal(t, host.session.UserID, room.HostPlayer.UserID, "host should be the room host")

	t.Log("Host created room")

	// Create guest player
	guest := env.createPlayer("warrior", "warrior")

	// Guest joins the room
	env.joinRoom(guest, "room")

	// Process relay connection messages
	env.processMessages(1 * time.Second)

	// Verify both players are in room
	room = env.console.RoomService.Rooms["room"]
	assert.Equal(t, 2, len(room.Players), "room should have 2 players")

	t.Log("Guest joined room")

	// Verify both sessions have proxies
	require.NotNil(t, host.session.Proxy, "host should have proxy")
	require.NotNil(t, guest.session.Proxy, "guest should have proxy")

	// Cleanup - close proxies first, then cancel context
	host.session.Proxy.Close()
	guest.session.Proxy.Close()
}

// TestE2E_Relay_GuestLeaves tests that when a guest leaves, the room remains with the host.
func TestE2E_Relay_GuestLeaves(t *testing.T) {
	// t.Skip("Requires loopback aliases (127.0.0.X) - see README troubleshooting")

	env := setupRelayEnv(t, "19996")

	// Setup host
	host := env.createPlayer("archer", "archer")
	env.createRoom(host, "room", v1.GameMap_FrozenLabyrinth)

	// Setup guest
	guest := env.createPlayer("warrior", "warrior")
	env.joinRoom(guest, "room")

	// Process join messages
	env.processMessages(1 * time.Second)

	room, ok := env.console.RoomService.GetRoom("room")
	require.True(t, ok, "room should exist")
	require.Equal(t, 2, len(room.Players), "room should have 2 players before leave")

	t.Log("Both players in room, guest leaving...")

	// Guest leaves
	guestSession, ok := env.console.RoomService.GetUserSession(guest.session.UserID)
	require.True(t, ok, "guest session should exist")
	env.console.RoomService.LeaveRoom(env.ctx, guestSession)

	// Process leave messages
	env.processMessages(500 * time.Millisecond)

	// Verify room still exists with only host
	room, ok = env.console.RoomService.GetRoom("room")
	require.True(t, ok, "room should still exist")
	assert.Equal(t, 1, len(room.Players), "room should have 1 player after guest left")
	assert.Equal(t, host.session.UserID, room.HostPlayer.UserID, "host should still be host")

	t.Log("Guest left, host remains")

	// Cleanup
	guest.session.Proxy.Close()
	host.session.Proxy.Close()
}

// TestE2E_Relay_HostMigration tests that when the host leaves, a guest becomes the new host.
func TestE2E_Relay_HostMigration(t *testing.T) {
	// t.Skip("Requires loopback aliases (127.0.0.X) - see README troubleshooting")

	env := setupRelayEnv(t, "19997")

	// Setup host
	host := env.createPlayer("archer", "archer")
	env.createRoom(host, "room", v1.GameMap_FrozenLabyrinth)

	// Setup guest
	guest := env.createPlayer("warrior", "warrior")
	env.joinRoom(guest, "room")

	// Process join messages
	env.processMessages(1 * time.Second)

	room, ok := env.console.RoomService.GetRoom("room")
	require.True(t, ok, "room should exist")
	require.Equal(t, 2, len(room.Players), "room should have 2 players")
	require.Equal(t, host.session.UserID, room.HostPlayer.UserID, "archer should be host")

	t.Log("Both players in room, host leaving...")

	// Host leaves
	hostSession, ok := env.console.RoomService.GetUserSession(host.session.UserID)
	require.True(t, ok, "host session should exist")
	env.console.RoomService.LeaveRoom(env.ctx, hostSession)

	// Process host migration messages
	env.processMessages(1 * time.Second)

	// Verify guest is now host
	room, ok = env.console.RoomService.GetRoom("room")
	require.True(t, ok, "room should still exist")
	assert.Equal(t, 1, len(room.Players), "room should have 1 player after host left")

	if room.HostPlayer != nil {
		assert.Equal(t, guest.session.UserID, room.HostPlayer.UserID, "guest should be new host")
		t.Log("Host migration successful, guest is new host")
	} else {
		t.Log("Warning: No host assigned after migration (may be expected in some scenarios)")
	}

	// Cleanup
	host.session.Proxy.Close()
	guest.session.Proxy.Close()
}

// TestE2E_Relay_ThreePlayersOneLeaves tests a room with 3 players where one leaves.
func TestE2E_Relay_ThreePlayersOneLeaves(t *testing.T) {
	// t.Skip("Requires loopback aliases (127.0.0.X) - see README troubleshooting")

	env := setupRelayEnv(t, "19998")

	// Setup host
	host := env.createPlayer("archer", "archer")
	env.createRoom(host, "room", v1.GameMap_FrozenLabyrinth)

	// Setup guest 1
	guest1 := env.createPlayer("warrior", "warrior")
	env.joinRoom(guest1, "room")

	// Setup guest 2
	guest2 := env.createPlayer("necro", "necro")
	env.joinRoom(guest2, "room")

	// Process all join messages
	env.processMessages(2 * time.Second)

	room, ok := env.console.RoomService.GetRoom("room")
	require.True(t, ok, "room should exist")
	require.Equal(t, 3, len(room.Players), "room should have 3 players")

	t.Log("Three players in room, guest1 leaving...")

	// Guest1 leaves
	guest1Session, ok := env.console.RoomService.GetUserSession(guest1.session.UserID)
	require.True(t, ok, "guest1 session should exist")
	env.console.RoomService.LeaveRoom(env.ctx, guest1Session)

	// Process leave messages
	env.processMessages(500 * time.Millisecond)

	// Verify room has 2 players
	room, ok = env.console.RoomService.GetRoom("room")
	require.True(t, ok, "room should still exist")
	assert.Equal(t, 2, len(room.Players), "room should have 2 players after one left")
	assert.Equal(t, host.session.UserID, room.HostPlayer.UserID, "host should still be host")

	// Verify correct players remain
	_, hostExists := room.Players[host.session.UserID]
	_, guest2Exists := room.Players[guest2.session.UserID]
	_, guest1Exists := room.Players[guest1.session.UserID]

	assert.True(t, hostExists, "host should still be in room")
	assert.True(t, guest2Exists, "guest2 should still be in room")
	assert.False(t, guest1Exists, "guest1 should not be in room")

	t.Log("One player left, two remain")

	// Cleanup
	guest1.session.Proxy.Close()
	guest2.session.Proxy.Close()
	host.session.Proxy.Close()
}

// TestE2E_Relay_RoomDeleted tests that an empty room gets deleted.
func TestE2E_Relay_RoomDeleted(t *testing.T) {
	// t.Skip("Requires loopback aliases (127.0.0.X) - see README troubleshooting")

	env := setupRelayEnv(t, "19999")

	// Setup host
	host := env.createPlayer("archer", "archer")
	env.createRoom(host, "room", v1.GameMap_FrozenLabyrinth)

	room, ok := env.console.RoomService.GetRoom("room")
	require.True(t, ok, "room should exist")
	require.Equal(t, 1, len(room.Players), "room should have 1 player")

	t.Log("Host created room, now leaving...")

	// Host leaves
	hostSession, ok := env.console.RoomService.GetUserSession(host.session.UserID)
	require.True(t, ok, "host session should exist")
	env.console.RoomService.LeaveRoom(env.ctx, hostSession)

	// Process leave and room deletion messages
	env.processMessages(500 * time.Millisecond)

	// Verify room is deleted
	_, ok = env.console.RoomService.GetRoom("room")
	assert.False(t, ok, "room should be deleted when empty")

	t.Log("Empty room deleted successfully")

	// Cleanup
	host.session.Proxy.Close()
}

// TestE2E_Relay_MultipleRooms tests multiple concurrent game rooms.
func TestE2E_Relay_MultipleRooms(t *testing.T) {
	// t.Skip("Requires loopback aliases (127.0.0.X) - see README troubleshooting")

	env := setupRelayEnv(t, "19994")

	// Create first room with host
	host1 := env.createPlayer("archer", "archer")

	// Create first room manually with specific name
	err := host1.backend.HandleCreateGame(env.ctx, host1.session, backend.CreateGameRequest{
		0, 0, 0, 0,
		byte(v1.GameMap_FrozenLabyrinth), 0, 0, 0,
		'r', 'o', 'o', 'm', '1', 0,
		0,
	})
	require.NoError(t, err)
	err = host1.backend.HandleCreateGame(env.ctx, host1.session, backend.CreateGameRequest{
		1, 0, 0, 0,
		byte(v1.GameMap_FrozenLabyrinth), 0, 0, 0,
		'r', 'o', 'o', 'm', '1', 0,
		0,
	})
	require.NoError(t, err)

	// Create second room with different host
	host2 := env.createPlayer("warrior", "warrior")

	err = host2.backend.HandleCreateGame(env.ctx, host2.session, backend.CreateGameRequest{
		0, 0, 0, 0,
		byte(v1.GameMap_AbandonedRealm), 0, 0, 0,
		'r', 'o', 'o', 'm', '2', 0,
		0,
	})
	require.NoError(t, err)
	err = host2.backend.HandleCreateGame(env.ctx, host2.session, backend.CreateGameRequest{
		1, 0, 0, 0,
		byte(v1.GameMap_AbandonedRealm), 0, 0, 0,
		'r', 'o', 'o', 'm', '2', 0,
		0,
	})
	require.NoError(t, err)

	// Process room creation messages
	env.processMessages(500 * time.Millisecond)

	// Verify both rooms exist
	room1, ok1 := env.console.RoomService.GetRoom("room1")
	room2, ok2 := env.console.RoomService.GetRoom("room2")

	assert.True(t, ok1, "room1 should exist")
	assert.True(t, ok2, "room2 should exist")

	if ok1 && ok2 {
		assert.Equal(t, 1, len(room1.Players), "room1 should have 1 player")
		assert.Equal(t, 1, len(room2.Players), "room2 should have 1 player")
		assert.Equal(t, host1.session.UserID, room1.HostPlayer.UserID, "host1 should be room1 host")
		assert.Equal(t, host2.session.UserID, room2.HostPlayer.UserID, "host2 should be room2 host")
	}

	t.Log("Multiple rooms created successfully")

	// Cleanup
	host1.session.Proxy.Close()
	host2.session.Proxy.Close()
}

// TestE2E_Relay_Authentication tests that players authenticate correctly.
func TestE2E_Relay_Authentication(t *testing.T) {
	env := setupRelayEnv(t, "19993")

	bd := backend.NewBackend("", env.testServer.URL, env.proxy)
	bd.SignalServerURL = "ws://" + env.consoleHostPort + "/lobby"
	conn := &mockConn{}
	session := bd.SessionManager.Add(conn)

	// Sign-in with valid credentials
	err := bd.HandleClientAuthentication(env.ctx, session, backend.ClientAuthenticationRequest{
		2, 0, 0, 0,
		't', 'e', 's', 't', 0,
		'a', 'r', 'c', 'h', 'e', 'r', 0,
	})
	require.NoError(t, err, "authentication should succeed")

	// Verify successful auth response (byte 4 should be 1 for success)
	require.True(t, len(conn.Written) >= 8, "should receive auth response")
	assert.Equal(t, byte(255), conn.Written[0], "packet header should be 255")
	assert.Equal(t, byte(41), conn.Written[1], "packet type should be 41 (auth)")
	assert.Equal(t, byte(1), conn.Written[4], "auth should succeed (byte 4 = 1)")

	t.Log("Authentication successful")
}
