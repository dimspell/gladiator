package acceptance

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	v1 "github.com/dimspell/gladiator/gen/multi/v1"
	"github.com/dimspell/gladiator/internal/app/logger"
	"github.com/dimspell/gladiator/internal/app/logger/logging"
	"github.com/dimspell/gladiator/internal/backend"
	"github.com/dimspell/gladiator/internal/backend/bsession"
	"github.com/dimspell/gladiator/internal/backend/packet"
	"github.com/dimspell/gladiator/internal/backend/proxy/p2p"
	"github.com/dimspell/gladiator/internal/console"
	"github.com/dimspell/gladiator/internal/console/database"
	"github.com/dimspell/gladiator/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestE2E_P2P(t *testing.T) {
	// t.Skip("Requires loopback aliases (127.0.0.X) - see README troubleshooting")

	logger.SetColoredLogger(os.Stderr, slog.LevelDebug, false)

	helperStartGameServer(t)

	proxy := &p2p.ProxyP2P{}

	// redirectFunc := redirect.New

	db, err := database.NewMemory()
	if err != nil {
		t.Fatalf("failed to create database: %v", err)
		return
	}
	defer db.Close()

	if err := database.Seed(db.Write); err != nil {
		t.Fatalf("failed to seed database: %v", err)
		return
	}

	// ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cs := console.NewConsole(db)
	ts := httptest.NewServer(h2c.NewHandler(cs.HttpRouter(), &http2.Server{}))
	defer ts.Close()

	// go cs.RoomService.Run(ctx)

	// Extract host:port from test server URL and set console address
	consoleHostPort := ts.URL[len("http://"):]
	cs.ConsoleBindAddr = consoleHostPort

	// proxy1.NewRedirect = redirectFunc
	bd1 := backend.NewBackend("", ts.URL, proxy, backend.WithHTTPClient(&http.Client{
		Timeout:   30 * time.Second,
		Transport: backend.SharedHttpClient.Transport,
	}))
	bd1.SignalServerURL = "ws://" + consoleHostPort + "/lobby"

	conn1 := &mockConn{}
	session1 := bd1.SessionManager.Add(conn1)

	// FIXME: Set IPRing in test mode2
	// session1.IpRing.IsTesting = true
	// session1.IpRing.UdpPortPrefix = 1300
	// session1.IpRing.TcpPortPrefix = 1400

	// Sign-in
	assert.NoError(t, bd1.HandleClientAuthentication(ctx, session1, backend.ClientAuthenticationRequest{
		2, 0, 0, 0, // Unknown
		't', 'e', 's', 't', 0, // Password
		'a', 'r', 'c', 'h', 'e', 'r', 0, // Username
	}))
	if !bytes.Equal([]byte{255, 41, 8, 0, 1, 0, 0, 0}, conn1.Written) {
		t.Errorf("Not logged in, got: %v", conn1.Written)
		return
	}

	// Select character
	assert.NoError(t, bd1.HandleSelectCharacter(ctx, session1, backend.SelectCharacterRequest{
		'a', 'r', 'c', 'h', 'e', 'r', 0, // User name
		'a', 'r', 'c', 'h', 'e', 'r', 0, // Character name
	}))
	err = session1.JoinLobby(ctx)
	if err != nil {
		t.Errorf("failed to join lobby: %v", err)
		return
	}
	err = session1.RegisterNewObserver(ctx)
	if err != nil {
		t.Errorf("failed to register new observer: %v", err)
		return
	}

	// Create a new game room
	assert.NoError(t, bd1.HandleCreateGame(ctx, session1, backend.CreateGameRequest{
		0, 0, 0, 0, // State
		byte(v1.GameMap_FrozenLabyrinth), 0, 0, 0, // Map ID
		'r', 'o', 'o', 'm', 0, // Game room name
		0, // Password
	}))
	assert.NoError(t, bd1.HandleCreateGame(ctx, session1, backend.CreateGameRequest{
		1, 0, 0, 0, // State
		byte(v1.GameMap_FrozenLabyrinth), 0, 0, 0, // Map ID
		'r', 'o', 'o', 'm', 0, // Game room name
		0, // Password
	}))

	cs.RoomService.HandleIncomingMessage(ctx, <-cs.RoomService.Messages)

	room, ok := cs.RoomService.Rooms["room"]
	if !ok {
		t.Errorf("failed to find room")
		return
	}
	if !room.Ready {
		t.Errorf("failed to create new room - it is unready")
		return
	}
	assert.Equal(t, "room", room.Name)
	assert.Equal(t, session1.UserID, room.CreatedBy.UserID)
	assert.Equal(t, session1.UserID, room.HostPlayer.UserID)
	assert.Equal(t, 1, len(room.Players))
	assert.Equal(t, session1.UserID, room.Players[1].UserID)
	assert.Equal(t, "archer", room.Players[1].User.Username)
	assert.Equal(t, byte(v1.ClassType_Archer), room.Players[1].Character.ClassType)

	// Other user
	bd2 := backend.NewBackend("", ts.URL, proxy)
	bd2.SignalServerURL = "ws://" + consoleHostPort + "/lobby"

	conn2 := &mockConn{}
	session2 := bd2.SessionManager.Add(conn2)

	// FIXME: Set IPRing in test mode
	// session2.IpRing.IsTesting = true
	// session2.IpRing.UdpPortPrefix = 2300
	// session2.IpRing.TcpPortPrefix = 2400

	// Sign-in by player2
	assert.NoError(t, bd2.HandleClientAuthentication(ctx, session2, backend.ClientAuthenticationRequest{
		2, 0, 0, 0, // Unknown
		't', 'e', 's', 't', 0, // Password
		'm', 'a', 'g', 'e', 0, // Username
	}))
	if !bytes.Equal([]byte{255, 41, 8, 0, 1, 0, 0, 0}, conn2.Written) {
		t.Errorf("Not logged in, got: %v", conn2.Written)
		return
	}

	// Select character by player2
	assert.NoError(t, bd2.HandleSelectCharacter(ctx, session2, backend.SelectCharacterRequest{
		'm', 'a', 'g', 'e', 0, // User name
		'm', 'a', 'g', 'e', 0, // Character name
	}))
	err = session2.JoinLobby(ctx)
	if err != nil {
		t.Errorf("failed to join lobby: %v", err)
		return
	}
	err = session2.RegisterNewObserver(ctx)
	if err != nil {
		t.Errorf("failed to register new observer: %v", err)
		return
	}

	// Truncate
	conn2.Written = nil

	// List games
	assert.NoError(t, bd2.HandleListGames(ctx, session2, backend.ListGamesRequest{}))

	// Check if user has received the game list with corresponding payload
	assert.Equal(t, []byte{
		1, 0, 0, 0, // Number of games
		127, 0, 0, 2, // IP address of host (127.0.0.2 for P2P mode)
		'r', 'o', 'o', 'm', 0, // Room name
		0, // Password
	}, findPacket(conn2.Written, packet.ListGames))

	// Truncate
	conn2.Written = nil

	// Select game
	assert.NoError(t, bd2.HandleSelectGame(ctx, session2, backend.SelectGameRequest{
		'r', 'o', 'o', 'm', 0, // Game name
		0, // Password
	}))

	// Check if the game is correct
	assert.Equal(t, []byte{
		byte(v1.GameMap_FrozenLabyrinth), 0, 0, 0, // Map ID
		byte(v1.ClassType_Archer), 0, 0, 0, // Host's character class type
		127, 0, 0, 2, // IP address of host (127.0.0.2 for P2P mode)
		'a', 'r', 'c', 'h', 'e', 'r', 0, // Player name
	}, findPacket(conn2.Written, packet.SelectGame))

	// Truncate
	conn2.Written = nil

	// Join to host
	assert.NoError(t, bd2.HandleJoinGame(ctx, session2, backend.JoinGameRequest{
		'r', 'o', 'o', 'm', 0, // Game name
		0, // Password
	}))

	// Ensure the response is correct
	assert.Equal(t, []byte{
		model.GameStateStarted, 0, // Game state
		byte(v1.ClassType_Archer), 0, 0, 0, // Host's character class type
		127, 0, 0, 2, // IP address of host (127.0.0.2 for P2P mode)
		'a', 'r', 'c', 'h', 'e', 'r', 0, // Player name
	}, findPacket(conn2.Written, packet.JoinGame))

	// Room contains all data
	room, ok = cs.RoomService.Rooms["room"]
	if !ok {
		t.Errorf("failed to find room")
		return
	}
	if !room.Ready {
		t.Errorf("failed to join room - it is unready")
		return
	}
	assert.Equal(t, "room", room.Name)
	assert.Equal(t, session1.UserID, room.CreatedBy.UserID)
	assert.Equal(t, session1.UserID, room.HostPlayer.UserID)
	assert.Equal(t, 2, len(room.Players))
	assert.Equal(t, session1.UserID, room.Players[1].UserID)
	assert.Equal(t, "archer", room.Players[1].User.Username)
	assert.Equal(t, byte(v1.ClassType_Archer), room.Players[1].Character.ClassType)
	assert.Equal(t, session2.UserID, room.Players[2].UserID)
	assert.Equal(t, "mage", room.Players[2].User.Username)
	assert.Equal(t, byte(v1.ClassType_Mage), room.Players[2].Character.ClassType)

	mpSession1, ok := cs.RoomService.GetUserSession(1)
	assert.True(t, ok)
	assert.Equal(t, session1.UserID, mpSession1.UserID)
	assert.Equal(t, "room", mpSession1.GameID)

	mpSession2, ok := cs.RoomService.GetUserSession(2)
	assert.True(t, ok)
	assert.Equal(t, session2.UserID, mpSession2.UserID)
	assert.Equal(t, "room", mpSession2.GameID)

	// Host user has correct data
	assert.Equal(t, int64(1), mpSession1.UserID)
	assert.Equal(t, "archer", mpSession1.User.Username)
	// P2P mode uses WebRTC for connectivity, so IPAddress is empty
	assert.Equal(t, "", mpSession1.IPAddress)

	// Joining user has also the same data
	assert.Equal(t, int64(2), mpSession2.UserID)
	assert.Equal(t, "mage", mpSession2.User.Username)
	// P2P mode uses WebRTC for connectivity, so IPAddress is empty
	assert.Equal(t, "", mpSession2.IPAddress)

	// RTCICECandidate
	// cs.RoomService.HandleIncomingMessage(ctx, <-cs.RoomService.Messages)
	// cs.RoomService.HandleIncomingMessage(ctx, <-cs.RoomService.Messages)
	// cs.RoomService.HandleIncomingMessage(ctx, <-cs.RoomService.Messages)
	//
	// RTCICECandidate
	// cs.RoomService.HandleIncomingMessage(ctx, <-cs.RoomService.Messages)
	// cs.RoomService.HandleIncomingMessage(ctx, <-cs.RoomService.Messages)
	// cs.RoomService.HandleIncomingMessage(ctx, <-cs.RoomService.Messages)

	go func() {
		<-time.After(time.Second * 3)
		close(cs.RoomService.Messages)
	}()
	for message := range cs.RoomService.Messages {
		cs.RoomService.HandleIncomingMessage(ctx, message)
		// t.Error("unhandled message", message)
	}
}

