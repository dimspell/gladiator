package signalserver

import (
	"net/http/httptest"
	"net/url"
	"testing"

	"go.uber.org/goleak"
	"golang.org/x/net/websocket"
)

func TestHandshake(t *testing.T) {
	defer goleak.VerifyNone(t)

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

	// Connect to the signaling server
	ws, err := websocket.Dial(wsURI.String(), "", ts.URL)
	if err != nil {
		t.Error(err)
		return
	}
	defer ws.Close()

	// Send "hello" message to the signaling server
	req := &Message{
		From:    userId,
		Type:    HandshakeRequest,
		Content: roomName,
	}
	if _, err := ws.Write(req.ToCBOR()); err != nil {
		t.Error(err)
		return
	}

	// Wait for the response
	buf := make([]byte, 128)
	n, err := ws.Read(buf)
	if err != nil {
		t.Error(err)
		return
	}

	t.Log(string(buf[:n]))
}
