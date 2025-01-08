package backend

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	v1 "github.com/dimspell/gladiator/gen/multi/v1"
	"github.com/dimspell/gladiator/internal/app/logger"
	"github.com/dimspell/gladiator/internal/backend/packet"
	"github.com/dimspell/gladiator/internal/backend/proxy"
	"github.com/dimspell/gladiator/internal/console"
	"github.com/dimspell/gladiator/internal/console/database"
	"github.com/dimspell/gladiator/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestE2E_P2P(t *testing.T) {
	logger.SetColoredLogger(os.Stderr, slog.LevelDebug, false)

	helperStartGameServer(t)

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

	cs := &console.Console{
		Multiplayer: console.NewMultiplayer(),
		DB:          db,
	}
	ts := httptest.NewServer(cs.HttpRouter())
	defer ts.Close()

	// go cs.Multiplayer.Run(ctx)

	// Remove the HTTP schema prefix
	cs.Addr = ts.URL[len("http://"):]

	proxy1 := proxy.NewPeerToPeer()
	// proxy1.NewRedirect = redirectFunc
	bd1 := NewBackend("", cs.Addr, proxy1)
	bd1.SignalServerURL = "ws://" + cs.Addr + "/lobby"

	conn1 := &mockConn{}
	session1 := bd1.AddSession(conn1)

	// FIXME: Set IPRing in test mode2
	// session1.IpRing.IsTesting = true
	// session1.IpRing.UdpPortPrefix = 1300
	// session1.IpRing.TcpPortPrefix = 1400

	// Sign-in
	assert.NoError(t, bd1.HandleClientAuthentication(ctx, session1, ClientAuthenticationRequest{
		2, 0, 0, 0, // Unknown
		't', 'e', 's', 't', 0, // Password
		'a', 'r', 'c', 'h', 'e', 'r', 0, // Username
	}))
	if !bytes.Equal([]byte{255, 41, 8, 0, 1, 0, 0, 0}, conn1.Written) {
		t.Errorf("Not logged in, got: %v", conn1.Written)
		return
	}

	// Select character
	assert.NoError(t, bd1.HandleSelectCharacter(ctx, session1, SelectCharacterRequest{
		'a', 'r', 'c', 'h', 'e', 'r', 0, // User name
		'a', 'r', 'c', 'h', 'e', 'r', 0, // Character name
	}))
	err = session1.JoinLobby(ctx)
	if err != nil {
		t.Errorf("failed to join lobby: %v", err)
		return
	}
	err = bd1.RegisterNewObserver(ctx, session1)
	if err != nil {
		t.Errorf("failed to register new observer: %v", err)
		return
	}

	// Create new game room
	assert.NoError(t, bd1.HandleCreateGame(ctx, session1, CreateGameRequest{
		0, 0, 0, 0, // State
		byte(v1.GameMap_FrozenLabyrinth), 0, 0, 0, // Map ID
		'r', 'o', 'o', 'm', 0, // Game room name
		0, // Password
	}))
	assert.NoError(t, bd1.HandleCreateGame(ctx, session1, CreateGameRequest{
		1, 0, 0, 0, // State
		byte(v1.GameMap_FrozenLabyrinth), 0, 0, 0, // Map ID
		'r', 'o', 'o', 'm', 0, // Game room name
		0, // Password
	}))

	cs.Multiplayer.HandleIncomingMessage(ctx, <-cs.Multiplayer.Messages)

	room, ok := cs.Multiplayer.Rooms["room"]
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
	proxy2 := proxy.NewPeerToPeer()
	// proxy2.NewRedirect = redirectFunc
	bd2 := NewBackend("", cs.Addr, proxy2)
	bd2.SignalServerURL = "ws://" + cs.Addr + "/lobby"

	conn2 := &mockConn{}
	session2 := bd2.AddSession(conn2)

	// FIXME: Set IPRing in test mode
	// session2.IpRing.IsTesting = true
	// session2.IpRing.UdpPortPrefix = 2300
	// session2.IpRing.TcpPortPrefix = 2400

	// Sign-in by player2
	assert.NoError(t, bd2.HandleClientAuthentication(ctx, session2, ClientAuthenticationRequest{
		2, 0, 0, 0, // Unknown
		't', 'e', 's', 't', 0, // Password
		'm', 'a', 'g', 'e', 0, // Username
	}))
	if !bytes.Equal([]byte{255, 41, 8, 0, 1, 0, 0, 0}, conn2.Written) {
		t.Errorf("Not logged in, got: %v", conn2.Written)
		return
	}

	// Select character by player2
	assert.NoError(t, bd2.HandleSelectCharacter(ctx, session2, SelectCharacterRequest{
		'm', 'a', 'g', 'e', 0, // User name
		'm', 'a', 'g', 'e', 0, // Character name
	}))
	err = session2.JoinLobby(ctx)
	if err != nil {
		t.Errorf("failed to join lobby: %v", err)
		return
	}
	err = bd2.RegisterNewObserver(ctx, session2)
	if err != nil {
		t.Errorf("failed to register new observer: %v", err)
		return
	}

	// Truncate
	conn2.Written = nil

	// List games
	assert.NoError(t, bd2.HandleListGames(ctx, session2, ListGamesRequest{}))

	// Check if user has received the game list with corresponding payload
	assert.Equal(t, []byte{
		1, 0, 0, 0, // Number of games
		127, 0, 1, 2, // IP address of host
		'r', 'o', 'o', 'm', 0, // Room name
		0, // Password
	}, findPacket(conn2.Written, packet.ListGames))

	// Truncate
	conn2.Written = nil

	// Select game
	assert.NoError(t, bd2.HandleSelectGame(ctx, session2, SelectGameRequest{
		'r', 'o', 'o', 'm', 0, // Game name
		0, // Password
	}))

	// Check if the game is correct
	assert.Equal(t, []byte{
		byte(v1.GameMap_FrozenLabyrinth), 0, 0, 0, // Map ID
		byte(v1.ClassType_Archer), 0, 0, 0, // Host's character class type
		// 127, 0, 1, 2, // IP address of host
		127, 0, 1, 2, // IP address of host
		'a', 'r', 'c', 'h', 'e', 'r', 0, // Player name
	}, findPacket(conn2.Written, packet.SelectGame))

	// Truncate
	conn2.Written = nil

	// Join to host
	assert.NoError(t, bd2.HandleJoinGame(ctx, session2, JoinGameRequest{
		'r', 'o', 'o', 'm', 0, // Game name
		0, // Password
	}))

	// Ensure the response is correct
	assert.Equal(t, []byte{
		model.GameStateStarted, 0, // Game state
		byte(v1.ClassType_Archer), 0, 0, 0, // Host's character class type
		// 127, 0, 1, 2, // IP address of host
		127, 0, 1, 2, // IP address of host
		'a', 'r', 'c', 'h', 'e', 'r', 0, // Player name
	}, findPacket(conn2.Written, packet.JoinGame))

	// Room contains all data
	room, ok = cs.Multiplayer.Rooms["room"]
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

	mpSession1, ok := cs.Multiplayer.GetUserSession(1)
	assert.True(t, ok)
	assert.Equal(t, session1.UserID, mpSession1.UserID)
	assert.Equal(t, "room", mpSession1.GameID)

	mpSession2, ok := cs.Multiplayer.GetUserSession(2)
	assert.True(t, ok)
	assert.Equal(t, session2.UserID, mpSession2.UserID)
	assert.Equal(t, "room", mpSession2.GameID)

	// Host user has correct data
	assert.Equal(t, int64(1), mpSession1.UserID)
	assert.Equal(t, "archer", mpSession1.User.Username)
	assert.Equal(t, "127.0.0.1", mpSession1.IPAddress)

	// Joining user has also the same data
	assert.Equal(t, int64(2), mpSession2.UserID)
	assert.Equal(t, "mage", mpSession2.User.Username)
	assert.Equal(t, "127.0.0.1", mpSession2.IPAddress)

	// RTCICECandidate
	// cs.Multiplayer.HandleIncomingMessage(ctx, <-cs.Multiplayer.Messages)
	// cs.Multiplayer.HandleIncomingMessage(ctx, <-cs.Multiplayer.Messages)
	// cs.Multiplayer.HandleIncomingMessage(ctx, <-cs.Multiplayer.Messages)
	//
	// RTCICECandidate
	// cs.Multiplayer.HandleIncomingMessage(ctx, <-cs.Multiplayer.Messages)
	// cs.Multiplayer.HandleIncomingMessage(ctx, <-cs.Multiplayer.Messages)
	// cs.Multiplayer.HandleIncomingMessage(ctx, <-cs.Multiplayer.Messages)

	go func() {
		<-time.After(time.Second * 3)
		close(cs.Multiplayer.Messages)
	}()
	for message := range cs.Multiplayer.Messages {
		cs.Multiplayer.HandleIncomingMessage(ctx, message)
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
					slog.Debug("Failed to write to UDP", "error", err)
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
					conn.Write([]byte{35, 35, 116, 101, 115, 116, 0})
				}
			}
		}()

		for {
			conn.SetDeadline(time.Now().Add(10 * time.Second))

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

// func TestPeerToPeer_CreateRoom(t *testing.T) {
// 	tests := []struct {
// 		name       string
// 		params     CreateParams
// 		wantIP     net.IP
// 		wantErr    bool
// 		setupState func(*bsession.Session)
// 	}{
// 		{
// 			name: "create room with valid params",
// 			params: CreateParams{
// 				GameID: "test-game",
// 			},
// 			wantIP:  net.IPv4(127, 0, 0, 1),
// 			wantErr: false,
// 		},
// 		{
// 			name: "create room with existing session state",
// 			params: CreateParams{
// 				GameID: "existing-game",
// 			},
// 			wantIP:  net.IPv4(127, 0, 0, 1),
// 			wantErr: false,
// 			setupState: func(s *bsession.Session) {
// 				s.State.gameRoom = NewGameRoom("old-game", &Player{})
// 			},
// 		},
// 		{
// 			name: "create room with empty game ID",
// 			params: CreateParams{
// 				GameID: "",
// 			},
// 			wantIP:  net.IPv4(127, 0, 0, 1),
// 			wantErr: false,
// 		},
// 	}
//
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			p := NewPeerToPeer()
// 			session := &Session{
// 				UserID:   1,
// 				Username: "testuser",
// 				State:    NewState(),
// 			}
//
// 			if tt.setupState != nil {
// 				tt.setupState(session)
// 			}
//
// 			gotIP, err := p.CreateRoom(tt.params, session)
//
// 			if tt.wantErr {
// 				assert.Error(t, err)
// 				return
// 			}
//
// 			assert.NoError(t, err)
// 			assert.Equal(t, tt.wantIP, gotIP)
// 			assert.NotNil(t, session.State.gameRoom)
// 			assert.Equal(t, tt.params.GameID, session.State.gameRoom.ID)
// 			assert.Equal(t, session.Username, session.State.gameRoom.HostPlayer.Username)
// 			assert.Equal(t, gotIP, session.State.gameRoom.HostPlayer.IP)
// 		})
// 	}
// }
