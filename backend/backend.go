package backend

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net"
	"net/http"
	"time"

	"github.com/dimspell/gladiator/gen/multi/v1/multiv1connect"
	"github.com/dimspell/gladiator/internal/proxy"
	"github.com/dimspell/gladiator/model"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Backend struct {
	Addr        string
	MyIPAddress string

	Sessions       map[string]*model.Session
	PacketLogger   *slog.Logger
	SessionCounter uint64

	Proxy     proxy.Proxy
	EventChan chan uint8
	listener  net.Listener

	characterClient multiv1connect.CharacterServiceClient
	gameClient      multiv1connect.GameServiceClient
	userClient      multiv1connect.UserServiceClient
	rankingClient   multiv1connect.RankingServiceClient
}

func NewBackend(backendAddr, consoleAddr, myIPAddress string) *Backend {
	if myIPAddress == "" {
		myIPAddress = "127.0.0.1"
	}

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

	// proxy.NewPeerToPeer(ctx)

	return &Backend{
		Addr:        backendAddr,
		MyIPAddress: myIPAddress,
		Sessions:    make(map[string]*model.Session),
		Proxy:       proxy.NewLAN(),

		characterClient: multiv1connect.NewCharacterServiceClient(httpClient, consoleUri),
		gameClient:      multiv1connect.NewGameServiceClient(httpClient, consoleUri),
		userClient:      multiv1connect.NewUserServiceClient(httpClient, consoleUri),
		rankingClient:   multiv1connect.NewRankingServiceClient(httpClient, consoleUri),
	}
}

type CustomPacket struct {
	PacketID uint8  `json:"packet_id"`
	Data     []byte `json:"data"`
}

func (b *Backend) CommandServerSideChannel() {
	go func() {
		mux := chi.NewRouter()
		mux.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		mux.Post("/send", func(w http.ResponseWriter, r *http.Request) {
			defer r.Body.Close()
			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			var packet CustomPacket
			if err := json.Unmarshal(body, &packet); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			// session, ok := srv.Sessions[r.Header.Get("session-id")]
			for _, session := range b.Sessions {
				// if session.Conn.RemoteAddr().String() == r.Header.Get("session-id") {
				// 	break
				// }
				if err := b.Send(session.Conn, PacketType(packet.PacketID), packet.Data); err != nil {
					slog.Warn("Could not send packet to client", "err", err)
				}
			}

			w.WriteHeader(http.StatusOK)
		})

		http.ListenAndServe("127.0.0.1:6110", mux)
	}()
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
	for _, session := range b.Sessions {
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
	}

	if b.listener != nil {
		if err := b.listener.Close(); err != nil {
			slog.Warn("Could not close listener", "err", err)
		}
		b.listener = nil
	}

	slog.Info("The backend is successfully shut down")
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

func (b *Backend) NewSession(conn net.Conn) *model.Session {
	if b.SessionCounter == math.MaxUint64 {
		b.SessionCounter = 0
	}
	b.SessionCounter++
	id := fmt.Sprintf("%s-%d", uuid.New().String(), b.SessionCounter)
	slog.Debug("New session", "session", id)

	session := &model.Session{
		Conn:           conn,
		ID:             id,
		LocalIpAddress: b.MyIPAddress,
	}
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

	// b.EventChan <- EventCloseConn
	session = nil
	return nil
}
