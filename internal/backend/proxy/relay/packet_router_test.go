package relay

import (
	"context"
	"fmt"
	"log/slog"
	"os"
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

func TestFakeHosts(t *testing.T) {
	logger.SetPlainTextLogger(os.Stderr, slog.LevelDebug)

	t.Run("Dynamic Join", func(t *testing.T) {
		t.Log("I am a host and someone joined me and we play together")

		// Arrange
		roomID := "testingRoom"

		mp := console.NewMultiplayer()
		relayServer, err := console.NewQUICRelay("localhost:9999", mp)
		if err != nil {
			t.Fatal(err)
			return
		}
		go relayServer.Start(t.Context())
		go func() {
			for {
				for event := range relayServer.Events {
					fmt.Println("event", event)
					mp.HandleRelayEvent(event)
				}
			}
		}()

		// player1, proxyClient1, lobbySession1 := createSession(mp, 1)
		// player1, proxyClient1, _ := createSession(mp, 1)
		_, proxyClient1, _ := createSession(mp, 1)
		if _, err := proxyClient1.CreateRoom(proxy.CreateParams{GameID: roomID}); err != nil {
			t.Error(err)
			return
		}
		mp.SetRoomReady(wire.Message{Content: roomID}) // Instead calling HostRoom

		fmt.Println(mp.Rooms)
		fmt.Println(mp.ListRooms())

		proxyClient1.router.disconnect()

		time.Sleep(time.Millisecond * 1000)

		// // player2, relayProxy2, lobbySession2 := createSession(mp, 2)
		// _, relayProxy2, _ := createSession(mp, 2)
		//
		// relayProxy2.router.roomID = roomID
		// relayProxy2.router.currentHostID = strconv.Itoa(int(player1.UserID))
		//
		// if err := relayProxy2.SelectGame(proxy.GameData{
		// 	Game: &v1.Game{GameId: roomID, Name: roomID, HostUserId: 1},
		// 	Players: []*v1.Player{
		// 		{
		// 			UserId:      player1.UserID,
		// 			Username:    player1.Username,
		// 			CharacterId: player1.CharacterID,
		// 			ClassType:   v1.ClassType_Knight,
		// 		},
		// 	},
		// }); err != nil {
		// 	t.Error(err)
		// 	return
		// }
		//
		// // join room
		// if err := relayProxy2.router.connect(t.Context(), roomID); err != nil {
		// 	t.Error(err)
		// 	return
		// }
		//
		// if err := startFakeHost(t.Context(), relayProxy2.router.manager, &Host{
		// 	PeerID:   strconv.Itoa(int(player1.UserID)),
		// 	HostType: "LISTEN",
		// 	UDPPort:  6113,
		// 	TCPPort:  6114,
		// }); err != nil {
		// 	t.Error(err)
		// 	return
		// }

		fmt.Println(mp.Rooms)
		fmt.Println(mp.ListRooms())

		// Act

		// Assert
		// 1 Number of users in the room = 2
		// 2 The first user is a host and the other is guest
		// 3 The guest is in the same room as the guest
		// 4 The host and the guest they have exact matching structure of fake hosts
	})

	t.Run("I am a host, playing alone and I closed the game", func(t *testing.T) {
		// Arrange

		// Act

		// Assert
		// 1 Room does not exist anymore in game list
		// 2 Host disconnected from the Relay
	})

	t.Run("I am a host, someone joined me and I closed the game", func(t *testing.T) {
		// Arrange

		// Act

		// Assert
		// 1 Room still does exist
		// 2 The oldest guest is now host in the room
		// 3 Host is disconnected from the Relay
		// 4 Guest is still connected to the Relay
		// 5 Guest closed the fake host
	})
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

type Host struct {
	PeerID   string
	HostType string
	UDPPort  int
	TCPPort  int

	assignedIP string
	fakeHost   *redirect.FakeHost
}

func startFakeHost(ctx context.Context, hm *redirect.HostManager, params *Host) error {
	ip, err := hm.AssignIP(params.PeerID)
	if err != nil {
		return err
	}

	h, err := hm.CreateFakeHost(ctx,
		"TEST",
		params.PeerID,
		ip,
		&redirect.ProxySpec{
			LocalIP: "127.0.0.1",
			Port:    params.TCPPort,
			Create: func(ipv4, port string) (redirect.Redirect, error) {
				if params.HostType == "LISTEN" {
					return redirect.ListenTCP(ipv4, port)
				}
				if params.HostType == "DIAL" {
					return redirect.DialTCP(ipv4, port)
				}
				return nil, fmt.Errorf("unknown host type %s", params.HostType)
			},
			OnReceive: func(p []byte) error {
				slog.Info("[TCP] Received", "data", string(p))
				return nil
			},
		},
		&redirect.ProxySpec{
			LocalIP: "127.0.0.1",
			Port:    params.UDPPort,
			Create: func(ipv4, port string) (redirect.Redirect, error) {
				if params.HostType == "LISTEN" {
					return redirect.ListenUDP(ipv4, port)
				}
				if params.HostType == "DIAL" {
					return redirect.DialUDP(ipv4, port)
				}
				return nil, fmt.Errorf("unknown host type %s", params.HostType)
			},
			OnReceive: func(p []byte) error {
				slog.Info("[UDP] Received", "data", string(p))
				return nil
			},
		},
		func(host *redirect.FakeHost) {
			fmt.Println("Disconnecting", params.PeerID, host)
			hm.StopHost(host)
		},
	)
	if err != nil {
		return err
	}

	params.assignedIP = ip
	params.fakeHost = h
	return nil
}
