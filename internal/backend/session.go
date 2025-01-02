package backend

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net"
	"sync"
	"time"

	"github.com/coder/websocket"
	multiv1 "github.com/dimspell/gladiator/gen/multi/v1"
	"github.com/dimspell/gladiator/internal/app/logger"
	"github.com/dimspell/gladiator/internal/model"
	"github.com/dimspell/gladiator/internal/proxy/p2p"
	"github.com/dimspell/gladiator/internal/wire"
	"github.com/google/uuid"
	"github.com/pion/webrtc/v4"
)

type Session struct {
	sync.RWMutex

	// ID keeps the session identifier for the backend.
	ID string

	UserID      int64
	Username    string
	CharacterID int64
	ClassType   model.ClassType

	// Conn stores the TCP connection between the backend and the game client.
	Conn net.Conn

	onceSelectedCharacter sync.Once
	observerDone          context.CancelFunc
	wsConn                *websocket.Conn

	IpRing *p2p.IpRing

	State *SessionState
}

func (s *Session) SetLogonData(user *multiv1.User) {
	s.Lock()
	s.UserID = user.UserId
	s.Username = user.Username
	s.Unlock()
}

func (s *Session) UpdateCharacter(character *multiv1.Character) {
	s.Lock()
	info := model.ParseCharacterInfo(character.Stats)

	s.CharacterID = character.CharacterId
	s.ClassType = info.ClassType
	s.Unlock()
}

func (s *Session) GetUserID() string { return fmt.Sprintf("%d", s.UserID) }

func (b *Backend) AddSession(tcpConn net.Conn) *Session {
	if b.SessionCounter == math.MaxUint64 {
		b.SessionCounter = 0
	}
	b.SessionCounter++
	id := fmt.Sprintf("%s-%d", uuid.New().String(), b.SessionCounter)
	slog.Debug("New session", "session", id, "backend", fmt.Sprintf("%p", b.Proxy))

	session := &Session{
		Conn: tcpConn,
		ID:   id,

		IpRing: p2p.NewIpRing(),

		State: &SessionState{},
		// RoomPlayers: p2p.NewPeers(),
	}
	b.ConnectedSessions.Store(session.ID, session)
	return session
}

func (b *Backend) CloseSession(session *Session) error {
	slog.Info("Session closed", "session", session.ID)

	b.Proxy.Close(session)

	if session.observerDone != nil {
		session.observerDone()
	}

	b.ConnectedSessions.Delete(session.ID)

	if session.Conn != nil {
		_ = session.Conn.Close()
	}

	session = nil
	return nil
}

func (s *Session) Send(packetType PacketType, payload []byte) error {
	return sendPacket(s.Conn, packetType, payload)
}

func sendPacket(conn net.Conn, packetType PacketType, payload []byte) error {
	if conn == nil {
		return fmt.Errorf("backend: invalid client connection")
	}

	data := encodePacket(packetType, payload)

	if logger.PacketLogger != nil {
		logger.PacketLogger.Debug("Sent",
			"packetType", packetType,
			"bytes", data,
			"length", len(data),
		)
	}

	_, err := conn.Write(data)
	return err
}

// ToPlayer creates a Player data object from the session data
func (s *Session) ToPlayer(ipAddr net.IP) wire.Player {
	return wire.Player{
		UserID:      s.UserID,
		Username:    s.Username,
		CharacterID: s.CharacterID,
		ClassType:   byte(s.ClassType),
		IPAddress:   ipAddr.To4().String(),
	}
}

func (b *Backend) ConnectToLobby(ctx context.Context, user *multiv1.User, session *Session) error {
	session.Lock()
	defer session.Unlock()

	ws, err := wire.Connect(ctx, b.SignalServerURL, wire.User{
		UserID:   user.UserId,
		Username: user.Username,
		Version:  wire.ProtoVersion,
	})
	if err != nil {
		return err
	}

	ctx, session.observerDone = context.WithCancel(ctx)
	session.wsConn = ws

	go func(ctx context.Context, ws *websocket.Conn) {
		<-ctx.Done()
		ws.CloseNow()
		return
	}(ctx, ws)
	return nil
}

