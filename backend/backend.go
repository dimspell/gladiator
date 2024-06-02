package backend

import (
	"context"
	"fmt"
	"io"
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
	"github.com/nats-io/nats.go"
)

const DesktopIP byte = 212
const LaptopIP byte = 169

const GameRoomName = "room"

// const ClientIP byte = DesktopIP
// const HostIP byte = LaptopIP

type Backend struct {
	Addr string

	Sessions       map[string]*model.Session
	PacketLogger   *slog.Logger
	Queue          *nats.Conn
	SessionCounter int

	Proxy     proxy.Proxy
	EventChan chan uint8
	Listener  net.Listener

	CharacterClient multiv1connect.CharacterServiceClient
	GameClient      multiv1connect.GameServiceClient
	UserClient      multiv1connect.UserServiceClient
	RankingClient   multiv1connect.RankingServiceClient
}

func NewBackend(backendAddr, consoleAddr string) *Backend {
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

	// nc, _ := nats.Connect(fmt.Sprintf("localhost:%d", server.DEFAULT_PORT))

	interceptor := connect.WithInterceptors(otelconnect.NewInterceptor())
	consoleUri := fmt.Sprintf("http://%s/grpc", consoleAddr)

	// p := proxy.NewClientProxy(fmt.Sprintf("192.168.121.%d", LaptopIP))

	return &Backend{
		Addr:         backendAddr,
		Sessions:     make(map[string]*model.Session),
		PacketLogger: slog.New(packetlogger.New(os.Stderr, &packetlogger.Options{Level: slog.LevelDebug})),
		// Queue:        nc,

		// ClientProxy: p,

		CharacterClient: multiv1connect.NewCharacterServiceClient(httpClient, consoleUri, interceptor),
		GameClient:      multiv1connect.NewGameServiceClient(httpClient, consoleUri, interceptor),
		UserClient:      multiv1connect.NewUserServiceClient(httpClient, consoleUri, interceptor),
		RankingClient:   multiv1connect.NewRankingServiceClient(httpClient, consoleUri, interceptor),
	}
}

func (b *Backend) Start(ctx context.Context) error {
	slog.Info("Starting backend")

	// Listen for incoming connections.
	listener, err := net.Listen("tcp4", b.Addr)
	if err != nil {
		slog.Error("Could not start listening on port 6112", "err", err)
		return err
	}
	b.Listener = listener

	slog.Info("Listening for new connections...")

	go func() {
		<-ctx.Done()
		b.Shutdown()
	}()

	// go b.ClientProxy.Start(ctx)
	return nil
}

func (b *Backend) Shutdown() {
	if b.Queue != nil {
		b.Queue.Drain()
	}

	// b.ClientProxy = nil

	// Close all open connections
	for _, session := range b.Sessions {
		session.Conn.Close()
	}

	b.Listener.Close()

	// TODO: Send a system message "(system) The server is going to close in less than 30 seconds"
	// TODO: Send a packet to trigger stats saving
	// TODO: Send a system message "(system): Your stats were saving, your game client might close in the next 10 seconds"
	// TODO: Send a packet to close the connection (malformed 255-21?)
}

func (b *Backend) Listen() {
	for {
		// Listen for an incoming connection.
		conn, err := b.Listener.Accept()
		if err != nil {
			slog.Warn("Error, when accepting incoming connection", "err", err)
			continue
		}
		slog.Info("Accepted connection",
			slog.String("remoteAddr", conn.RemoteAddr().String()),
			slog.String("localAddr", conn.LocalAddr().String()),
		)

		// Handle connections in a new goroutine.
		// go handleRequest(connPort, conn)
		go func() {
			if err := b.handleClient(conn); err != nil {
				slog.Warn("Communication with client has failed",
					"err", err)
			}
		}()
	}
}

func (b *Backend) handleClient(conn net.Conn) error {
	session, err := b.handshake(conn)
	if err != nil {
		conn.Close()
		if err == io.EOF {
			return nil
		}
		slog.Warn("Handshake failed", "err", err)
		return err
	}
	defer b.CloseSession(session)

	for {
		if err := b.handleCommands(session); err != nil {
			slog.Warn("Command failed", "err", err)
			return err
		}
	}
}

func (b *Backend) NewSession(conn net.Conn) *model.Session {
	// id := uuid.New().String()
	b.SessionCounter++
	id := fmt.Sprintf("%d", b.SessionCounter)
	slog.Debug("New session", "session", id)

	session := &model.Session{Conn: conn, ID: id}
	b.Sessions[id] = session
	return session
}

func (b *Backend) Events(ctx context.Context) error {
	// ctx, cancel := context.WithCancel(ctx)
	// defer cancel()

	// for {
	// 	select {
	// 	case <-ctx.Done():
	// 		return ctx.Err()
	// 	case eventType := <-b.EventChan:
	// 		// TODO: Make a better distinction between events
	// 		switch eventType {
	// 		case EventNone:
	// 			continue
	// 		case EventHostGame:
	// 			go proxy.MockHostTCPServer(ctx)
	// 			go proxy.MockHostUDPServer(ctx)
	// 		case EventCloseConn:
	// 			cancel()
	// 		}
	// 	}
	// }
	return nil
}

const (
	EventNone uint8 = iota
	EventHostGame
	EventCloseConn
)

func (b *Backend) CloseSession(session *model.Session) error {
	slog.Info("Session closed", "session", session.ID)

	b.Proxy.Close()

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
