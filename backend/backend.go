package backend

import (
	"context"
	"net"

	"github.com/dispel-re/dispel-multi/database/memory"
	"github.com/dispel-re/dispel-multi/model"
)

type Backend struct {
	DB *memory.Memory

	Sessions map[string]*model.Session
}

func NewBackend(db *memory.Memory) *Backend {
	return nil
}

func (b *Backend) Shutdown(ctx context.Context) {
	// Close all open connections
	for _, session := range b.Sessions {
		session.Conn.Close()
	}

	// TODO: Send a system message "(system) The server is going to close in less than 30 seconds"
	// TODO: Send a packet to trigger stats saving
	// TODO: Send a system message "(system): Your stats were saving, your game client might close in next 10 seconds"
	// TODO: Send a packet to close the connection (malformed 255-21?)
}

func (b *Backend) NewSession(conn net.Conn) *model.Session {
	session := &model.Session{Conn: conn}
	b.Sessions["id"] = session
	return session
}

func (b *Backend) CloseSession(session *model.Session) error {
	// TODO: wrap all errors
	_, ok := b.Sessions["id"]
	if ok {
		delete(b.Sessions, "id")
	}

	if session.Conn != nil {
		session.Conn.Close()
	}
	return nil
}
