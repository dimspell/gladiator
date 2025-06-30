package backend

import (
	"context"
	"log/slog"
	"net"
	"os"
	"testing"
	"time"

	v1 "github.com/dimspell/gladiator/gen/multi/v1"
	"github.com/dimspell/gladiator/internal/app/logger"
	"github.com/dimspell/gladiator/internal/backend/bsession"
	"github.com/dimspell/gladiator/internal/backend/proxy/direct"
	"github.com/dimspell/gladiator/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestBackend_RegisterNewObserver(t *testing.T) {
	logger.SetPlainTextLogger(os.Stderr, slog.LevelDebug)
	// defer goleak.VerifyNone(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	b, _, _ := helperNewBackend(t)
	conn := &mockConn{RemoteAddress: &net.IPAddr{IP: net.ParseIP("127.0.0.1")}}
	session := &bsession.Session{ID: "TEST", Conn: conn, UserID: 2137, Username: "JP"}

	if err := b.ConnectToLobby(ctx, &v1.User{UserId: session.UserID, Username: session.Username}, session); err != nil {
		t.Error(err)
		return
	}
	if len(conn.Written) != 0 {
		t.Error("expected no data written to the backend client")
	}
	session.Stop()
}

func TestBackend_UpdateCharacterInfo(t *testing.T) {
	logger.SetPlainTextLogger(os.Stderr, slog.LevelDebug)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	b, _, cs := helperNewBackend(t)
	conn := &mockConn{}
	session := &bsession.Session{ID: "TEST", Conn: conn, UserID: 2137, Username: "JP"}

	// Authentication
	if err := b.ConnectToLobby(ctx, &v1.User{UserId: session.UserID, Username: session.Username}, session); err != nil {
		t.Error(err)
		return
	}

	// Character selection
	session.CharacterID = 4
	session.ClassType = model.ClassTypeMage
	session.Proxy = &direct.LAN{}

	// First call, after approved character selection
	if err := session.JoinLobby(ctx); err != nil {
		t.Error(err)
		return
	}
	if err := b.RegisterNewObserver(ctx, session); err != nil {
		t.Error(err)
		return
	}
	defer session.Stop()

	us, ok := cs.Multiplayer.GetUserSession(2137)
	if !ok {
		t.Error("expected user session connected to the lobby")
		return
	}

	assert.Equal(t, int64(2137), us.UserID)
	assert.Equal(t, int64(2137), us.User.UserID)
	assert.Equal(t, "dev", us.User.Version)
	assert.Equal(t, "JP", us.User.Username)
	assert.Equal(t, int64(4), us.Character.CharacterID)
	assert.Equal(t, byte(0x3), us.Character.ClassType)

	if err := session.SendChatMessage(ctx, "Hello, World!"); err != nil {
		t.Error(err)
		return
	}

	time.Sleep(1 * time.Second)
	cs.Multiplayer.DebugState()
}
