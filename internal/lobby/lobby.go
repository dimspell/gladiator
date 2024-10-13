package lobby

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/coder/websocket"
	"github.com/dimspell/gladiator/internal/wire"
	"github.com/go-chi/chi/v5"
)

type Lobby struct {
	Multiplayer *Multiplayer
}

func NewLobby(ctx context.Context) *Lobby {
	return &Lobby{
		Multiplayer: NewMultiplayer(ctx),
	}
}

func (lb *Lobby) Prepare(addr string) (start, shutdown func(ctx context.Context) error) {
	httpServer := &http.Server{
		Addr:         addr,
		Handler:      lb.Handler(),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	start = func(ctx context.Context) error {
		slog.Info("Lobby server is running on", "addr", addr)
		return httpServer.ListenAndServe()
	}

	shutdown = func(ctx context.Context) error {
		if err := httpServer.Shutdown(ctx); err != nil {
			slog.Error("Failed to shut down the lobby server", "error", err)
			return err
		}
		return nil
	}

	return start, shutdown
}

func (lb *Lobby) Handler() http.Handler {
	r := chi.NewRouter()

	r.Mount("/", http.HandlerFunc(lb.HandleWebSocket))

	return r
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
			"roomName", roomName)
		return
	}
	defer conn.CloseNow()

	if conn.Subprotocol() != wire.SupportedRealm {
		_ = conn.Close(websocket.StatusPolicyViolation, "client must speak the right subprotocol")
		return
	}

	if err := lb.Multiplayer.HandleSession(r.Context(), NewUserSession(userID, conn)); err != nil {
		return
	}
}
