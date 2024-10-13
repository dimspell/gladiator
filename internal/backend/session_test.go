package backend

import (
	"context"
	"log/slog"
	"net"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/dimspell/gladiator/internal/app/logger"
	"github.com/dimspell/gladiator/internal/lobby"
	"github.com/dimspell/gladiator/internal/model"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
)

func TestBackend_RegisterNewObserver(t *testing.T) {
	logger.SetPlainTextLogger(os.Stderr, slog.LevelDebug)
	defer goleak.VerifyNone(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	lb := lobby.NewLobby(ctx)
	ts := httptest.NewServer(lb.Handler())
	defer ts.Close()

	b := &Backend{
		SignalServerURL: "ws://" + ts.URL[len("http://"):], // Skip schema prefix.
	}
	conn := &mockConn{RemoteAddress: &net.IPAddr{IP: net.ParseIP("127.0.0.1")}}
	session := &Session{ID: "TEST", Conn: conn, UserID: 2137, Username: "JP"}

	if err := b.ConnectToLobby(ctx, session); err != nil {
		t.Error(err)
		return
	}
	if len(conn.Written) != 0 {
		t.Error("expected no data written to the backend client")
	}
	session.observerDone()
}

func TestBackend_UpdateCharacterInfo(t *testing.T) {
	logger.SetPlainTextLogger(os.Stderr, slog.LevelDebug)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	lb := lobby.NewLobby(ctx)
	ts := httptest.NewServer(lb.Handler())
	defer ts.Close()

	b := &Backend{
		SignalServerURL: "ws://" + ts.URL[len("http://"):], // Skip schema prefix.
	}
	conn := &mockConn{RemoteAddress: &net.IPAddr{IP: net.ParseIP("127.0.0.1")}}
	session := &Session{ID: "TEST", Conn: conn, UserID: 2137, Username: "JP"}

	// Authentication
	if err := b.ConnectToLobby(ctx, session); err != nil {
		t.Error(err)
		return
	}

	// Character selection
	session.CharacterID = 4
	session.ClassType = model.ClassTypeMage

	// First call, after approved character selection
	if err := b.JoinLobby(ctx, session); err != nil {
		t.Error(err)
		return
	}
	if err := b.RegisterNewObserver(ctx, session); err != nil {
		t.Error(err)
		return
	}
	defer session.observerDone()

	time.Sleep(1 * time.Second)
	lb.Multiplayer.DebugState()

	us, ok := lb.Multiplayer.GetUserSession("2137")
	if !ok {
		t.Error("expected user session connected to the lobby")
		return
	}

	assert.Equal(t, "2137", us.UserID)
	assert.Equal(t, "2137", us.User.UserID)
	assert.Equal(t, "latest", us.User.Version)
	assert.Equal(t, "JP", us.User.Username)
	assert.Equal(t, "4", us.Character.CharacterID)
	assert.Equal(t, 0x3, us.Character.ClassType)
}
