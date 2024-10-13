package backend

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/dimspell/gladiator/gen/multi/v1/multiv1connect"
	"github.com/dimspell/gladiator/internal/proxy"
)

type Backend struct {
	Addr            string
	SignalServerURL string

	listener net.Listener

	ConnectedSessions sync.Map
	SessionCounter    uint64

	Proxy proxy.Proxy

	characterClient multiv1connect.CharacterServiceClient
	gameClient      multiv1connect.GameServiceClient
	userClient      multiv1connect.UserServiceClient
	rankingClient   multiv1connect.RankingServiceClient
}

func NewBackend(backendAddr, consoleAddr string, gameProxy proxy.Proxy) *Backend {
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

	// TODO: Name the schema as parameter
	consoleUri := fmt.Sprintf("%s://%s/grpc", "http", consoleAddr)

	return &Backend{
		Addr:  backendAddr,
		Proxy: gameProxy,

		characterClient: multiv1connect.NewCharacterServiceClient(httpClient, consoleUri),
		gameClient:      multiv1connect.NewGameServiceClient(httpClient, consoleUri),
		userClient:      multiv1connect.NewUserServiceClient(httpClient, consoleUri),
		rankingClient:   multiv1connect.NewRankingServiceClient(httpClient, consoleUri),
	}
}

func (b *Backend) Start() error {
	slog.Info("Starting backend")
	// b.CommandServerSideChannel()

	// Listen for incoming connections.
	listener, err := net.Listen("tcp4", b.Addr)
	if err != nil {
		slog.Error("Could not start listening on port 6112", "err", err)
		return err
	}
	b.listener = listener

	slog.Info("Backend listening", "addr", b.listener.Addr())
	return nil
}

func (b *Backend) Shutdown() {
	slog.Info("Shutting down the backend...")

	// Close all open connections
	b.ConnectedSessions.Range(func(k, v any) bool {
		session := v.(*Session)

		// TODO: Send a system message "(system) The server is going to close in less than 30 seconds"
		_ = b.Send(session.Conn,
			ReceiveMessage,
			NewGlobalMessage("system-info", "The server is going to close..."))

		// TODO: Send a packet to trigger stats saving
		// TODO: Send a system message "(system): Your stats were saving, your game client might close in the next 10 seconds"

		// TODO: Send a packet to close the connection (malformed 255-21?)
		if err := session.Conn.Close(); err != nil {
			slog.Error("Could not close session", "err", err, "session", session.ID)
		}

		return true
	})

	if b.listener != nil {
		if err := b.listener.Close(); err != nil {
			slog.Warn("Could not close listener", "err", err)
		}
		b.listener = nil
	}

	slog.Info("The backend has successfully shut down")
}

func (b *Backend) Listen() {
	slog.Info("Backend is listening for new connections...", "addr", b.Addr)

	for {
		if b.listener == nil {
			return
		}
		// Listen for an incoming connection.
		conn, err := b.listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}
			slog.Warn("Error, when accepting incoming connection", "err", err)
			continue
		}
		slog.Info("Accepted connection",
			slog.String("remoteAddr", conn.RemoteAddr().String()),
			slog.String("localAddr", conn.LocalAddr().String()),
		)

		// Handle connections in a new goroutine.
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
		if err := conn.Close(); err != nil {
			slog.Error("Could not close connection in handshake", "err", err)
			return err
		}
		if err == io.EOF {
			return nil
		}
		slog.Warn("Handshake failed", "err", err)
		return err
	}
	defer func() {
		err := b.CloseSession(session)
		if err != nil {
			slog.Warn("Close session failed", "err", err)
		}
	}()

	for {
		if err := b.handleCommands(session); err != nil {
			slog.Warn("Command failed", "err", err)
			return err
		}
	}
}