func helperStartGameServer(t testing.TB) {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())

	// Listen for incoming connections.
	tcpListener, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", "6114"))
	if err != nil {
		t.Fatal(err)
	}

	udpAddr, err := net.ResolveUDPAddr("udp", net.JoinHostPort("127.0.0.1", "6113"))
	if err != nil {
		t.Fatal(err)
	}

	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		t.Fatal(err)
	}

	// Listen UDP
	go func() {
		for {
			if ctx.Err() != nil {
				fmt.Println("context err")
				return
			}

			buf := make([]byte, 1024)
			n, _, err := udpConn.ReadFrom(buf)
			if err != nil {
				break
			}

			if buf[0] == '#' {
				resp := append([]byte{27, 0}, buf[1:n]...)
				_, err := udpConn.WriteToUDP(resp, udpAddr)
				if err != nil {
					slog.Debug("Failed to write to UDP", logging.Error(err))
					return
				}
				slog.Debug("UDP response", "response", string(resp))
			}
		}
	}()

	processPackets := func(conn net.Conn) {
		t.Log("Someone has connected over the TCP")

		message := make(chan []byte, 1)

		go func() {
			defer conn.Close()

			for {
				select {
				case <-ctx.Done():
					return
				case msg, ok := <-message:
					if !ok {
						return
					}
					slog.Debug("message received", "msg", string(msg))
					_, _ = conn.Write([]byte{35, 35, 116, 101, 115, 116, 0})
				}
			}
		}()

		for {
			_ = conn.SetDeadline(time.Now().Add(10 * time.Second))

			buf := make([]byte, 1024)
			n, err := conn.Read(buf)
			if err != nil {
				close(message)
				return
			}
			message <- buf[:n]
		}
	}

	go func() {
		for {
			if ctx.Err() != nil {
				return
			}

			// Listen for an incoming connection.
			conn, err := tcpListener.Accept()
			if err != nil {
				continue
			}
			go processPackets(conn)
		}
	}()

	t.Cleanup(func() {
		t.Log("Shutting down the game server")

		cancel()
		udpConn.Close()
		tcpListener.Close()
	})
}

