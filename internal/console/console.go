package console

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dimspell/gladiator/gen/multi/v1/multiv1connect"
	"github.com/dimspell/gladiator/internal/app/logger/logging"
	"github.com/dimspell/gladiator/internal/console/database"
	"github.com/dimspell/gladiator/internal/metrics"
	"github.com/dimspell/gladiator/internal/model"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/cors"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func init() {
	metrics.InitConsole()
	metrics.InitRelay()
}

type Console struct {
	Config      *Config
	DB          *database.SQLite
	Multiplayer *Multiplayer
	Relay       *Relay
}

func NewConsole(db *database.SQLite, opts ...Option) *Console {
	config := DefaultConfig()
	for _, fn := range opts {
		if err := fn(config); err != nil {
			panic("failed to initialize config: " + err.Error())
		}
	}

	multiplayer := NewMultiplayer()

	var relay *Relay
	var err error
	if config.RunMode == model.RunModeRelay {
		relay, err = NewRelay(config.RelayBindAddr, multiplayer)
		if err != nil {
			panic("failed to initialize relay: " + err.Error())
		}

		multiplayer.Relay = relay
	}

	return &Console{
		DB:          db,
		Multiplayer: multiplayer,
		Relay:       relay,
		Config:      config,
	}
}

type Option func(*Config) error

type Config struct {
	RunMode            model.RunMode
	ConsoleBindAddr    string
	ConsolePublicAddr  string
	RelayBindAddr      string
	RelayPublicAddr    string
	CORSAllowedOrigins []string
	Version            string
}

func DefaultConfig() *Config {
	return &Config{
		RunMode:            model.RunModeLAN,
		ConsoleBindAddr:    "localhost:2137",
		ConsolePublicAddr:  "http://localhost:2137",
		RelayBindAddr:      ":9999",
		RelayPublicAddr:    "localhost:9999",
		CORSAllowedOrigins: []string{"*"},
		Version:            "dev",
	}
}

// TODO: For production replace it with []string{"https://dispel-multi.net"}
func WithCORSAllowedOrigins(allowedOrigins []string) Option {
	return func(c *Config) error {
		c.CORSAllowedOrigins = allowedOrigins
		return nil
	}
}

func WithConsoleAddr(bindAddr, publicAddr string) Option {
	return func(c *Config) error {
		c.ConsoleBindAddr = bindAddr
		c.ConsolePublicAddr = publicAddr
		return nil
	}
}

func WithRelayAddr(bindAddr, publicAddr string) Option {
	return func(c *Config) error {
		c.RelayBindAddr = bindAddr
		c.RelayPublicAddr = publicAddr
		c.RunMode = model.RunModeRelay
		return nil
	}
}

func WithVersion(version string) Option {
	return func(c *Config) error {
		c.Version = version
		return nil
	}
}

func (c *Console) HttpRouter() http.Handler {
	mux := chi.NewRouter()

	mux.Use(middleware.Recoverer)
	mux.Use(middleware.Throttle(100))

	{ // Set up meta routes (readiness, liveness, metrics etc.)
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
		mux.Get("/_metrics", promhttp.Handler().ServeHTTP)
	}

	{ // Set up routes used by the launcher
		wellKnown := chi.NewRouter()
		// wellKnown.Use(slogchi.New(slog.Default()))
		wellKnown.Use(cors.New(cors.Options{
			AllowedOrigins:   c.Config.CORSAllowedOrigins,
			AllowCredentials: false,
			Debug:            false,
			AllowedMethods:   []string{http.MethodGet},
			AllowedHeaders:   []string{"Content-Type"},
			MaxAge:           7200,
		}).Handler)

		wellKnown.Get("/console.json", c.WellKnownInfo())
		mux.Mount("/.well-known/", wellKnown)
	}

	{ // Set up gRPC routes for the backend
		api := chi.NewRouter()
		api.Use(middleware.Timeout(5 * time.Second))
		// api.Use(slogchi.New(slog.Default()))
		api.Use(cors.New(cors.Options{
			AllowedOrigins:   c.Config.CORSAllowedOrigins,
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
		api.Mount(multiv1connect.NewGameServiceHandler(&gameServiceServer{Multiplayer: c.Multiplayer}))
		api.Mount(multiv1connect.NewUserServiceHandler(&userServiceServer{c.DB}))
		api.Mount(multiv1connect.NewRankingServiceHandler(&rankingServiceServer{c.DB}))
		mux.Mount("/grpc/", http.StripPrefix("/grpc", api))
	}

	{ // Set up the lobby (websocket) routes for the backend
		lobby := chi.NewRouter()
		lobby.Use(middleware.Timeout(24 * time.Hour))
		lobby.Mount("/", http.HandlerFunc(c.HandleWebSocket))

		mux.Mount("/lobby", lobby)
	}

	return mux
}

func (c *Console) Handlers() (start GracefulFunc, shutdown GracefulFunc) {
	httpServer := &http.Server{
		Addr:         c.Config.ConsoleBindAddr,
		Handler:      h2c.NewHandler(c.HttpRouter(), &http2.Server{}),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	start = func(ctx context.Context) error {
		slog.Info("Configured console server", "addr", c.Config.ConsoleBindAddr)

		go c.Multiplayer.Run(ctx)
		go c.Relay.Start(ctx)

		// TODO: Move it elsewhere
		if c.Relay != nil && c.Relay.Server != nil {
			go func() {
				for {
					for event := range c.Relay.Server.Events {
						c.Multiplayer.HandleRelayEvent(event)
					}
				}
			}()
		}

		return httpServer.ListenAndServe()
	}

	shutdown = func(ctx context.Context) error {
		slog.Info("Started shutting down the console server")

		c.Multiplayer.Stop()
		if err := c.Relay.Stop(ctx); err != nil {
			slog.Warn("Failed to shut down relay", "error", logging.Error(err))
		}

		if err := httpServer.Shutdown(ctx); err != nil {
			slog.Error("Failed shutting down the console server", logging.Error(err))
			return err
		}
		slog.Info("Successfully shut down the console server")
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
	return func(w http.ResponseWriter, r *http.Request) {
		wk := model.WellKnown{
			Version: c.Config.Version,
			Addr:    c.Config.ConsolePublicAddr,
			RunMode: c.Config.RunMode,
		}

		switch c.Config.RunMode {
		case model.RunModeRelay:
			wk.RelayServerAddr = c.Config.RelayPublicAddr
		case model.RunModeLAN:
			wk.CallerIP = getCallerIP(r.RemoteAddr)
		}

		renderJSON(w, r, wk)
	}
}

func getCallerIP(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return ""
	}
	ip := net.ParseIP(host).To4()
	if ip == nil {
		return ""
	}
	return ip.String()
}
