package backend

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"net"
	"time"

	"github.com/google/uuid"
)

type Session struct {
	ID          string
	UserID      int64
	CharacterID int64
	Username    string

	Conn net.Conn
}

func (b *Backend) AddSession(tcpConn net.Conn) *Session {
	if b.SessionCounter == math.MaxUint64 {
		b.SessionCounter = 0
	}
	b.SessionCounter++
	id := fmt.Sprintf("%s-%d", uuid.New().String(), b.SessionCounter)
	slog.Debug("New session", "session", id)

	session := &Session{
		Conn: conn,
		ID:   id,
	}
	b.ConnectedSessions.Store(session.ID, session)
	return session
}

func (b *Backend) CloseSession(session *Session) error {
	slog.Info("Session closed", "session", session.ID)

	b.Proxy.Close()

	b.ConnectedSessions.Delete(session.ID)

	if session.Conn != nil {
		_ = session.Conn.Close()
	}

	session = nil
	return nil
}
