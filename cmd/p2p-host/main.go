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
		ID:          "1",
		UserID:      1,
		Username:    "host",
		CharacterID: 1,
		ClassType:   model.ClassTypeArcher,
		State:       &bsession.SessionState{},
	}
	user1 := &multiv1.User{
		UserId:   1,
		Username: "host",
	}

	if err := session.ConnectOverWebsocket(ctx, user1, fmt.Sprintf("ws://%s/lobby", consoleUri)); err != nil {
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

	game, err := gm.CreateGame(ctx, connect.NewRequest(&multiv1.CreateGameRequest{
		GameName:      "testing",
		Password:      "",
		MapId:         0,
		HostUserId:    1,
		HostIpAddress: "127.0.1.2", // Not used for P2P traffic
	}))
	if err != nil {
		slog.Error("failed to create game over console", "error", err)
		return
	}
	slog.Info("created game over console")

	if _, err := px.CreateRoom(proxy.CreateParams{GameID: game.Msg.Game.GameId}, session); err != nil {
		slog.Error("failed to create room over proxy", "error", err)
		return
	}
	slog.Info("created room over proxy")

	if err := px.HostRoom(proxy.HostParams{GameID: game.Msg.Game.GameId}, session); err != nil {
		slog.Error("failed to host room over proxy", "error", err)
		return
	}
	slog.Info("created a host room over proxy")

	select {}
}
