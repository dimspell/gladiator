package backend

import (
	"context"
	"errors"
	"log/slog"
	"net"

	"github.com/coder/websocket"
	multiv1 "github.com/dimspell/gladiator/gen/multi/v1"
	"github.com/dimspell/gladiator/internal/app/logger/logging"
	"github.com/dimspell/gladiator/internal/backend/bsession"
	"github.com/dimspell/gladiator/internal/backend/proxy"
	"github.com/dimspell/gladiator/internal/model"
)

func (b *Backend) AddSession(tcpConn net.Conn) *bsession.Session {
	slog.Debug("New session")

	session := bsession.NewSession(tcpConn)
	session.Proxy = b.CreateProxy.Create(session)

	b.ConnectedSessions.Store(session.ID, session)
	return session
}

func (b *Backend) CloseSession(session *bsession.Session) error {
	slog.Info("Session closed", "session", session.ID)

	if session.Proxy != nil {
		session.Proxy.Close()
	}
	session.StopObserver()

	b.ConnectedSessions.Delete(session.ID)

	session = nil
	return nil
}

func (b *Backend) ConnectToLobby(ctx context.Context, user *multiv1.User, session *bsession.Session) error {
	return session.ConnectOverWebsocket(ctx, user, b.SignalServerURL)
}

func (b *Backend) RegisterNewObserver(ctx context.Context, session *bsession.Session) error {
	handlers := []proxy.MessageHandler{
		NewLobbyEventHandler(session).Handle,
		session.Proxy.Handle,
	}
	observe := func(ctx context.Context, wsConn *websocket.Conn) {
		for {
			if ctx.Err() != nil {
				return
			}

			// Read the broadcast and handle them as commands.
			p, err := session.ConsumeWebSocket(ctx)
			if err != nil {
				if errors.Is(err, context.Canceled) {
					return
				}
				slog.Error("Error reading from WebSocket", "session", session.ID, logging.Error(err))
				return
			}

			// slog.Debug("Signal from lobby", "type", et.String(), "session", session.ID, "payload", string(p[1:]))

			// TODO: Register handlers and handle them here.
			for _, handleFn := range handlers {
				if err := handleFn(ctx, p); err != nil {
					slog.Error("Error handling message", "session", session.ID, logging.Error(err))
					return
				}
			}
		}
	}
	return session.StartObserver(ctx, observe)
}

type Proxy interface {
	// Create creates a proxy for the session
	Create(session *bsession.Session) proxy.ProxyClient
	Mode() model.RunMode
}
