package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	"connectrpc.com/connect"
	multiv1 "github.com/dimspell/gladiator/gen/multi/v1"
	"github.com/dimspell/gladiator/gen/multi/v1/multiv1connect"
	"github.com/dimspell/gladiator/internal/app/logger"
	"github.com/dimspell/gladiator/internal/app/logger/logging"
	"github.com/dimspell/gladiator/internal/backend/bsession"
	"github.com/dimspell/gladiator/internal/backend/proxy"
	"github.com/dimspell/gladiator/internal/backend/proxy/relay"
	"github.com/dimspell/gladiator/internal/model"
)

var _ net.Conn = (*mockConn)(nil)

type mockConn struct{}

func (m mockConn) Read(b []byte) (n int, err error) {
	// TODO implement me
	panic("implement me")
}

func (m mockConn) Write(b []byte) (n int, err error) {
	// TODO implement me
	panic("implement me")
}

func (m mockConn) Close() error {
	// TODO implement me
	panic("implement me")
}

func (m mockConn) LocalAddr() net.Addr {
	// TODO implement me
	panic("implement me")
}

func (m mockConn) RemoteAddr() net.Addr {
	// TODO implement me
	panic("implement me")
}

func (m mockConn) SetDeadline(t time.Time) error {
	// TODO implement me
	panic("implement me")
}

func (m mockConn) SetReadDeadline(t time.Time) error {
	// TODO implement me
	panic("implement me")
}

func (m mockConn) SetWriteDeadline(t time.Time) error {
	// TODO implement me
	panic("implement me")
}

type Map struct {
	SessionID   string
	UserID      int64
	CharacterID int
	UserName    string
	ClassType   model.ClassType
	IPPrefix    net.IP

	Game    *multiv1.Game
	Players []*multiv1.Player
}

const roomID = "testroom"

var mapping = map[string]Map{
	"2": { // joins to host
		IPPrefix:    net.IPv4(127, 0, 3, 1).To4(),
		SessionID:   "sid-2",
		UserID:      2,
		CharacterID: 2,
		UserName:    "mage",
		ClassType:   model.ClassTypeArcher,

		Game: &multiv1.Game{
			GameId:        roomID,
			Name:          roomID,
			Password:      "",
			MapId:         multiv1.GameMap_ScatteredShelter,
			HostUserId:    1,
			HostIpAddress: "",
		},
		Players: []*multiv1.Player{
			{
				UserId:      1,
				Username:    "mage",
				CharacterId: 1,
				ClassType:   multiv1.ClassType_Mage,
				IpAddress:   "",
			},
		},
	},
	"3": { // joins as third
		SessionID:   "sid-3",
		UserID:      3,
		UserName:    "warrior",
		CharacterID: 3,
		ClassType:   model.ClassTypeWarrior,

		Game: &multiv1.Game{
			GameId:        roomID,
			Name:          roomID,
			Password:      "",
			MapId:         multiv1.GameMap_ScatteredShelter,
			HostUserId:    1,
			HostIpAddress: "",
		},
		Players: []*multiv1.Player{
			{
				UserId:      1,
				Username:    "mage",
				CharacterId: 1,
				ClassType:   multiv1.ClassType_Mage,
				IpAddress:   "",
			},
			{
				UserId:      2,
				Username:    "archer",
				CharacterId: 2,
				ClassType:   multiv1.ClassType_Archer,
				IpAddress:   "",
			},
		},
	},
	"4": { // joins to 2 and 3 after 2 became a host
		SessionID:   "sid-4",
		UserID:      4,
		UserName:    "knight",
		CharacterID: 4,
		ClassType:   model.ClassTypeKnight,

		Game: &multiv1.Game{
			GameId:        roomID,
			Name:          roomID,
			Password:      "",
			MapId:         multiv1.GameMap_ScatteredShelter,
			HostUserId:    2,
			HostIpAddress: "",
		},
		Players: []*multiv1.Player{
			{
				UserId:      2,
				Username:    "archer",
				CharacterId: 2,
				ClassType:   multiv1.ClassType_Archer,
				IpAddress:   "",
			},
			{
				UserId:      3,
				Username:    "warrior",
				CharacterId: 3,
				ClassType:   multiv1.ClassType_Warrior,
				IpAddress:   "",
			},
		},
	},
}

func main() {
	logger.SetColoredLogger(os.Stderr, slog.LevelDebug, false)

	var userID string
	flag.StringVar(&userID, "user-id", "", "User ID")
	flag.Parse()

	user, ok := mapping[userID]
	if !ok {
		return
	}

	px := &relay.ProxyRelay{
		RelayServerAddr: "localhost:9999",
		IPPrefix:        user.IPPrefix,
	}
	session := bsession.NewSession(nil)
	session.ID = user.SessionID
	session.UserID = user.UserID
	session.Username = user.UserName
	session.CharacterID = int64(user.CharacterID)
	session.ClassType = user.ClassType
	session.Proxy = px.Create(session)
	session.Conn = &mockConn{}

	ctx := context.TODO()

	if err := session.ConnectOverWebsocket(ctx, &multiv1.User{
		UserId:   session.UserID,
		Username: session.Username,
	}, fmt.Sprintf("ws://%s/lobby", "localhost:2137")); err != nil {
		slog.Error("failed to connect over websocket", logging.Error(err))
		return
	}
	if err := session.JoinLobby(ctx); err != nil {
		slog.Error("JoinLobby", logging.Error(err))
		return
	}

	go func(ctx context.Context) {
		for {
			p, err := session.ConsumeWebSocket(ctx)
			if err != nil {
				if errors.Is(err, context.Canceled) {
					return
				}
				slog.Error("Error reading from WebSocket", "session", session.ID, logging.Error(err))
				return
			}
			if err := session.Proxy.Handle(ctx, p); err != nil {
				slog.Error("Error handling message", "session", session.ID, logging.Error(err))
				return
			}
		}
	}(ctx)

	var err error

	err = session.Proxy.SelectGame(proxy.GameData{
		Game:    user.Game,
		Players: user.Players,
	})
	if err != nil {
		slog.Error("SelectGame", logging.Error(err))
		return
	}

	consoleUri := fmt.Sprintf("%s://%s/grpc", "http", "localhost:2137")
	gameClient := multiv1connect.NewGameServiceClient(&http.Client{Timeout: 10 * time.Second}, consoleUri)
	gameClient.JoinGame(ctx, connect.NewRequest(&multiv1.JoinGameRequest{
		UserId:     session.UserID,
		GameRoomId: roomID,
		IpAddress:  "127.0.0.1",
	}))

	_, err = session.Proxy.Join(ctx, proxy.JoinParams{
		HostUserID: 1,
		GameID:     roomID,
		HostUserIP: "127.0.0.2",
	})
	if err != nil {
		slog.Error("Join", logging.Error(err))
		return
	}

	select {}
}