// p2pTestEnv contains the test environment for P2P tests.
type p2pTestEnv struct {
	t               *testing.T
	ctx             context.Context
	cancel          context.CancelFunc
	console         *console.Console
	testServer      *httptest.Server
	consoleHostPort string
	proxy           *p2p.ProxyP2P
}

// p2pPlayer represents a player in the P2P test.
type p2pPlayer struct {
	backend *backend.Backend
	conn    *mockConn
	session *bsession.Session
	name    string
}

// setupP2PEnv creates the test environment for P2P tests.
func setupP2PEnv(t *testing.T) *p2pTestEnv {
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

	return &p2pTestEnv{
		t:               t,
		ctx:             ctx,
		cancel:          cancel,
		console:         cs,
		testServer:      ts,
		consoleHostPort: consoleHostPort,
		proxy:           &p2p.ProxyP2P{},
	}
}

// createPlayer creates and authenticates a player.
func (env *p2pTestEnv) createPlayer(username, characterName string) *p2pPlayer {
	// Under -race the in-process auth server is slow enough to exceed the
	// default 5s SharedHttpClient timeout; use a generous client for the test.
	bd := backend.NewBackend("", env.testServer.URL, env.proxy, backend.WithHTTPClient(&http.Client{
		Timeout:   30 * time.Second,
		Transport: backend.SharedHttpClient.Transport,
	}))
	bd.SignalServerURL = "ws://" + env.consoleHostPort + "/lobby"

	conn := &mockConn{}
	session := bd.SessionManager.Add(conn)

	// Sign-in
	authReq := backend.ClientAuthenticationRequest(append(
		[]byte{2, 0, 0, 0},
		append([]byte("test\x00"), append([]byte(username), 0)...)...,
	))
	require.NoError(env.t, bd.HandleClientAuthentication(env.ctx, session, authReq))
	require.True(env.t, bytes.Equal([]byte{255, 41, 8, 0, 1, 0, 0, 0}, conn.Written),
		"Player %s not logged in, got: %v", username, conn.Written)

	// Select character
	selectReq := backend.SelectCharacterRequest(append(
		append([]byte(username), 0),
		append([]byte(characterName), 0)...,
	))
	require.NoError(env.t, bd.HandleSelectCharacter(env.ctx, session, selectReq))

	require.NoError(env.t, session.JoinLobby(env.ctx), "failed to join lobby")
	require.NoError(env.t, session.RegisterNewObserver(env.ctx), "failed to register observer")

	conn.Written = nil // Clear written data

	return &p2pPlayer{
		backend: bd,
		conn:    conn,
		session: session,
		name:    username,
	}
}

