package main

import (
	"context"
	"encoding/json"
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
	"github.com/go-chi/chi/v5"
)

func main() {
	logger.SetColoredLogger(os.Stderr, slog.LevelDebug, false)

	px := &relay.ProxyRelay{
		RelayServerAddr: "localhost:9999",
	}
	session := bsession.NewSession(nil)
	session.ID = "sid-1"
	session.UserID = 1
	session.Username = "knight"
	session.CharacterID = 1
	session.ClassType = model.ClassTypeKnight
	proxyClient := px.Create(session).(*relay.Relay)
	session.Proxy = proxyClient

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

	_, err = session.Proxy.CreateRoom(proxy.CreateParams{
		GameID: roomID,
	})
	if err != nil {
		slog.Error("CreateRoom", logging.Error(err))
		return
	}

	consoleUri := fmt.Sprintf("%s://%s/grpc", "http", "localhost:2137")
	gameClient := multiv1connect.NewGameServiceClient(&http.Client{Timeout: 10 * time.Second}, consoleUri)
	if _, err := gameClient.CreateGame(ctx, connect.NewRequest(&multiv1.CreateGameRequest{
		GameName:      roomID,
		Password:      "",
		MapId:         multiv1.GameMap(1),
		HostUserId:    session.UserID,
		HostIpAddress: "127.0.0.1",
	})); err != nil {
		slog.Error("CreateGame", logging.Error(err))
		return
	}

	// startFakeBackendServer(ctx)

	err = session.Proxy.HostRoom(ctx, proxy.HostParams{GameID: roomID})
	if err != nil {
		slog.Error("HostRoom", logging.Error(err))
		return
	}

	r := chi.NewRouter()
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		v := proxyClient.Debug()

		doc, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(doc)
	})

	addr := fmt.Sprintf("localhost:9991")
	fmt.Println("Listening on", fmt.Sprintf("http://%s/", addr))
	_ = http.ListenAndServe(addr, r)
}
