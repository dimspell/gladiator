package signalserver

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/coder/websocket"
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

const supportedSubProtocol = "signalserver"

func (h *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	roomName := params.Get("roomName")
	userID := params.Get("userID")

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		Subprotocols: []string{supportedSubProtocol},
	})
	if err != nil {
		slog.Error("Could not accept the connection",
			"error", err,
			"origin", r.Header.Get("Origin"),
			"userId", userID,
			"roomName", roomName)
		return
	}
	defer conn.CloseNow()

	if conn.Subprotocol() != supportedSubProtocol {
		_ = conn.Close(websocket.StatusPolicyViolation, "client must speak the echo subprotocol")
		return
	}

	h.Join(r.Context(), roomName).Members.Set(userID, conn)
	defer h.Leave(roomName, userID)

	for {
		err = h.WaitAndHandleMessage(r.Context(), conn, roomName)
		if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
			return
		}
		if err != nil {
			slog.Error("Could not handle the message", "error", err)
			return
		}
	}
}

func (h *Server) WaitAndHandleMessage(ctx context.Context, conn *websocket.Conn, roomName string) error {
	_, payload, err := conn.Read(ctx)
	if err != nil {
		slog.Warn("Could not read the message", "error", err, "closeError", websocket.CloseStatus(err))
		return err
	}
	// if len(payload) == 0 && payload[0] == 0x00 {
	// 	ctx, cancel := context.WithTimeout(ctx, time.Second)
	// 	defer cancel()
	//
	// 	if err := conn.Write(ctx, websocket.MessageText, []byte{0x00}); err != nil {
	// 		return err
	// 	}
	// }

	var m Message
	if err := DefaultCodec.Unmarshal(payload, &m); err != nil {
		return nil
	}
	if ch, ok := h.GetChannel(roomName); ok {
		ch.Messages <- m
	}
	return nil
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
	go c.Run(ctx)
	return c
}

func (h *Server) Leave(channelName string, userID string) {
	if c, ok := h.GetChannel(channelName); ok {
		ctx := context.TODO()
		// ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
		// defer cancel()

		c.Broadcast(ctx, Leave, Message{Type: Leave, From: userID})
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

func (c *Channel) Run(ctx context.Context) {
	for msg := range c.Messages {
		slog.Debug("Received a signal message", "channel", c.Name, "type", msg.Type.String(), "from", msg.From, "to", msg.To)

		switch msg.Type {
		case HandshakeRequest:
			c.SendJoin(ctx, msg)
		case RTCOffer, RTCAnswer, RTCICECandidate:
			c.ForwardRTCMessage(ctx, msg)
		default:
			// Do nothing
		}
	}
}

func (c *Channel) SendJoin(ctx context.Context, msg Message) {
	ctx = context.TODO()
	// ctx, cancel := context.WithTimeout(context.TODO(), time.Second*10)
	// defer cancel()

	if ws, ok := c.Members.Get(msg.From); ok {
		SendMessage(ctx, ws, HandshakeResponse, Message{
			Type:    HandshakeResponse,
			To:      msg.From,
			Content: msg.From,
		})
	}
	c.Broadcast(ctx, Join, Message{
		Type:    Join,
		Content: msg.Content,
	})
}

func (c *Channel) ForwardRTCMessage(ctx context.Context, msg Message) {
	slog.Debug("Forwarding RTC message", "channel", c.Name, "type", msg.Type.String(), "from", msg.From, "to", msg.To)

	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	if ws, ok := c.Members.Get(msg.To); ok {
		SendMessage(ctx, ws, msg.Type, msg)
	}
}

func (c *Channel) Broadcast(ctx context.Context, msgType EventType, msg Message) {
	payload, err := DefaultCodec.Marshal(msg)
	if err != nil {
		slog.Error("Could not marshal the websocket message", "error", err)
		return
	}
	payload = append([]byte{byte(msgType)}, payload...)
	c.Members.Range(func(ws *websocket.Conn) bool {
		if err := ws.Write(ctx, websocket.MessageText, payload); err != nil {
			slog.Error("Could not broadcast websocket message", "error", err)
		}
		return true
	})
}

func SendMessage(ctx context.Context, ws *websocket.Conn, msgType EventType, msg Message) {
	slog.Debug("Sending a signal message", "type", msgType.String(), "from", msg.From, "to", msg.To)

	payload, err := DefaultCodec.Marshal(msg)
	if err != nil {
		slog.Error("Could not marshal the websocket message", "error", err)
		return
	}
	payload = append([]byte{byte(msgType)}, payload...)
	if err := ws.Write(ctx, websocket.MessageText, payload); err != nil {
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