// createRoom creates a game room with the host player.
func (env *p2pTestEnv) createRoom(host *p2pPlayer, roomName string, mapID v1.GameMap) {
	// Create game room (first call sets state=0)
	createReq := backend.CreateGameRequest(append(
		[]byte{0, 0, 0, 0, byte(mapID), 0, 0, 0},
		append([]byte(roomName), 0, 0)...,
	))
	require.NoError(env.t, host.backend.HandleCreateGame(env.ctx, host.session, createReq))

	// Set room ready (second call sets state=1)
	readyReq := backend.CreateGameRequest(append(
		[]byte{1, 0, 0, 0, byte(mapID), 0, 0, 0},
		append([]byte(roomName), 0, 0)...,
	))
	require.NoError(env.t, host.backend.HandleCreateGame(env.ctx, host.session, readyReq))

	// Process the SetRoomReady message
	env.console.RoomService.HandleIncomingMessage(env.ctx, <-env.console.RoomService.Messages)

	host.conn.Written = nil
}

// joinRoom has a player join an existing room.
func (env *p2pTestEnv) joinRoom(player *p2pPlayer, roomName string) {
	// List games
	require.NoError(env.t, player.backend.HandleListGames(env.ctx, player.session, backend.ListGamesRequest{}))
	player.conn.Written = nil

	// Select game
	selectReq := backend.SelectGameRequest(append([]byte(roomName), 0, 0))
	require.NoError(env.t, player.backend.HandleSelectGame(env.ctx, player.session, selectReq))
	player.conn.Written = nil

	// Join game
	joinReq := backend.JoinGameRequest(append([]byte(roomName), 0, 0))
	require.NoError(env.t, player.backend.HandleJoinGame(env.ctx, player.session, joinReq))
	player.conn.Written = nil
}

