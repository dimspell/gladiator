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
	"github.com/dimspell/gladiator/internal/backend/proxy"
	"github.com/dimspell/gladiator/internal/backend/proxy/p2p"
	"github.com/dimspell/gladiator/internal/console"
	"github.com/dimspell/gladiator/internal/console/database"
	"github.com/dimspell/gladiator/internal/model"
)

func TestWebRTC(t *testing.T) {
	logger.SetColoredLogger(os.Stderr, slog.LevelDebug, false)

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
	proxy1 := p2p.NewPeerToPeer()
	bd1 := NewBackend("", cs.Addr, proxy1)
	bd1.SignalServerURL = "ws://" + cs.Addr + "/lobby"

	conn1 := &mockConn{}
	session1 := bd1.AddSession(conn1)
	session1.UserID = 1
	session1.CharacterID = 1
	session1.ClassType = model.ClassTypeArcher

	// FIXME: Set IPRing in test mode
	// session1.IpRing.IsTesting = true
	// session1.IpRing.UdpPortPrefix = 1300
	// session1.IpRing.TcpPortPrefix = 1400

	if err := bd1.ConnectToLobby(ctx, &v1.User{UserId: 1, Username: "user1"}, session1); err != nil {
		t.Fatalf("failed to connect to lobby: %v", err)
		return
	}
	if err := session1.JoinLobby(ctx); err != nil {
		t.Fatalf("failed to join lobby: %v", err)
		return
	}
	if err := bd1.RegisterNewObserver(ctx, session1); err != nil {
		t.Fatalf("failed to register observer: %v", err)
		return
	}

	// Create new game room by the player1
	roomId := "room"
	if _, err := proxy1.CreateRoom(proxy.CreateParams{GameID: roomId}, session1); err != nil {
		t.Fatalf("failed to create room: %v", err)
		return
	}
	if _, err := bd1.gameClient.CreateGame(ctx, connect.NewRequest(&v1.CreateGameRequest{
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
	proxy2 := p2p.NewPeerToPeer()
	bd2 := NewBackend("", cs.Addr, proxy2)
	bd2.SignalServerURL = "ws://" + cs.Addr + "/lobby"

	conn2 := &mockConn{}
	session2 := bd2.AddSession(conn2)
	session2.UserID = 2
	session2.CharacterID = 2
	session2.ClassType = model.ClassTypeMage

	// FIXME: Set IPRing in test mode
	// session2.IpRing.IsTesting = true
	// session2.IpRing.UdpPortPrefix = 2300
	// session2.IpRing.TcpPortPrefix = 2400

	if err := bd2.ConnectToLobby(ctx, &v1.User{UserId: 2, Username: "user2"}, session2); err != nil {
		t.Fatalf("failed to connect to lobby: %v", err)
		return
	}
	if err := session2.JoinLobby(ctx); err != nil {
		t.Fatalf("failed to join lobby: %v", err)
		return
	}
	if err := bd2.RegisterNewObserver(ctx, session2); err != nil {
		t.Fatalf("failed to register observer: %v", err)
		return
	}

	// Make the packet redirect
	// ip, portTCP, portUDP := session2.IpRing.NextAddr()
	// peer := &Peer{
	// 	CreatorID: session2.GetUserID(),
	// 	Addr:       &redirect.Addressing{IP: ip, TCPPort: portTCP, UDPPort: portUDP},
	// 	Mode:       redirect.OtherUserIsHost,
	// }
	//
	// gameRoom := NewGameRoom(roomId, session2.ToPlayer(net.IPv4(127, 0, 0, 21)))
	// session2.State.SetGameRoom(gameRoom)
	//
	// peers := map[string]*Peer{peer.CreatorID: peer}
	// proxy2.manager.SessionStore[session2] = &GameManager{
	// 	Game:  gameRoom,
	// 	SessionStore: peers,
	// }

	// <-webrtc.GatheringCompletePromise(peer.Connection)
}
