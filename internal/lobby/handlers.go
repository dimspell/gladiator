package lobby

import (
	"log/slog"
	"net/http"

	"github.com/coder/websocket"
)

const (
	WebsocketSupportedSubProtocol = "signalserver"
)

func (lb *Lobby) HandleGetUsers(w http.ResponseWriter, r *http.Request) {
	// lb.Multiplayer.Presence.List()
}

func (lb *Lobby) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	roomName := params.Get("roomName")
	userID := params.Get("userID")
	// version := params.Get("version")

	// FIXME: Improve validation.
	// For now only the DISPEL channel is available to use.
	if roomName != "DISPEL" || userID == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		Subprotocols: []string{WebsocketSupportedSubProtocol},
	})
	if err != nil {
		slog.Error("Could not accept the connection",
			"error", err,
			"origin", r.Header.Get("Origin"),
			"userId", userID,
			"roomName", roomName)
		return
	}
	defer conn.CloseNow()

	if conn.Subprotocol() != WebsocketSupportedSubProtocol {
		_ = conn.Close(websocket.StatusPolicyViolation, "client must speak the right subprotocol")
		return
	}

	if err := lb.Multiplayer.HandleSession(r.Context(), NewUserSession(userID, conn)); err != nil {
		return
	}
}