// processMessages processes all pending WebSocket messages for a short duration.
func (env *p2pTestEnv) processMessages(duration time.Duration) {
	timeout := time.After(duration)
	for {
		select {
		case msg := <-env.console.RoomService.Messages:
			env.console.RoomService.HandleIncomingMessage(env.ctx, msg)
		case <-timeout:
			return
		}
	}
}

// TestE2E_P2P_HostMigration tests that when the host leaves, another player becomes host.
func TestE2E_P2P_HostMigration(t *testing.T) {
	env := setupP2PEnv(t)

	// Create players
	host := env.createPlayer("archer", "archer")
	guest := env.createPlayer("mage", "mage")

	// Host creates room
	env.createRoom(host, "testroom", v1.GameMap_FrozenLabyrinth)

	snap := env.console.RoomService.GetRoomSnapshot("testroom")
	require.True(t, snap.Exists, "room not found")
	require.Equal(t, host.session.UserID, snap.HostUserID, "host should be archer")

	// Guest joins
	env.joinRoom(guest, "testroom")

	// Process WebRTC signaling messages
	env.processMessages(3 * time.Second)

	// Verify both players are in room
	snap = env.console.RoomService.GetRoomSnapshot("testroom")
	require.Equal(t, 2, len(snap.PlayerIDs), "should have 2 players")

	// Get the host's user session for LeaveRoom
	hostSession, ok := env.console.RoomService.GetUserSession(host.session.UserID)
	require.True(t, ok, "host session not found")

	// Host leaves
	env.console.RoomService.LeaveRoom(env.ctx, hostSession)

	// Process any remaining messages
	env.processMessages(1 * time.Second)

	// Verify guest is now host
	snap = env.console.RoomService.GetRoomSnapshot("testroom")
	require.True(t, snap.Exists, "room should still exist")
	require.Equal(t, 1, len(snap.PlayerIDs), "should have 1 player after host left")
	require.Equal(t, guest.session.UserID, snap.HostUserID, "mage should now be host")

	t.Log("Host migration successful: mage is now host")
}

