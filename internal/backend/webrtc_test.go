package backend

import (
	"context"
	"log/slog"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"connectrpc.com/connect"
	v1 "github.com/dimspell/gladiator/gen/multi/v1"
	"github.com/dimspell/gladiator/internal/app/logger"
	"github.com/dimspell/gladiator/internal/console"
	"github.com/dimspell/gladiator/internal/console/database"
	"github.com/dimspell/gladiator/internal/model"
	"github.com/dimspell/gladiator/internal/proxy/p2p"
	"github.com/dimspell/gladiator/internal/proxy/redirect"
)

func TestWebRTC(t *testing.T) {
	logger.SetColoredLogger(os.Stderr, slog.LevelDebug, false)

	redirectFunc := redirect.New

	// Create in-memory database
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create console instance and serve the HTTP
	cs := &console.Console{
		Multiplayer: console.NewMultiplayer(),
		DB:          db,
	}
	ts := httptest.NewServer(cs.HttpRouter())
	defer ts.Close()

	// Remove the HTTP schema prefix
	cs.Addr = ts.URL[len("http://"):]

	go func() {
		<-time.After(3 * time.Second)
		close(cs.Multiplayer.Messages)
	}()
	go func() {
		for message := range cs.Multiplayer.Messages {
			t.Log("console handled message", message)
			cs.Multiplayer.HandleIncomingMessage(ctx, message)
		}
	}()

	// Mock the hosting user's proxy - player1
	proxy1 := NewPeerToPeer()
	proxy1.NewRedirect = redirectFunc
	bd1 := NewBackend("", cs.Addr, proxy1)
	bd1.SignalServerURL = "ws://" + cs.Addr + "/lobby"

	conn1 := &mockConn{}
	session1 := bd1.AddSession(conn1)
	session1.UserID = 1
	session1.CharacterID = 1
	session1.ClassType = model.ClassTypeArcher
	session1.IpRing.IsTesting = true
	session1.IpRing.UdpPortPrefix = 1300
	session1.IpRing.TcpPortPrefix = 1400

	if err := bd1.ConnectToLobby(ctx, &v1.User{UserId: 1, Username: "user1"}, session1); err != nil {
		t.Fatalf("failed to connect to lobby: %v", err)
		return
	}
	if err := bd1.JoinLobby(ctx, session1); err != nil {
		t.Fatalf("failed to join lobby: %v", err)
		return
	}
	if err := bd1.RegisterNewObserver(ctx, session1); err != nil {
		t.Fatalf("failed to register observer: %v", err)
		return
	}

	// Create new game room by the player1
	roomId := "room"
	if _, err := proxy1.CreateRoom(CreateParams{GameID: roomId}, session1); err != nil {
		t.Fatalf("failed to create room: %v", err)
		return
	}
	if _, err := bd1.gameClient.CreateGame(context.TODO(), connect.NewRequest(&v1.CreateGameRequest{
		GameName:      roomId,
		MapId:         v1.GameMap_AbandonedRealm,
		HostUserId:    1,
		HostIpAddress: "192.168.1.1",
	})); err != nil {
		t.Fatalf("failed to create game: %v", err)
	}

	if err := session1.SendSetRoomReady(ctx, roomId); err != nil {
		t.Fatalf("failed to send set room ready: %v", err)
		return
	}
	if len(cs.Multiplayer.Rooms) != 1 {
		t.Fatalf("multiplayer should have 1 room")
		return
	}

	// Create a joining user, a guest - player2
	proxy2 := NewPeerToPeer()
	proxy2.NewRedirect = redirectFunc
	bd2 := NewBackend("", cs.Addr, proxy2)
	bd2.SignalServerURL = "ws://" + cs.Addr + "/lobby"

	conn2 := &mockConn{}
	session2 := bd2.AddSession(conn2)
	session2.UserID = 2
	session2.CharacterID = 2
	session2.ClassType = model.ClassTypeMage
	session2.IpRing.IsTesting = true
	session2.IpRing.UdpPortPrefix = 2300
	session2.IpRing.TcpPortPrefix = 2400

	if err := bd2.ConnectToLobby(ctx, &v1.User{UserId: 2, Username: "user2"}, session2); err != nil {
		t.Fatalf("failed to connect to lobby: %v", err)
		return
	}
	if err := bd2.JoinLobby(ctx, session2); err != nil {
		t.Fatalf("failed to join lobby: %v", err)
		return
	}
	if err := bd2.RegisterNewObserver(ctx, session2); err != nil {
		t.Fatalf("failed to register observer: %v", err)
		return
	}

	// Make the packet redirect
	ip, portTCP, portUDP := session2.IpRing.NextAddr()
	peer := &p2p.Peer{
		PeerUserID: session2.GetUserID(),
		Addr:       &redirect.Addressing{IP: ip, TCPPort: portTCP, UDPPort: portUDP},
		Mode:       redirect.OtherUserIsHost,
	}

	gameRoom := NewGameRoom()
	session2.SetGameRoom(gameRoom)

	peers := map[string]*p2p.Peer{peer.PeerUserID: peer}
	proxy2.Peers[session2] = &PeersToSessionMapping{
		Game:  gameRoom,
		Peers: peers,
	}
}

// 	player2.Peers.Range(func(_ string, peer *p2p.Peer) {
// 		<-webrtc.GatheringCompletePromise(peer.Connection)
// 	})

// params := GetPlayerAddrParams{
//	GameID:     roomId,
//	UserID:     "2",
//	IPAddress:  "",
//	HostUserID: "1",
// }
// peer := session2.IpRing.NextPeerAddress(
//	params.UserID,
//	params.UserID == session2.GetUserID(),
//	params.UserID == params.HostUserID,
// )
// ip, err := proxy2.Join(JoinParams{
//	HostUserID: "1",
//	GameID:     roomId,
//	HostUserIP: "127.0.0.1",
// }, session2)
