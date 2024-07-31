package signalserver

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/dimspell/gladiator/console/signalserver/message"
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

func (h *Server) Get(channelName string) (*Channel, bool) {
	h.RLock()
	channel, ok := h.Channels[channelName]
	h.RUnlock()
	return channel, ok
}

func (h *Server) Set(channelName string, channel *Channel) {
	h.Lock()
	h.Channels[channelName] = channel
	h.Unlock()
}

func (h *Server) Delete(channelName string) {
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

	h.Join(roomName).Members.Set(userID, conn)
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

		var m message.Message
		if err := cbor.Unmarshal(rawSignal, &m); err != nil {
			continue
		}
		if ch, ok := h.Get(roomName); ok {
			ch.Messages <- m
		}
	}
}

func (h *Server) Join(channelName string) *Channel {
	c := &Channel{
		Name:     channelName,
		Members:  &Members{ws: make(map[string]*websocket.Conn)},
		Messages: make(chan message.Message),
	}
	if existing, ok := h.Get(channelName); !ok {
		go c.Run()
		h.Set(channelName, c)
		return c
	} else {
		return existing
	}
}

func (h *Server) Leave(channelName string, userID string) {
	if c, ok := h.Get(channelName); ok {
		c.Broadcast(message.Message{Type: message.Leave, From: userID})
		c.Members.Delete(userID)
		if c.Members.Count() == 0 {
			close(c.Messages)
			h.Delete(channelName)
		}
	}
}

type Channel struct {
	Name     string
	Members  *Members
	Messages chan message.Message
}

func (c *Channel) Run() {
	for msg := range c.Messages {
		switch msg.Type {
		case message.HandshakeRequest:
			if member, ok := c.Members.Get(msg.From); ok {
				SendMessage(member, message.Message{
					Type:    message.HandshakeResponse,
					Content: msg.From,
				})
			}
			c.Broadcast(
				message.Message{
					Type: message.Join,
					Content: message.Member{
						ID:   msg.From,
						Name: msg.Content.(string),
						// Channel: c.Name,
					},
				})
		case message.RTCOffer, message.RTCAnswer, message.RTCICECandidate:
			if member, ok := c.Members.Get(msg.To); ok {
				SendMessage(member, msg)
			}
		default:
			// Do nothing
		}
	}
}

func (c *Channel) Broadcast(msg message.Message) {
	payload, _ := cbor.Marshal(msg)
	payload = append([]byte{byte(msg.Type)}, payload...)
	c.Members.Range(func(ws *websocket.Conn) bool {
		if err := ws.WriteMessage(websocket.TextMessage, payload); err != nil {
			log.Println("write:", err)
		}
		return true
	})
}

func SendMessage(ws *websocket.Conn, msg message.Message) {
	payload, _ := cbor.Marshal(msg)
	payload = append([]byte{byte(msg.Type)}, payload...)
	if err := ws.WriteMessage(websocket.TextMessage, payload); err != nil {
		log.Println("write:", err)
	}
}

func (h *Server) Run() (start func(context.Context) error, shutdown func(context.Context) error) {
	httpServer := &http.Server{
		Addr:         ":5050",
		Handler:      h,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	publicIP := "127.0.0.1"                            // IP Address that TURN can be contacted by
	port := 3478                                       // Listening port
	users := `username1=password1,username2=password2` // List of username and password (e.g. "user=pass,user=pass")
	realm := "dispelmulti.net"                         // Realm

	turnServer, err := startTURNServer(&publicIP, &port, &users, &realm)
	if err != nil {
		log.Panicf("Could not start TURN server: %s", err)
	}

	start = func(ctx context.Context) error {
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
