package backend

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/dimspell/gladiator/gen/multi/v1/multiv1connect"
	"github.com/dimspell/gladiator/internal/app/logger/logging"
	"github.com/dimspell/gladiator/internal/backend/bsession"
	"github.com/dimspell/gladiator/internal/backend/packet"
	"github.com/dimspell/gladiator/internal/model"
)

type Backend struct {
	Addr            string
	SignalServerURL string

	listener net.Listener

	ConnectedSessions sync.Map

	CreateProxy Proxy

	characterClient multiv1connect.CharacterServiceClient
	gameClient      multiv1connect.GameServiceClient
	userClient      multiv1connect.UserServiceClient
	rankingClient   multiv1connect.RankingServiceClient
}

func NewBackend(backendAddr, consolePublicAddr string, createProxy Proxy) *Backend {
	characterClient, gameClient, userClient, rankingClient := createServiceClients(consolePublicAddr)

	return &Backend{
		Addr:        backendAddr,
		CreateProxy: createProxy,

		characterClient: characterClient,
		gameClient:      gameClient,
		userClient:      userClient,
		rankingClient:   rankingClient,
	}
}

func createServiceClients(consoleAddr string) (
	multiv1connect.CharacterServiceClient,
	multiv1connect.GameServiceClient,
	multiv1connect.UserServiceClient,
	multiv1connect.RankingServiceClient,
) {
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

	consoleUri := fmt.Sprintf("%s/grpc", consoleAddr)

	characterClient := multiv1connect.NewCharacterServiceClient(httpClient, consoleUri)
	gameClient := multiv1connect.NewGameServiceClient(httpClient, consoleUri)
	userClient := multiv1connect.NewUserServiceClient(httpClient, consoleUri)
	rankingClient := multiv1connect.NewRankingServiceClient(httpClient, consoleUri)

	return characterClient, gameClient, userClient, rankingClient
}

func (b *Backend) Start() error {
	slog.Info("Starting backend")
	// b.CommandServerSideChannel()

	// Listen for incoming connections.
	listener, err := net.Listen("tcp4", b.Addr)
	if err != nil {
		slog.Error("Could not start listening on port 6112", logging.Error(err))
		return err
	}
	b.listener = listener

	slog.Info("Backend listening", "addr", b.listener.Addr(), "mode", b.CreateProxy.Mode())
	return nil
}

func (b *Backend) Shutdown() {
	slog.Info("Shutting down the backend...")

	// Close all open connections
	b.ConnectedSessions.Range(func(k, v any) bool {
		session := v.(*bsession.Session)

		// TODO: Send a system message "(system) The server is going to close in less than 30 seconds"
		_ = session.SendToGame(
			packet.ReceiveMessage,
			NewGlobalMessage("system-info", "The server is going to shut down..."))

		// TODO: Send a packet to trigger stats saving
		// TODO: Send a system message "(system): Your stats were saving, your game client might close in the next 10 seconds"

		// TODO: Send a packet to close the connection (malformed 255-21?)
		if err := session.Conn.Close(); err != nil {
			slog.Error("Could not close session", logging.Error(err), "session", session.ID)
		}

		return true
	})

	if b.listener != nil {
		if err := b.listener.Close(); err != nil {
			slog.Warn("Could not close listener", logging.Error(err))
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
			slog.Warn("Error, when accepting incoming connection", logging.Error(err))
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
					logging.Error(err))
			}
		}()
	}
}

func (b *Backend) handleClient(conn net.Conn) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	session, err := b.handshake(conn)
	if err != nil {
		if err2 := conn.Close(); err2 != nil {
			slog.Error("Could not close connection in handshake", logging.Error(err))
			return errors.Join(err, err2)
		}
		if err == io.EOF {
			return nil
		}
		slog.Warn("Handshake failed", logging.Error(err))
		return err
	}
	defer func() {
		if err := b.CloseSession(session); err != nil {
			slog.Warn("Close session failed", logging.Error(err))
		}
	}()

	for {
		if err := b.handleCommands(ctx, session); err != nil {
			slog.Warn("Command failed", logging.Error(err))
			return err
		}
	}
}

// type ConfigOption func(backend *Backend) error
// []ConfigOption,

func GetMetadata(ctx context.Context, consoleAddr string) (*model.WellKnown, error) {
	httpClient := &http.Client{Timeout: 3 * time.Second}
	
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/.well-known/console.json", consoleAddr), nil)
	if err != nil {
		return nil, err
	}
	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("incorrect http-status code: %d", res.StatusCode)
	}

	var resp model.WellKnown
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