// TestE2E_P2P_ThirdPlayerJoins tests 3 players joining a game room.
func TestE2E_P2P_ThirdPlayerJoins(t *testing.T) {
	env := setupP2PEnv(t)

	// Create players
	host := env.createPlayer("archer", "archer")
	guest1 := env.createPlayer("mage", "mage")
	guest2 := env.createPlayer("warrior", "warrior")

	// Host creates room
	env.createRoom(host, "bigroom", v1.GameMap_AbandonedRealm)

	// First guest joins
	env.joinRoom(guest1, "bigroom")

	// Process WebRTC signaling for first guest
	env.processMessages(2 * time.Second)

	snap := env.console.RoomService.GetRoomSnapshot("bigroom")
	require.Equal(t, 2, len(snap.PlayerIDs), "should have 2 players after first guest joins")

	// Second guest joins
	env.joinRoom(guest2, "bigroom")

	// Process WebRTC signaling for second guest
	env.processMessages(3 * time.Second)

	// Verify all 3 players are in room
	snap = env.console.RoomService.GetRoomSnapshot("bigroom")
	require.True(t, snap.Exists, "room not found")
	require.Equal(t, 3, len(snap.PlayerIDs), "should have 3 players")
	require.Equal(t, host.session.UserID, snap.HostUserID, "host should still be archer")

	// Verify each player is present
	hasHost := containsUserID(snap.PlayerIDs, host.session.UserID)
	hasGuest1 := containsUserID(snap.PlayerIDs, guest1.session.UserID)
	hasGuest2 := containsUserID(snap.PlayerIDs, guest2.session.UserID)
	require.True(t, hasHost, "archer should be in room")
	require.True(t, hasGuest1, "mage should be in room")
	require.True(t, hasGuest2, "warrior should be in room")

	t.Log("3-player room setup successful")
}

// TestE2E_P2P_FourPlayersOneLeaves tests a 4-player room where one player leaves.
func TestE2E_P2P_FourPlayersOneLeaves(t *testing.T) {
	env := setupP2PEnv(t)

	// Create players
	host := env.createPlayer("archer", "archer")
	guest1 := env.createPlayer("mage", "mage")
	guest2 := env.createPlayer("warrior", "warrior")
	guest3 := env.createPlayer("necro", "necro")

	// Host creates room
	env.createRoom(host, "fullroom", v1.GameMap_CrimsonAshes)

	// All guests join sequentially
	env.joinRoom(guest1, "fullroom")
	env.processMessages(2 * time.Second)

	env.joinRoom(guest2, "fullroom")
	env.processMessages(2 * time.Second)

	env.joinRoom(guest3, "fullroom")
	env.processMessages(3 * time.Second)

	// Verify 4 players in room
	snap := env.console.RoomService.GetRoomSnapshot("fullroom")
	require.True(t, snap.Exists, "room not found")
	require.Equal(t, 4, len(snap.PlayerIDs), "should have 4 players")

	t.Log("4-player room setup complete")

	// Guest2 (warrior) leaves
	guest2Session, ok := env.console.RoomService.GetUserSession(guest2.session.UserID)
	require.True(t, ok, "guest2 session not found")
	env.console.RoomService.LeaveRoom(env.ctx, guest2Session)

	// Process leave messages
	env.processMessages(1 * time.Second)

	// Verify cleanup
	snap = env.console.RoomService.GetRoomSnapshot("fullroom")
	require.True(t, snap.Exists, "room should still exist")
	require.Equal(t, 3, len(snap.PlayerIDs), "should have 3 players after one left")
	require.Equal(t, host.session.UserID, snap.HostUserID, "host should still be archer")

	// Verify warrior is gone but others remain
	hasHost := containsUserID(snap.PlayerIDs, host.session.UserID)
	hasGuest1 := containsUserID(snap.PlayerIDs, guest1.session.UserID)
	hasGuest2 := containsUserID(snap.PlayerIDs, guest2.session.UserID)
	hasGuest3 := containsUserID(snap.PlayerIDs, guest3.session.UserID)
	require.True(t, hasHost, "archer should be in room")
	require.True(t, hasGuest1, "mage should be in room")
	require.False(t, hasGuest2, "warrior should NOT be in room")
	require.True(t, hasGuest3, "necro should be in room")

	t.Log("Player cleanup after leave successful")
}

