package console

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/coder/websocket"
	"github.com/dimspell/gladiator/internal/wire"
)

func (c *Console) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	channelName := params.Get("channelName")
	userID, err := strconv.ParseInt(params.Get("userID"), 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	// version := params.Get("version")

	// FIXME: Improve validation.
	// For now only the DISPEL channel is available to use.
	if channelName != "DISPEL" || userID == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	version := r.Header.Get("X-Version")
	if version != wire.ProtoVersion {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		Subprotocols: []string{wire.SupportedRealm},
	})
	if err != nil {
		slog.Error("Could not accept the connection",
			"error", err,
			"origin", r.Header.Get("Origin"),
			"userId", userID,
			"channelName", channelName)
		return
	}
	defer conn.CloseNow()

	if conn.Subprotocol() != wire.SupportedRealm {
		_ = conn.Close(websocket.StatusPolicyViolation, "client must speak the right subprotocol")
		return
	}

	if err := c.Multiplayer.HandleSession(r.Context(), NewUserSession(userID, conn)); err != nil {
		return
	}
}
