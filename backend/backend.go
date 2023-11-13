package backend

import (
	"context"
	"log/slog"
	"net"

	"github.com/dispel-re/dispel-multi/internal/database"
	"github.com/dispel-re/dispel-multi/model"
	"github.com/google/uuid"
)

type Backend struct {
	DB *database.Queries

	Sessions map[string]*model.Session
}

// func NewBackend(db *memory.Memory) *Backend {
func NewBackend(db *database.Queries) *Backend {
	return &Backend{
		DB:       db,
		Sessions: make(map[string]*model.Session),
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
