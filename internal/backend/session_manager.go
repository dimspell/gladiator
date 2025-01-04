package backend

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"

	"github.com/coder/websocket"
	multiv1 "github.com/dimspell/gladiator/gen/multi/v1"
	"github.com/dimspell/gladiator/internal/backend/bsession"
	"github.com/dimspell/gladiator/internal/backend/proxy"
)

func (b *Backend) AddSession(tcpConn net.Conn) *bsession.Session {
	slog.Debug("New session", "backend", fmt.Sprintf("%p", b.Proxy))

	session := bsession.NewSession(tcpConn)

	b.ConnectedSessions.Store(session.ID, session)
	return session
}

func (b *Backend) CloseSession(session *bsession.Session) error {
	slog.Info("Session closed", "session", session.ID)

	b.Proxy.Close(session)
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
		bsession.NewLobbyEventHandler(session),
		b.Proxy.ExtendWire(session),
	}
	observe := func(ctx context.Context, wsConn *websocket.Conn) {
		for {
			if ctx.Err() != nil {
				return
			}

			// Read the broadcast & handle them as commands.
			p, err := session.ConsumeWebSocket(ctx)
			if err != nil {
				if errors.Is(err, context.Canceled) {
					return
				}
				slog.Error("Error reading from WebSocket", "session", session.ID, "error", err)
				return
			}

			// slog.Debug("Signal from lobby", "type", et.String(), "session", session.ID, "payload", string(p[1:]))

			// TODO: Register handlers and handle them here.
			for _, handler := range handlers {
				if err := handler.Handle(ctx, p); err != nil {
					slog.Error("Error handling message", "session", session.ID, "error", err)
					return
				}
			}
		}
	}
	return session.StartObserver(ctx, observe)
}
