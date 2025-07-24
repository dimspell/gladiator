package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	multiv1 "github.com/dimspell/gladiator/gen/multi/v1"
	"github.com/dimspell/gladiator/gen/multi/v1/multiv1connect"
	"github.com/dimspell/gladiator/internal/app/logger"
	"github.com/dimspell/gladiator/internal/app/logger/logging"
	"github.com/dimspell/gladiator/internal/backend/bsession"
	"github.com/dimspell/gladiator/internal/backend/proxy/p2p"
	"github.com/dimspell/gladiator/internal/model"
)

const (
	consoleUri        = "127.0.0.1:2137"
	roomId            = "room"
	otherUserId int64 = 1
)

var (
	meUserId int64 = 2
	meName         = "meuser"
)

var httpClient = &http.Client{Timeout: 10 * time.Second}

func main() {
	flag.StringVar(&meName, "name", meName, "name")
	flag.Int64Var(&meUserId, "id", meUserId, "id")
	flag.Parse()

	logger.SetColoredLogger(os.Stderr, slog.LevelDebug, false)

	ctx := context.Background()

	gm := multiv1connect.NewGameServiceClient(httpClient, fmt.Sprintf("http://%s/grpc", consoleUri))

	session := &bsession.Session{
		ID:          fmt.Sprintf("%d", meUserId),
		UserID:      meUserId,
		Username:    meName,
		CharacterID: meUserId,
		ClassType:   model.ClassTypeArcher,
		State:       &bsession.SessionState{},
	}
	user2 := &multiv1.User{
		UserId:   meUserId,
		Username: meName,
	}
	px := p2p.NewPeerToPeer(session, gm)
	// px.NewUDPRedirect = redirect.NewNoop
	// px.NewTCPRedirect = redirect.NewLineReader

	if err := session.ConnectOverWebsocket(ctx, user2, fmt.Sprintf("ws://%s/lobby", consoleUri)); err != nil {
		slog.Error("failed to connect over websocket", logging.Error(err))
		return
	}
	slog.Info("connected over websocket")

	if err := session.JoinLobby(ctx); err != nil {
		slog.Error("failed to join lobby over websocket", logging.Error(err))
		return
	}
	slog.Info("joined lobby over websocket")

	go func() {
		for {
			payload, err := session.ConsumeWebSocket(ctx)
			if err != nil {
				slog.Error("failed to consume websocket", logging.Error(err))
				return
			}
			if err := px.Handle(ctx, payload); err != nil {
				slog.Error("failed to handle websocket", logging.Error(err))
				return
			}
		}
	}()

	if _, _, err := px.GetGame(ctx, roomId); err != nil {
		slog.Error("failed to select a game", logging.Error(err))
		return
	}
	if _, err := px.JoinGame(ctx, roomId, ""); err != nil {
		slog.Error("failed to join game", logging.Error(err))
		return
	}

	slog.Info("joined game")

	select {}
}
