package backend

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	"connectrpc.com/connect"
	"connectrpc.com/otelconnect"
	"github.com/dispel-re/dispel-multi/backend/packetlogger"
	"github.com/dispel-re/dispel-multi/backend/proxy"
	"github.com/dispel-re/dispel-multi/gen/multi/v1/multiv1connect"
	"github.com/dispel-re/dispel-multi/model"
	"github.com/google/uuid"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
)

type Backend struct {
	Sessions     map[string]*model.Session
	PacketLogger *slog.Logger
	Queue        *nats.Conn

	EventChan chan uint8

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

	nc, _ := nats.Connect(fmt.Sprintf("localhost:%", server.DEFAULT_PORT))

	interceptor := connect.WithInterceptors(otelconnect.NewInterceptor())
	consoleUri := fmt.Sprintf("http://%s/grpc", consoleAddr)

	return &Backend{
		Sessions:     make(map[string]*model.Session),
		PacketLogger: slog.New(packetlogger.New(os.Stderr, &packetlogger.Options{Level: slog.LevelDebug})),
		Queue:        nc,

		CharacterClient: multiv1connect.NewCharacterServiceClient(httpClient, consoleUri, interceptor),
		GameClient:      multiv1connect.NewGameServiceClient(httpClient, consoleUri, interceptor),
		UserClient:      multiv1connect.NewUserServiceClient(httpClient, consoleUri, interceptor),
		RankingClient:   multiv1connect.NewRankingServiceClient(httpClient, consoleUri, interceptor),
	}
}

func (b *Backend) Start(ctx context.Context) {
	if err := b.Events(ctx); err != nil {
		log.Fatal("Backend.Start", err)
	}
}

func (b *Backend) Shutdown(ctx context.Context) {
	// Close all open connections
	for _, session := range b.Sessions {
		session.Conn.Close()
	}

	if b.Queue != nil {
		b.Queue.Drain()
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

func (b *Backend) Events(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case eventType := <-b.EventChan:
			// TODO: Make a better distinction between events
			switch eventType {
			case EventNone:
				continue
			case EventHostGame:
				go proxy.MockHostTCPServer(ctx)
				go proxy.MockHostUDPServer(ctx)
			case EventCloseConn:
				cancel()
			}
		}
	}
}

const (
	EventNone uint8 = iota
	EventHostGame
	EventCloseConn
)

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

	b.EventChan <- EventCloseConn
	session = nil
	return nil
}