func (b *Backend) JoinLobby(ctx context.Context, session *Session) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second*3)
	defer cancel()

	if err := session.SendEvent(ctx, wire.JoinLobby, wire.Player{
		UserID:      session.UserID,
		Username:    session.Username,
		CharacterID: session.CharacterID,
		ClassType:   byte(session.ClassType),
	}); err != nil {
		return fmt.Errorf("backend: failed send a join lobby message: %w for %d", err, session.UserID)
	}

	// Expect to receive the joined message.
	_, response, err := session.wsConn.Read(ctx)
	if err != nil {
		return fmt.Errorf("backend: failed receive a join lobby message: %w", err)
	}
	if len(response) != 1 || wire.EventType(response[0]) != wire.JoinedLobby {
		return fmt.Errorf("expected joined message, got: %s (len=%d)", string(response), len(response))
	}
	return nil
}

type MessageHandler interface {
	Handle(ctx context.Context, payload []byte) error
}

func (b *Backend) RegisterNewObserver(ctx context.Context, session *Session) error {
	if session.wsConn == nil {
		return fmt.Errorf("backend: invalid websocket client connection")
	}

	handlers := []MessageHandler{
		&SessionMessageHandler{session: session},
		b.Proxy.ExtendWire(session),
	}
	observe := func(ctx context.Context, wsConn *websocket.Conn) {
		for {
			if ctx.Err() != nil {
				return
			}

			// Read the broadcast & handle them as commands.
			_, p, err := wsConn.Read(ctx)
			if err != nil {
				if errors.Is(err, context.Canceled) {
					return
				}
				slog.Error("Error reading from WebSocket", "session", session.ID, "error", err)
				return
			}

			// slog.Debug("Signal from lobby", "type", et.String(), "session", session.ID, "payload", string(p[1:]))

			// TODO: Register handlers and handle them here.
			for _, handler := range handlers {
				if err := handler.Handle(ctx, p); err != nil {
					slog.Error("Error handling message", "session", session.ID, "error", err)
					return
				}
			}
		}
	}
	go observe(ctx, session.wsConn)
	return nil
}

func (s *Session) SendEvent(ctx context.Context, eventType wire.EventType, content any) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	return wire.Write(ctx, s.wsConn, wire.Compose(
		eventType,
		wire.Message{
			From:    s.GetUserID(),
			Type:    eventType,
			Content: content,
		}),
	)
}

func (s *Session) SendEventTo(ctx context.Context, eventType wire.EventType, content any, recipientId string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	return wire.Write(ctx, s.wsConn, wire.Compose(
		eventType,
		wire.Message{
			From:    s.GetUserID(),
			To:      recipientId,
			Type:    eventType,
			Content: content,
		}),
	)
}

func (s *Session) SendChatMessage(ctx context.Context, text string) error {
	return s.SendEvent(ctx, wire.Chat, wire.ChatMessage{
		User: s.Username,
		Text: text,
	})
}

func (s *Session) SendSetRoomReady(ctx context.Context, gameRoomId string) error {
	return s.SendEvent(ctx, wire.SetRoomReady, gameRoomId)
}

func (s *Session) SendLeaveRoom(ctx context.Context, gameRoom *GameRoom) error {
	return s.SendEvent(ctx, wire.LeaveRoom, gameRoom.ID)
}

func (s *Session) SendRTCICECandidate(ctx context.Context, candidate webrtc.ICECandidateInit, recipientId string) error {
	return s.SendEventTo(ctx, wire.RTCICECandidate, candidate, recipientId)
}

func (s *Session) SendRTCOffer(ctx context.Context, offer webrtc.SessionDescription, recipientId string) error {
	return s.SendEventTo(ctx, wire.RTCOffer, wire.Offer{
		UserID: s.UserID,
		Offer:  offer,
	}, recipientId)
}

func (s *Session) SendRTCAnswer(ctx context.Context, answer webrtc.SessionDescription, recipientId string) error {
	return s.SendEventTo(ctx, wire.RTCAnswer, wire.Offer{
		UserID: s.UserID,
		Offer:  answer,
	}, recipientId)
}
