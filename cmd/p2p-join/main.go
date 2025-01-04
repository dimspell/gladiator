package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"connectrpc.com/connect"
	multiv1 "github.com/dimspell/gladiator/gen/multi/v1"
	"github.com/dimspell/gladiator/gen/multi/v1/multiv1connect"
	"github.com/dimspell/gladiator/internal/app/logger"
	"github.com/dimspell/gladiator/internal/backend/bsession"
	"github.com/dimspell/gladiator/internal/backend/proxy"
	"github.com/dimspell/gladiator/internal/model"
)

const (
	consoleUri = "127.0.0.1:2137"
)

var httpClient = &http.Client{Timeout: 10 * time.Second}

func main() {
	logger.SetColoredLogger(os.Stderr, slog.LevelDebug, false)

	ctx := context.Background()

	gm := multiv1connect.NewGameServiceClient(httpClient, fmt.Sprintf("http://%s/grpc", consoleUri))
	px := proxy.NewPeerToPeer()

	session := &bsession.Session{
		ID:          "2",
		UserID:      2,
		Username:    "joiner",
		CharacterID: 2,
		ClassType:   model.ClassTypeMage,
		State:       &bsession.SessionState{},
	}
	user2 := &multiv1.User{
		UserId:   2,
		Username: "joiner",
	}

	if err := session.ConnectOverWebsocket(ctx, user2, fmt.Sprintf("ws://%s/lobby", consoleUri)); err != nil {
		slog.Error("failed to connect over websocket", "error", err)
		return
	}
	slog.Info("connected over websocket")

	if err := session.JoinLobby(ctx); err != nil {
		slog.Error("failed to join lobby over websocket", "error", err)
		return
	}
	slog.Info("joined lobby over websocket")

	go func() {
		handler := px.ExtendWire(session)

		for {
			payload, err := session.ConsumeWebSocket(ctx)
			if err != nil {
				slog.Error("failed to consume websocket", "error", err)
				return
			}
			if err := handler.Handle(ctx, payload); err != nil {
				slog.Error("failed to handle websocket", "error", err)
				return
			}
		}
	}()

	game, err := gm.GetGame(ctx, connect.NewRequest(&multiv1.GetGameRequest{
		GameRoomId: "testing",
	}))
	if err != nil {
		slog.Error("failed to get game", "error", err)
		return
	}
	slog.Info("got game", "game", game.Msg)

	if err := px.SelectGame(proxy.GameData{
		Game:    game.Msg.Game,
		Players: game.Msg.Players,
	}, session); err != nil {
		slog.Error("failed to select a game", "error", err)
		return
	}

	addr, err := px.GetPlayerAddr(proxy.GetPlayerAddrParams{
		GameID:     "testing",
		UserID:     "1",
		IPAddress:  "127.0.1.2",
		HostUserID: "1",
	}, session)
	if err != nil {
		slog.Error("failed to get player address", "error", err)
		return
	}
	slog.Info("got player address", "address", addr)

	// px.GetPlayerAddr(proxy.GetPlayerAddrParams{}, session)
	// px.ExtendWire(session)

	join, err := gm.JoinGame(ctx, connect.NewRequest(&multiv1.JoinGameRequest{
		UserId:     2,
		GameRoomId: "testing",
		IpAddress:  "127.0.0.1",
	}))
	if err != nil {
		slog.Error("failed to join game", "error", err)
		return
	}
	slog.Info("joined game", "game", join.Msg)

	if _, err := px.Join(proxy.JoinParams{
		HostUserID: "1",
		GameID:     "testing",
		HostUserIP: "127.0.1.2",
	}, session); err != nil {
		slog.Error("failed to join game", "error", err)
		return
	}

	addr2, err := px.GetPlayerAddr(proxy.GetPlayerAddrParams{
		GameID:     "testing",
		UserID:     "1",
		IPAddress:  "127.0.1.2",
		HostUserID: "1",
	}, session)
	if err != nil {
		slog.Error("failed to get player address", "error", err)
		return
	}
	slog.Info("got player address", "address", addr2)

	select {}
}
