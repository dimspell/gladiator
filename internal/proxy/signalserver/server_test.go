package signalserver

import (
	"context"
	"log/slog"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/lmittmann/tint"
	"go.uber.org/goleak"
)

func TestHandshake(t *testing.T) {
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

	defer goleak.VerifyNone(t)

	DefaultCodec = NewJSONCodec()

	// Start the signaling server
	h, err := NewServer()
	if err != nil {
		t.Error(err)
		return
	}
	ts := httptest.NewServer(h)
	defer ts.Close()

	wsURI, _ := url.Parse(ts.URL)
	wsURI.Scheme = "ws"

	const (
		userId   = "userTester"
		roomName = "testRoom"
	)
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
	if err != nil {
		t.Error(err)
		return
	}
	defer ws.CloseNow()

	// Send "hello" message to the signaling server
	req := &Message{
		From:    userId,
		Type:    HandshakeRequest,
		Content: roomName,
	}
	if err := ws.Write(ctx, websocket.MessageText, req.Encode()); err != nil {
		t.Error(err)
		return
	}

	// Wait for the response
	_, data, err := ws.Read(ctx)
	if err != nil {
		t.Error(err)
		return
	}
	t.Log(string(data))

	if err := ws.Close(websocket.StatusNormalClosure, "fin"); err != nil {
		t.Error(err)
		return
	}
}
