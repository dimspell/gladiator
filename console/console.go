package console

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
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/dispel-re/dispel-multi/gen/multi/v1/multiv1connect"
)

type Console struct {
	DB      *database.Queries
	Backend *backend.Backend
}

func NewConsole(db *database.Queries, b *backend.Backend) *Console {
	return &Console{
		DB:      db,
		Backend: b,
	}
}

func (c *Console) Serve(ctx context.Context, consoleAddr, backendAddr string) error {
	mux := http.NewServeMux()

	mux.Handle(multiv1connect.NewCharacterServiceHandler(&characterServiceServer{DB: c.DB}))
	mux.Handle(multiv1connect.NewGameServiceHandler(&gameServiceServer{DB: c.DB}))
	mux.Handle(multiv1connect.NewUserServiceHandler(&userServiceServer{DB: c.DB}))
	mux.Handle(multiv1connect.NewRankingServiceHandler(&rankingServiceServer{DB: c.DB}))

	server := &http.Server{
		Addr:    consoleAddr,
		Handler: h2c.NewHandler(mux, &http2.Server{}),
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

func (c *Console) graceful(ctx context.Context, start func() error, shutdown func(context.Context) error) error {
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
