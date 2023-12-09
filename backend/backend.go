package backend

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	"connectrpc.com/connect"
	"connectrpc.com/otelconnect"
	"github.com/dispel-re/dispel-multi/backend/packetlogger"
	"github.com/dispel-re/dispel-multi/gen/multi/v1/multiv1connect"
	"github.com/dispel-re/dispel-multi/model"
	"github.com/google/uuid"
)

type Backend struct {
	Sessions     map[string]*model.Session
	PacketLogger *slog.Logger

	CharacterClient multiv1connect.CharacterServiceClient
	GameClient      multiv1connect.GameServiceClient
	UserClient      multiv1connect.UserServiceClient
	RankingClient   multiv1connect.RankingServiceClient
}

func NewBackend(consoleAddr string) *Backend {
	httpClient := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			Proxy:                 http.DefaultTransport.(*http.Transport).Proxy,
			DialContext:           http.DefaultTransport.(*http.Transport).DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}

	interceptor := connect.WithInterceptors(otelconnect.NewInterceptor())
	consoleUri := fmt.Sprintf("http://%s/grpc", consoleAddr)

	return &Backend{
		Sessions:     make(map[string]*model.Session),
		PacketLogger: slog.New(packetlogger.New(os.Stderr, &packetlogger.Options{Level: slog.LevelDebug})),

		CharacterClient: multiv1connect.NewCharacterServiceClient(httpClient, consoleUri, interceptor),
		GameClient:      multiv1connect.NewGameServiceClient(httpClient, consoleUri, interceptor),
		UserClient:      multiv1connect.NewUserServiceClient(httpClient, consoleUri, interceptor),
		RankingClient:   multiv1connect.NewRankingServiceClient(httpClient, consoleUri, interceptor),
	}
}

func (b *Backend) Shutdown(ctx context.Context) {
	// Close all open connections
	for _, session := range b.Sessions {
		session.Conn.Close()
	}

	// TODO: Send a system message "(system) The server is going to close in less than 30 seconds"
	// TODO: Send a packet to trigger stats saving
	// TODO: Send a system message "(system): Your stats were saving, your game client might close in the next 10 seconds"
	// TODO: Send a packet to close the connection (malformed 255-21?)
}

func (b *Backend) NewSession(conn net.Conn) *model.Session {
	id := uuid.New().String()
	slog.Debug("New session", "session", id)

	session := &model.Session{Conn: conn, ID: id}
	b.Sessions[id] = session
	return session
}

func (b *Backend) CloseSession(session *model.Session) error {
	slog.Debug("Session closed", "session", session.ID)

	// TODO: wrap all errors
	_, ok := b.Sessions[session.ID]
	if ok {
		delete(b.Sessions, session.ID)
	}

	if session.Conn != nil {
		_ = session.Conn.Close()
	}

	session = nil
	return nil
}
