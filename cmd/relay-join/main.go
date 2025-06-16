package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
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

func main() {
	logger.SetColoredLogger(os.Stderr, slog.LevelDebug, false)

	px := &relay.ProxyRelay{
		RelayServerAddr: "localhost:9999",
	}
	session := bsession.NewSession(nil)
	session.ID = "sid-2"
	session.UserID = 2
	session.Username = "mage"
	session.CharacterID = 2
	session.ClassType = model.ClassTypeArcher
	session.Proxy = px.Create(session)

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

	roomID := "testroom"
	var err error

	err = session.Proxy.SelectGame(proxy.GameData{
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
