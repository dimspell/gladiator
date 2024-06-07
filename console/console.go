package console

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"connectrpc.com/connect"
	"connectrpc.com/otelconnect"
	"github.com/dispel-re/dispel-multi/console/database"
	"github.com/dispel-re/dispel-multi/gen/multi/v1/multiv1connect"
	"github.com/dispel-re/dispel-multi/model"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/riandyrn/otelchi"
	"github.com/rs/cors"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

var (
	healthy int32
)

type Console struct {
	Addr string
	DB   *database.Queries

	closeFunc     func(ctx context.Context) error
	StartupChan   chan bool
	ReadinessChan chan bool
	LivenessChan  chan bool

	CORSAllowedOrigins []string
}

func NewConsole(db *database.Queries, addr string) *Console {
	return &Console{
		Addr:               addr,
		DB:                 db,
		StartupChan:        make(chan bool),
		ReadinessChan:      make(chan bool),
		LivenessChan:       make(chan bool),
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
	mux.Use(middleware.DefaultLogger)
	mux.Use(middleware.Throttle(100))
	mux.Use(middleware.Timeout(5 * time.Second))
	mux.Use(otelchi.Middleware("console", otelchi.WithChiRoutes(mux)))

	{ // Setup meta routes (readiness, liveness, metrics etc.)
		mux.Get("/_health", func(w http.ResponseWriter, r *http.Request) {
			if atomic.LoadInt32(&healthy) == 0 {
				renderJSON(w, r, map[string]string{"status": "OK"})
				return
			}
			w.WriteHeader(http.StatusServiceUnavailable)
		})
		mux.Get("/_metrics", promhttp.Handler().ServeHTTP)
	}

	{ // Setup routes used by the launcher
		wellKnown := chi.NewRouter()
		wellKnown.Use(cors.New(cors.Options{
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

		wellKnown.Get("/dispel-multi.json", func(w http.ResponseWriter, r *http.Request) {
			renderJSON(w, r, model.WellKnown{
				ZeroTier: model.ZeroTier{Enabled: false},
			})
		})
		mux.Mount("/.well-known/", wellKnown)
	}

	{ // Setup gRPC routes for the backend
		api := chi.NewRouter()
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

		interceptors := connect.WithInterceptors(otelconnect.NewInterceptor())
		api.Mount(multiv1connect.NewCharacterServiceHandler(&characterServiceServer{DB: c.DB}, interceptors))
		api.Mount(multiv1connect.NewGameServiceHandler(&gameServiceServer{DB: c.DB}, interceptors))
		api.Mount(multiv1connect.NewUserServiceHandler(&userServiceServer{DB: c.DB}, interceptors))
		api.Mount(multiv1connect.NewRankingServiceHandler(&rankingServiceServer{DB: c.DB}, interceptors))
		mux.Mount("/grpc/", http.StripPrefix("/grpc", api))
	}

	return mux
}

func (c *Console) Serve(ctx context.Context) error {
	httpServer := &http.Server{
		Addr:         c.Addr,
		Handler:      h2c.NewHandler(c.HttpRouter(), &http2.Server{}),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	start := func(ctx context.Context) error {
		// TODO: Set readiness, startup, liveness probe
		atomic.StoreInt32(&healthy, 0)
		slog.Info("Configured console server", "addr", c.Addr)

		return httpServer.ListenAndServe()
	}

	stop := func(ctx context.Context) error {
		return httpServer.Shutdown(ctx)
	}
	c.closeFunc = stop

	// return c.graceful(ctx, start, stop)
	return start(ctx)
}

func (c *Console) Stop(ctx context.Context) error {
	if c.closeFunc != nil {
		return c.closeFunc(ctx)
	}
	return nil
}

type gracefulFunc func(context.Context) error

func (c *Console) graceful(ctx context.Context, start gracefulFunc, shutdown gracefulFunc) error {
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
