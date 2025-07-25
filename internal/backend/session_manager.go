package backend

import (
	"context"
	"log/slog"
	"net"
	"sync"

	multiv1 "github.com/dimspell/gladiator/gen/multi/v1"
	"github.com/dimspell/gladiator/gen/multi/v1/multiv1connect"
	"github.com/dimspell/gladiator/internal/app/logger/logging"
	"github.com/dimspell/gladiator/internal/backend/bsession"
	"github.com/dimspell/gladiator/internal/backend/packet"
	"github.com/dimspell/gladiator/internal/backend/proxy"
	"github.com/dimspell/gladiator/internal/model"
)

type ProxyFactory interface {
	Create(session *bsession.Session, gameClient multiv1connect.GameServiceClient) proxy.ProxyClient
	Mode() model.RunMode
}

type SessionManager struct {
	ConnectedSessions *sync.Map
	ProxyFactory      ProxyFactory
	GameClient        multiv1connect.GameServiceClient
}

func NewSessionManager(proxyFactory ProxyFactory, gameClient multiv1connect.GameServiceClient) *SessionManager {
	return &SessionManager{
		ConnectedSessions: new(sync.Map),
		ProxyFactory:      proxyFactory,
		GameClient:        gameClient,
	}
}

func (s *SessionManager) Add(tcpConn net.Conn) *bsession.Session {
	session := bsession.NewSession(tcpConn)
	session.Proxy = s.ProxyFactory.Create(session, s.GameClient)

	s.ConnectedSessions.Store(session.ID, session)
	return session
}

func (s *SessionManager) Remove(session *bsession.Session) {
	slog.Info("Session closed", "session", session.ID)

	if session.Proxy != nil {
		session.Proxy.Close()
	}

	session.StopObserver()
	s.ConnectedSessions.Delete(session.ID)
	session = nil
}

func (s *SessionManager) RemoveAll() {
	// Close all open connections
	s.ConnectedSessions.Range(func(k, v any) bool {
		session := v.(*bsession.Session)

		// TODO: Send a system message "(system) The server is going to close in less than 30 seconds"
		_ = session.SendToGame(
			packet.ReceiveMessage,
			packet.NewGlobalMessage("system-info", "The server is going to shut down..."))

		// TODO: Send a packet to trigger stats saving
		// TODO: Send a system message "(system): Your stats were saving, your game client might close in the next 10 seconds"

		// TODO: Send a packet to close the connection (malformed 255-21?)
		if err := session.Conn.Close(); err != nil {
			slog.Error("Could not close session", logging.Error(err), "session", session.ID)
		}

		return true
	})
}

func (b *Backend) AddSession(tcpConn net.Conn) *bsession.Session {
	return b.SessionManager.Add(tcpConn)
}

func (b *Backend) CloseSession(session *bsession.Session) {
	b.SessionManager.Remove(session)
}

func (b *Backend) ConnectToLobby(ctx context.Context, user *multiv1.User, session *bsession.Session) error {
	return session.ConnectOverWebsocket(ctx, user, b.SignalServerURL)
}
