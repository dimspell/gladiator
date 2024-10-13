package wire

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"time"

	"github.com/coder/websocket"
)

func Connect(ctx context.Context, wsURL string, user User) (*websocket.Conn, error) {
	// Parse the provided signaling server URL
	u, err := url.Parse(wsURL)
	if err != nil {
		return nil, err
	}

	// Set query parameters.
	v := u.Query()
	v.Set("userID", user.UserID)
	v.Set("roomName", "DISPEL")
	u.RawQuery = v.Encode()

	// Encode the URL to the WebSocket with the query parameters.
	wsURL = u.String()
	slog.Debug("Connecting to the signaling server", "userID", user.UserID, "url", wsURL)

	// Give 5 seconds to establish WebSocket connection.
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Connect to the signaling server and return it.
	// TODO: Add HTTP Authorization header with bearer token
	ws, _, err := websocket.Dial(ctx, u.String(), &websocket.DialOptions{
		Subprotocols: []string{SupportedRealm},
	})
	if err != nil {
		return nil, err
	}

	// Send player information.
	// TODO: that data could be set in the JWT header
	if err := ws.Write(ctx, websocket.MessageText, ComposeTyped(Hello, MessageContent[User]{From: user.UserID, Content: user})); err != nil {
		return nil, err
	}

	// Expect to receive the welcome message.
	_, p, err := ws.Read(ctx)
	if err != nil {
		return nil, err
	}
	if len(p) != 1 || EventType(p[0]) != Welcome {
		return nil, fmt.Errorf("expected welcome message, got: %s", string(p))
	}

	return ws, nil
}
