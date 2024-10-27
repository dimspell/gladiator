package backend

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	v1 "github.com/dimspell/gladiator/gen/multi/v1"
	"github.com/dimspell/gladiator/internal/app/logger"
	"github.com/dimspell/gladiator/internal/console"
	"github.com/dimspell/gladiator/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestBackend_RegisterNewObserver(t *testing.T) {
	logger.SetPlainTextLogger(os.Stderr, slog.LevelDebug)
	// defer goleak.VerifyNone(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	lb := console.Console{DB: nil, Multiplayer: console.NewMultiplayer()}
	ts := httptest.NewServer(http.HandlerFunc(lb.HandleWebSocket))
	defer ts.Close()

	b := &Backend{
		SignalServerURL: "ws://" + ts.URL[len("http://"):], // Skip schema prefix.
		Proxy:           NewLAN("127.0.0.1"),
	}
	conn := &mockConn{RemoteAddress: &net.IPAddr{IP: net.ParseIP("127.0.0.1")}}
	session := &Session{ID: "TEST", Conn: conn, UserID: 2137, Username: "JP"}

	if err := b.ConnectToLobby(ctx, &v1.User{UserId: session.UserID, Username: session.Username}, session); err != nil {
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

	lb := console.Console{DB: nil, Multiplayer: console.NewMultiplayer()}
	ts := httptest.NewServer(http.HandlerFunc(lb.HandleWebSocket))
	defer ts.Close()

	b := &Backend{
		SignalServerURL: "ws://" + ts.URL[len("http://"):], // Skip schema prefix.
		Proxy:           NewLAN("127.0.0.1"),
	}
	conn := &mockConn{RemoteAddress: &net.IPAddr{IP: net.ParseIP("127.0.0.1")}}
	session := &Session{ID: "TEST", Conn: conn, UserID: 2137, Username: "JP"}

	// Authentication
	if err := b.ConnectToLobby(ctx, &v1.User{UserId: session.UserID, Username: session.Username}, session); err != nil {
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

	us, ok := lb.Multiplayer.GetUserSession("2137")
	if !ok {
		t.Error("expected user session connected to the lobby")
		return
	}

	assert.Equal(t, "2137", us.UserID)
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
	lb.Multiplayer.DebugState()
}

// type mockWS struct {
//	Written []byte
// }
//
// func (m *mockWS) Write(ctx context.Context, messageType websocket.MessageType, p []byte) error {
//	m.Written = append(m.Written, p...)
//	return nil
// }
//
// func TestSession_SendChatMessage(t *testing.T) {
//	ctx := context.Background()
//	wsConn := &mockWS{}
//	session := &Session{UserID: 1, Username: "hello"}
//	text := "Hello, World!"
//	//if err := session.SendChatMessage(context.Background(), "Hello, World!"); err != nil {
//	//	t.Error(err)
//	//	return
//	//}
//
//	if err := wire.Write(ctx, wsConn, wire.ComposeTyped(
//		wire.Chat,
//		wire.MessageContent[wire.ChatMessage]{
//			From: fmt.Sprintf("%d", session.UserID),
//			Type: wire.Chat,
//			Content: wire.ChatMessage{
//				User: session.Username,
//				Text: text,
//			},
//		}),
//	); err != nil {
//		t.Error(err)
//		return
//	}
//
//	fmt.Println(wsConn.Written)
//	fmt.Println(string(wsConn.Written))
// }
