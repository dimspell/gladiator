package lobby

import (
	"context"
	"log/slog"
	"net/http"
	"time"

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
