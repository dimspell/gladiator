package console

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
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
	"github.com/pion/turn/v3"
	"github.com/pion/webrtc/v4"
	"github.com/rs/cors"
	slogchi "github.com/samber/slog-chi"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"golang.org/x/net/websocket"
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

	{ // Setup WebSocket routes for P2P communication (signaling)
		mux.Handle("/ws", websocket.Handler(c.WebSocketHandler))
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

	var turnServer *turn.Server

	start = func(ctx context.Context) error {
		publicIP := "127.0.0.1"                            // IP Address that TURN can be contacted by
		port := 3478                                       // Listening port
		users := `username1=password1,username2=password2` // List of username and password (e.g. "user=pass,user=pass")
		realm := "dispelmulti.net"                         // Realm

		var err error
		turnServer, err = startTURNServer(&publicIP, &port, &users, &realm)
		if err != nil {
			return err
		}

		slog.Info("Configured console server", "addr", c.Addr)
		return httpServer.ListenAndServe()
	}

	shutdown = func(ctx context.Context) error {
		slog.Info("Started shutting down the console server")

		if turnServer != nil {
			turnServer.Close()
		}

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

func (c *Console) WebSocketHandler(ws *websocket.Conn) {
	// Prepare the configuration
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			// {
			// 	URLs: []string{"stun:stun.l.google.com:19302"},
			// },
			{
				URLs:       []string{"turn:127.0.0.1:3478"},
				Username:   "username1",
				Credential: "password1",
			},
		},
	}

	// Create a new RTCPeerConnection
	peerConnection, err := webrtc.NewPeerConnection(config)
	if err != nil {
		panic(err)
	}

	// When Pion gathers a new ICE Candidate send it to the client. This is how
	// ice trickle is implemented. Everytime we have a new candidate available we send
	// it as soon as it is ready. We don't wait to emit an Offer/Answer until they are
	// all available
	peerConnection.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}

		outbound, marshalErr := json.Marshal(c.ToJSON())
		if marshalErr != nil {
			panic(marshalErr)
		}

		if _, err = ws.Write(outbound); err != nil {
			panic(err)
		}
	})

	// Set the handler for ICE connection state
	// This will notify you when the peer has connected/disconnected
	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		fmt.Printf("ICE Connection State has changed: %s\n", connectionState.String())
	})

	// Send the current time via a DataChannel to the remote peer every 3 seconds
	peerConnection.OnDataChannel(func(d *webrtc.DataChannel) {
		d.OnMessage(func(msg webrtc.DataChannelMessage) {
			slog.Info("Received message", "message", string(msg.Data))
		})

		d.OnOpen(func() {
			for range time.Tick(time.Second * 3) {
				if err = d.SendText(time.Now().String()); err != nil {
					panic(err)
				}
			}
		})

		d.OnClose(func() {
			slog.Info("Connection closed")
		})
	})

	buf := make([]byte, 1500)
	for {
		// Read each inbound WebSocket Message
		n, err := ws.Read(buf)
		if err != nil {
			if err == io.EOF {
				peerConnection.Close()
				return
			}
			panic(err)
		}

		// Unmarshal each inbound WebSocket message
		var (
			candidate webrtc.ICECandidateInit
			offer     webrtc.SessionDescription
		)

		log.Println(string(buf[:n]))

		switch {
		// Attempt to unmarshal as a SessionDescription. If the SDP field is empty
		// assume it is not one.
		case json.Unmarshal(buf[:n], &offer) == nil && offer.SDP != "":
			slog.Info("Received Offer", "offer", offer)

			if err = peerConnection.SetRemoteDescription(offer); err != nil {
				panic(err)
			}

			answer, answerErr := peerConnection.CreateAnswer(nil)
			if answerErr != nil {
				panic(answerErr)
			}

			if err = peerConnection.SetLocalDescription(answer); err != nil {
				panic(err)
			}

			outbound, marshalErr := json.Marshal(answer)
			if marshalErr != nil {
				panic(marshalErr)
			}

			if _, err = ws.Write(outbound); err != nil {
				panic(err)
			}
		// Attempt to unmarshal as a ICECandidateInit. If the candidate field is empty
		// assume it is not one.
		case json.Unmarshal(buf[:n], &candidate) == nil && candidate.Candidate != "":
			slog.Info("Received ICE Candidate", "candidate", candidate)

			if err = peerConnection.AddICECandidate(candidate); err != nil {
				panic(err)
			}
		default:
			panic("Unknown message")
		}
	}
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
