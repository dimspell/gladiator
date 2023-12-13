package console

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"connectrpc.com/connect"
	"connectrpc.com/otelconnect"
	"github.com/dispel-re/dispel-multi/backend"
	"github.com/dispel-re/dispel-multi/console/database"
	"github.com/dispel-re/dispel-multi/gen/multi/v1/multiv1connect"
	"github.com/dispel-re/dispel-multi/model"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/riandyrn/otelchi"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
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
	mux := chi.NewRouter()

	mux.Get("/_health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("."))
	})
	mux.Get("/_metrics", promhttp.Handler().ServeHTTP)
	mux.Get("/.well-known/dispel-multi.json", func(w http.ResponseWriter, r *http.Request) {
		resp := model.WellKnown{ZeroTier: model.ZeroTier{
			Enabled: false,
		}}
		document, _ := json.Marshal(resp)
		w.Write(document)
	})

	interceptors := connect.WithInterceptors(otelconnect.NewInterceptor())
	mux.Route("/grpc", func(r chi.Router) {
		r.Use(middleware.Timeout(5 * time.Second))

		r.Handle(multiv1connect.NewCharacterServiceHandler(&characterServiceServer{DB: c.DB}, interceptors))
		r.Handle(multiv1connect.NewGameServiceHandler(&gameServiceServer{DB: c.DB}, interceptors))
		r.Handle(multiv1connect.NewUserServiceHandler(&userServiceServer{DB: c.DB}, interceptors))
		r.Handle(multiv1connect.NewRankingServiceHandler(&rankingServiceServer{DB: c.DB}, interceptors))
	})

	mux.Use(middleware.Recoverer)
	mux.Use(middleware.Throttle(100))
	mux.Use(otelchi.Middleware("console", otelchi.WithChiRoutes(mux)))

	server := &http.Server{
		Addr:    consoleAddr,
		Handler: h2c.NewHandler(mux, &http2.Server{}),
	}

	start := func() error {
		if c.Backend != nil {
			go func() {
				c.Backend.Listen(backendAddr)
			}()
		}

		// TODO: Set readiness, startup, liveness probe
		return server.ListenAndServe()
	}

	stop := func(ctx context.Context) error {
		if c.Backend != nil {
			c.Backend.Shutdown(ctx)
		}

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
