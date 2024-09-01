package signalserver

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/fxamacker/cbor/v2"
	"github.com/gorilla/websocket"
)

type Server struct {
	sync.RWMutex
	Channels map[string]*Channel
}

func NewServer() (*Server, error) {
	return &Server{
		Channels: make(map[string]*Channel),
	}, nil
}

func (h *Server) GetChannel(channelName string) (*Channel, bool) {
	h.RLock()
	channel, ok := h.Channels[channelName]
	h.RUnlock()
	return channel, ok
}

func (h *Server) SetChannel(channelName string, channel *Channel) {
	h.Lock()
	h.Channels[channelName] = channel
	h.Unlock()
}

func (h *Server) DeleteChannel(channelName string) {
	h.Lock()
	delete(h.Channels, channelName)
	h.Unlock()
}

func (h *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	roomName := params.Get("roomName")
	userID := params.Get("userID")

	var origin string
	upgrader := websocket.Upgrader{
		CheckOrigin: func(*http.Request) bool { return true },
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("Could not upgrade connection",
			"error", err,
			"origin", origin,
			"userId", userID,
			"roomName", roomName)
		return
	}

	h.Join(r.Context(), roomName).Members.Set(userID, conn)
	defer h.Leave(roomName, userID)

	for {
		_, rawSignal, err := conn.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			break
		}
		if len(rawSignal) == 0 && rawSignal[0] == 0x00 {
			if err := conn.WriteMessage(websocket.BinaryMessage, []byte{0x00}); err != nil {
				return
			}
		}

		var m Message
		if err := cbor.Unmarshal(rawSignal, &m); err != nil {
			continue
		}
		if ch, ok := h.GetChannel(roomName); ok {
			ch.Messages <- m
		}
	}
}

func (h *Server) Join(ctx context.Context, channelName string) *Channel {
	if existing, ok := h.GetChannel(channelName); ok {
		return existing
	}

	c := &Channel{
		Name:     channelName,
		Members:  &Members{ws: make(map[string]*websocket.Conn)},
		Messages: make(chan Message),
	}
	h.SetChannel(channelName, c)
	go c.Run()
	return c
}

func (h *Server) Leave(channelName string, userID string) {
	if c, ok := h.GetChannel(channelName); ok {
		c.Broadcast(Leave, Message{Type: Leave, From: userID})
		c.Members.Delete(userID)
		if c.Members.Count() == 0 {
			close(c.Messages)
			h.DeleteChannel(channelName)
		}
	}
}

type Channel struct {
	Name     string
	Members  *Members
	Messages chan Message
}

func (c *Channel) Run() {
	for msg := range c.Messages {
		switch msg.Type {
		case HandshakeRequest:
			if ws, ok := c.Members.Get(msg.From); ok {
				SendMessage(ws, HandshakeResponse, MessageContent[string]{
					Type:    HandshakeResponse,
					Content: msg.From,
				})
			}
			c.Broadcast(Join, Message{
				Type:    Join,
				Content: msg.Content,
			})
		case RTCOffer, RTCAnswer, RTCICECandidate:
			if ws, ok := c.Members.Get(msg.To); ok {
				SendMessage(ws, msg.Type, msg)
			}
		default:
			// Do nothing
		}
	}
}

func (c *Channel) Broadcast(msgType EventType, msg any) {
	payload, err := cbor.Marshal(msg)
	if err != nil {
		slog.Error("Could not marshal the websocket message", "error", err)
		return
	}
	payload = append([]byte{byte(msgType)}, payload...)
	c.Members.Range(func(ws *websocket.Conn) bool {
		if err := ws.WriteMessage(websocket.TextMessage, payload); err != nil {
			slog.Error("Could not broadcast websocket message", "error", err)
		}
		return true
	})
}

func SendMessage(ws *websocket.Conn, msgType EventType, msg any) {
	payload, err := cbor.Marshal(msg)
	if err != nil {
		slog.Error("Could not marshal the websocket message", "error", err)
		return
	}
	payload = append([]byte{byte(msgType)}, payload...)
	if err := ws.WriteMessage(websocket.TextMessage, payload); err != nil {
		slog.Error("Could not write websocket message", "error", err)
	}
}

func (h *Server) Run(httpAddr, turnPublicIP string, turnPortNumber int) (start func(context.Context) error, shutdown func(context.Context) error) {
	if httpAddr == "" {
		httpAddr = "ws://localhost:5050"
	}
	if turnPublicIP == "" {
		turnPublicIP = "127.0.0.1" // IP Address that TURN can be contacted by
	}
	if turnPortNumber == 0 {
		turnPortNumber = 3478 // Listening port
	}

	u, err := url.Parse(httpAddr)
	if err != nil {
		panic(err)
	}

	httpServer := &http.Server{
		Addr:         u.Host,
		Handler:      h,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	users := `username1=password1,username2=password2` // List of username and password (e.g. "user=pass,user=pass")
	realm := "dispelmulti.net"                         // Realm

	turnServer, err := startTURNServer(&turnPublicIP, &turnPortNumber, &users, &realm)
	if err != nil {
		log.Panicf("Could not start TURN server: %s", err)
	}

	start = func(ctx context.Context) error {
		slog.Info("Signal server is running on", "addr", httpAddr)
		return httpServer.ListenAndServe()
	}

	shutdown = func(ctx context.Context) error {
		if err := httpServer.Shutdown(ctx); err != nil {
			slog.Error("Failed shutting down the console server", "error", err)
			return err
		}
		if err := turnServer.Close(); err != nil {
			return err
		}
		return nil
	}

	return start, shutdown
}