// TestE2E_P2P_HostLeavesWithMultiplePlayers tests host migration in a room with 3+ players.
func TestE2E_P2P_HostLeavesWithMultiplePlayers(t *testing.T) {
	env := setupP2PEnv(t)

	// Create players
	host := env.createPlayer("archer", "archer")
	guest1 := env.createPlayer("mage", "mage")
	guest2 := env.createPlayer("warrior", "warrior")

	// Host creates room
	env.createRoom(host, "migroom", v1.GameMap_FrozenLabyrinth)

	// Guests join
	env.joinRoom(guest1, "migroom")
	env.processMessages(2 * time.Second)

	env.joinRoom(guest2, "migroom")
	env.processMessages(3 * time.Second)

	// Verify 3 players
	snap := env.console.RoomService.GetRoomSnapshot("migroom")
	require.Equal(t, 3, len(snap.PlayerIDs), "should have 3 players")
	require.Equal(t, host.session.UserID, snap.HostUserID)

	// Record which guest joined first (for host selection)
	guest1Session, _ := env.console.RoomService.GetUserSession(guest1.session.UserID)
	guest2Session, _ := env.console.RoomService.GetUserSession(guest2.session.UserID)
	earlierGuest := guest1Session
	if guest2Session.JoinedAt.Before(guest1Session.JoinedAt) {
		earlierGuest = guest2Session
	}

	// Host leaves
	hostSession, _ := env.console.RoomService.GetUserSession(host.session.UserID)
	env.console.RoomService.LeaveRoom(env.ctx, hostSession)

	// Process messages
	env.processMessages(1 * time.Second)

	// Verify new host is the earlier guest
	snap = env.console.RoomService.GetRoomSnapshot("migroom")
	require.True(t, snap.Exists, "room should exist")
	require.Equal(t, 2, len(snap.PlayerIDs), "should have 2 players")
	require.Equal(t, earlierGuest.UserID, snap.HostUserID, "earlier guest should be new host")

	t.Logf("Host migration with 3 players: new host is user %d", snap.HostUserID)
}

// TestE2E_P2P_AllGuestsLeave tests that room is cleaned up when all guests leave.
func TestE2E_P2P_AllGuestsLeave(t *testing.T) {
	env := setupP2PEnv(t)

	// Create players
	host := env.createPlayer("archer", "archer")
	guest1 := env.createPlayer("mage", "mage")
	guest2 := env.createPlayer("warrior", "warrior")

	// Host creates room
	env.createRoom(host, "emptyroom", v1.GameMap_CrimsonAshes)

	// Guests join
	env.joinRoom(guest1, "emptyroom")
	env.processMessages(2 * time.Second)

	env.joinRoom(guest2, "emptyroom")
	env.processMessages(2 * time.Second)

	// Verify 3 players
	snap := env.console.RoomService.GetRoomSnapshot("emptyroom")
	require.Equal(t, 3, len(snap.PlayerIDs))

	// Both guests leave
	guest1Session, _ := env.console.RoomService.GetUserSession(guest1.session.UserID)
	env.console.RoomService.LeaveRoom(env.ctx, guest1Session)

	guest2Session, _ := env.console.RoomService.GetUserSession(guest2.session.UserID)
	env.console.RoomService.LeaveRoom(env.ctx, guest2Session)

	// Process messages
	env.processMessages(1 * time.Second)

	// Verify only host remains
	snap = env.console.RoomService.GetRoomSnapshot("emptyroom")
	require.True(t, snap.Exists, "room should exist")
	require.Equal(t, 1, len(snap.PlayerIDs), "only host should remain")
	require.Equal(t, host.session.UserID, snap.HostUserID)

	t.Log("All guests left, host remains alone")
}

// containsUserID reports whether ids contains id.
func containsUserID(ids []int64, id int64) bool {
	for _, u := range ids {
		if u == id {
			return true
		}
	}
	return false
}
