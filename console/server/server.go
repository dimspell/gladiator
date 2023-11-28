package server

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dispel-re/dispel-multi/backend"
	"github.com/dispel-re/dispel-multi/internal/database"
	"github.com/go-chi/chi/v5"
)

type ConsoleServer struct {
	DB      *database.Queries
	Backend *backend.Backend
}

func (c *ConsoleServer) Serve(ctx context.Context, consoleAddr, backendAddr string) error {
	server := &http.Server{
		Addr:    consoleAddr,
		Handler: c.Routing(),
	}

	start := func() error {
		go func() {
			c.Backend.Listen(backendAddr)
		}()

		// TODO: Set readiness, startup, liveness probe
		return server.ListenAndServe()
	}

	stop := func(ctx context.Context) error {
		c.Backend.Shutdown(ctx)

		return server.Shutdown(ctx)
	}

	return c.graceful(ctx, start, stop)
}

func (c *ConsoleServer) graceful(ctx context.Context, start func() error, shutdown func(context.Context) error) error {
	var (
		stopChan = make(chan os.Signal, 1)
		errChan  = make(chan error, 1)
	)

	// Set up the graceful shutdown handler (traps SIGINT and SIGTERM)
	go func() {
		signal.Notify(stopChan, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

		select {
		case <-stopChan:
		case <-ctx.Done():
		}

		timer, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := shutdown(timer); err != nil {
			errChan <- err
			return
		}

		errChan <- nil
	}()

	// Start the server
	if err := start(); !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return <-errChan
}

func (c *ConsoleServer) Routing() http.Handler {
	r := chi.NewRouter()

	r.Get("/startupProbe", c.StartupProbe)
	r.Get("/readinessProbe", c.ReadinessProbe)
	r.Get("/livenessProbe", c.LivenessProbe)
	r.Get("/metrics", c.Metrics().ServeHTTP)

	r.Route("/channels", func(r chi.Router) {
		r.Get("/", c.ListChannels)
		// r.Post("/", c.AddChannel)

		// r.Delete("/{channel}", c.DeleteChannel)
		// r.Get("/{channel}/chat", c.GetChannelChat)
	})

	r.Route("/sessions", func(r chi.Router) {
		// r.Get("/", c.ListConnections)

		// r.Get("/{connection}", nil) // Get connection details - user, his inventory, chars..
		// r.Delete("/{connection}", nil) // Close connection

		// r.Get("/{connection}/packets", nil)
		// r.Post("/{connection}/send-message", nil)
		// r.Post("/{connection}/send-packet", nil)
	})

	r.Route("/games", func(r chi.Router) {
		// r.Get("/", c.ListGames)

		// r.Get("/{game}", c.Get)
		// r.Delete("/{game}", c.DeleteGame)
	})

	r.Route("/user", func(r chi.Router) {
		r.Get("/{userId}/characters", c.ListCharacters)

		// r.Post("/character", c.AddCharacter)
		// r.Post("/import-character", c.ImportCharacterFromSav)
	})

	return r
}
