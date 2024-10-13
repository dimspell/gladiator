package lobby

import (
	"context"
	"log/slog"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/dimspell/gladiator/internal/wire"
	"github.com/lmittmann/tint"
)

func helperDebugPrettyPrint(tb testing.TB) {
	tb.Helper()

	slog.SetDefault(slog.New(
		tint.NewHandler(
			os.Stderr,
			&tint.Options{
				Level:      slog.LevelDebug,
				TimeFormat: time.TimeOnly,
				AddSource:  true,
			},
		),
	))
}

// var _ ConnReadWriter = (*mockWebSocket)(nil)
//
// type mockWebSocket struct {
// 	id     string
// 	writer *bytes.Buffer
// }
//
// func (s *mockWebSocket) CloseNow() error {
// 	log.Println("Closing mock websocket, id:", s.id)
// 	return nil
// }
//
// func newMockWebSocket(id string) *mockWebSocket {
// 	return &mockWebSocket{id: id, writer: bytes.NewBuffer(nil)}
// }
//
// func (s *mockWebSocket) Read(ctx context.Context) (websocket.MessageType, []byte, error) {
// 	if err := ctx.Err(); err != nil {
// 		return 0, nil, err
// 	}
// 	return websocket.MessageText, compose(icesignal.Join, icesignal.Message{
// 		To:      "",
// 		From:    "",
// 		Type:    icesignal.Join,
// 		Content: "Hello",
// 	}), nil
// }
//
// func (s *mockWebSocket) Write(ctx context.Context, typ websocket.MessageType, p []byte) error {
// 	_, err := s.writer.Write(p)
// 	return err
// }

func wsConnect(userId, tsURL string) (*websocket.Conn, error) {
	const roomName = "DISPEL"

	wsURI, _ := url.Parse(tsURL)
	wsURI.Scheme = "ws"

	v := wsURI.Query()
	v.Set("userID", userId)
	v.Set("roomName", roomName)
	wsURI.RawQuery = v.Encode()

	// Give 3 minutes for the whole test to finish
	ctx, cancel := context.WithTimeout(context.TODO(), 3*time.Second)
	defer cancel()

	// Connect to the signaling server
	ws, _, err := websocket.Dial(ctx, wsURI.String(), &websocket.DialOptions{
		Subprotocols: []string{"signalserver"},
	})
	return ws, err
}

func TestLobby(t *testing.T) {
	helperDebugPrettyPrint(t)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	lobby := NewLobby(ctx)
	ts := httptest.NewServer(lobby.Handler())
	defer ts.Close()

	wsFirst, err := wsConnect("first", ts.URL)
	if err != nil {
		t.Error(err)
		return
	}

	wsSecond, err := wsConnect("second", ts.URL)
	if err != nil {
		t.Error(err)
		return
	}

	go func() {
		for {
			slog.Info("first")
			_, p, err := wsFirst.Read(ctx)
			if err != nil {
				t.Log(err)
				return
			}
			slog.Info(string(p))
		}
	}()

	go func() {
		for {
			slog.Info("second")
			_, p, err := wsSecond.Read(ctx)
			if err != nil {
				t.Log(err)
				return
			}
			slog.Info(string(p))
		}
	}()

	if err := wsFirst.Write(ctx, websocket.MessageText, compose(wire.Chat, wire.Message{
		From:    "first",
		Content: "Hello, World!",
	})); err != nil {
		t.Error(err)
		return
	}

	if err := wsSecond.Write(ctx, websocket.MessageText, compose(wire.Chat, wire.Message{
		From:    "second",
		Content: "Welcome!",
	})); err != nil {
		t.Error(err)
		return
	}

	time.Sleep(1 * time.Second)
}
