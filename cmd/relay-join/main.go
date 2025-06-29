package main

import (
	"context"
	"encoding/json"
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
	"github.com/go-chi/chi/v5"
)

var _ net.Conn = (*mockConn)(nil)

type mockConn struct{}

func (m mockConn) Read(b []byte) (n int, err error) {
	// TODO implement me
	panic("implement me")
}

func (m mockConn) Write(b []byte) (n int, err error) {
	return fmt.Fprintln(os.Stderr, b)
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
}

const roomID = "testroom"

var mapping = map[string]Map{
	"2": { // joins to host
		IPPrefix:    net.IPv4(127, 0, 2, 1).To4(),
		SessionID:   "sid-2",
		UserID:      2,
		CharacterID: 2,
		UserName:    "warrior",
		ClassType:   model.ClassTypeWarrior,
	},
	"3": { // joins as third
		IPPrefix:    net.IPv4(127, 0, 3, 1).To4(),
		SessionID:   "sid-3",
		UserID:      3,
		UserName:    "archer",
		CharacterID: 3,
		ClassType:   model.ClassTypeArcher,
	},
	"4": { // joins to 2 and 3 after 2 became a host
		IPPrefix:    net.IPv4(127, 0, 4, 1).To4(),
		SessionID:   "sid-4",
		UserID:      4,
		UserName:    "mage",
		CharacterID: 4,
		ClassType:   model.ClassTypeMage,
	},
}

func main() {
	// logger.SetColoredLogger(os.Stderr, slog.LevelDebug, false)
	logger.SetColoredLogger(os.Stderr, slog.LevelInfo, false)

	var userID string
	flag.StringVar(&userID, "player", "", "ID of player variant")
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
	proxyClient := px.Create(session).(*relay.Relay)
	session.Proxy = proxyClient
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

	consoleUri := fmt.Sprintf("%s://%s/grpc", "http", "localhost:2137")
	gameClient := multiv1connect.NewGameServiceClient(&http.Client{Timeout: 10 * time.Second}, consoleUri)

	gameRes, err := gameClient.GetGame(ctx, connect.NewRequest(&multiv1.GetGameRequest{
		GameRoomId: roomID,
	}))
	if err != nil {
		slog.Error("GetGame", logging.Error(err))
		return
	}

	if err := session.Proxy.SelectGame(proxy.GameData{
		Game:    gameRes.Msg.Game,
		Players: gameRes.Msg.Players,
	}); err != nil {
		slog.Error("SelectGame", logging.Error(err))
		return
	}

	if _, err := gameClient.JoinGame(ctx, connect.NewRequest(&multiv1.JoinGameRequest{
		UserId:     session.UserID,
		GameRoomId: roomID,
		IpAddress:  "127.0.0.1",
	})); err != nil {
		slog.Error("JoinGame", logging.Error(err))
		return
	}

	if _, err := session.Proxy.Join(ctx, proxy.JoinParams{
		HostUserID: 1,
		GameID:     roomID,
		HostUserIP: "127.0.0.2",
	}); err != nil {
		slog.Error("Join", logging.Error(err))
		return
	}

	r := chi.NewRouter()
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		v := State{
			User:  user,
			Debug: proxyClient.Debug(),
		}

		doc, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(doc)
	})

	addr := fmt.Sprintf("localhost:999%d", user.UserID)
	fmt.Println("Listening on", fmt.Sprintf("http://%s/", addr))
	_ = http.ListenAndServe(addr, r)
}

type State struct {
	User  Map `json:"user"`
	Debug any `json:"debug"`
}
