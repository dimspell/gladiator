package console

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dimspell/gladiator/console/database"
	"github.com/dimspell/gladiator/gen/multi/v1/multiv1connect"
	"github.com/dimspell/gladiator/model"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/cors"
	slogchi "github.com/samber/slog-chi"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

type Console struct {
	Addr               string
	RunMode            model.RunMode
	DB                 *database.SQLite
	CORSAllowedOrigins []string
}

func NewConsole(db *database.SQLite, addr string) *Console {
	return &Console{
		Addr:               addr,
		DB:                 db,
		CORSAllowedOrigins: []string{"*"},
	}
}

type Option func(*Console) error

// TODO: For production replace it with []string{"https://dispel-multi.net"}
func WithCORSAllowedOrigins(allowedOrigins []string) Option {
	return func(c *Console) error {
		c.CORSAllowedOrigins = allowedOrigins
		return nil
	}
}

func (c *Console) HttpRouter() http.Handler {
	mux := chi.NewRouter()

	mux.Use(middleware.Recoverer)
	mux.Use(middleware.Throttle(100))
	mux.Use(middleware.Timeout(5 * time.Second))

	{ // Setup meta routes (readiness, liveness, metrics etc.)
		mux.Get("/_health", func(w http.ResponseWriter, r *http.Request) {
			if err := c.DB.Ping(); err != nil {
				w.WriteHeader(http.StatusServiceUnavailable)
				renderJSON(w, r, map[string]string{
					"status":    "ERROR",
					"component": "database",
					"error":     err.Error(),
				})
				return
			}

			w.WriteHeader(http.StatusOK)
			renderJSON(w, r, map[string]string{"status": "OK"})
		})
		// mux.Get("/_metrics", promhttp.Handler().ServeHTTP)
	}

	{ // Setup routes used by the launcher
		wellKnown := chi.NewRouter()
		wellKnown.Use(slogchi.New(slog.Default()))
		wellKnown.Use(cors.New(cors.Options{
			AllowedOrigins:   c.CORSAllowedOrigins,
			AllowCredentials: false,
			Debug:            false,
			AllowedMethods:   []string{http.MethodGet},
			AllowedHeaders:   []string{"Content-Type"},
			MaxAge:           7200,
		}).Handler)

		wellKnown.Get("/console.json", c.WellKnownInfo())
		mux.Mount("/.well-known/", wellKnown)
	}

	{ // Setup gRPC routes for the backend
		api := chi.NewRouter()
		api.Use(slogchi.New(slog.Default()))
		api.Use(cors.New(cors.Options{
			AllowedOrigins:   c.CORSAllowedOrigins,
			AllowCredentials: false,
			Debug:            false,
			AllowedMethods: []string{
				http.MethodGet,
				http.MethodPost,
			},
			AllowedHeaders: []string{
				"Content-Type",
				"Connect-Protocol-Version",
				"Connect-Timeout-Ms",
				"Grpc-Timeout",
				"X-Grpc-Web",
				"X-User-Agent",
			},
			ExposedHeaders: []string{
				"Grpc-Status",
				"Grpc-Message",
				"Grpc-Status-Details-Bin",
			},
			MaxAge: 7200,
		}).Handler)

		api.Mount(multiv1connect.NewCharacterServiceHandler(&characterServiceServer{c.DB}))
		api.Mount(multiv1connect.NewGameServiceHandler(&gameServiceServer{c.DB}))
		api.Mount(multiv1connect.NewUserServiceHandler(&userServiceServer{c.DB}))
		api.Mount(multiv1connect.NewRankingServiceHandler(&rankingServiceServer{c.DB}))
		mux.Mount("/grpc/", http.StripPrefix("/grpc", api))
	}

	return mux
}

func (c *Console) Handlers() (start GracefulFunc, shutdown GracefulFunc) {
	httpServer := &http.Server{
		Addr:         c.Addr,
		Handler:      h2c.NewHandler(c.HttpRouter(), &http2.Server{}),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	start = func(ctx context.Context) error {
		slog.Info("Configured console server", "addr", c.Addr)
		return httpServer.ListenAndServe()
	}

	shutdown = func(ctx context.Context) error {
		slog.Info("Started shutting down the console server")
		if err := httpServer.Shutdown(ctx); err != nil {
			slog.Error("Failed shutting down the console server", "error", err)
			return err
		}
		slog.Info("Shut down the console server")
		return nil
	}

	return start, shutdown
}

type GracefulFunc func(context.Context) error

func (c *Console) Graceful(ctx context.Context, start GracefulFunc, shutdown GracefulFunc) error {
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

		timer, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := shutdown(timer); err != nil {
			errChan <- err
			return
		}

		errChan <- nil
	}()

	// Start the server
	if err := start(ctx); !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return <-errChan
}

func (c *Console) WellKnownInfo() http.HandlerFunc {
	if c.RunMode == "" {
		c.RunMode = model.RunModeLAN
	}
	return func(w http.ResponseWriter, r *http.Request) {
		slog.Info("Serving well-known info", "caller_ip", r.RemoteAddr, "caller_agent", r.UserAgent())

		renderJSON(w, r, model.WellKnown{
			Version:  "dev",
			Protocol: "http",
			Addr:     c.Addr,
			RunMode:  c.RunMode.String(),
			Caller: model.WellKnownCaller{
				Addr: r.RemoteAddr,
			},
		})
	}
}
