package bsession

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/coder/websocket"
	multiv1 "github.com/dimspell/gladiator/gen/multi/v1"
	"github.com/dimspell/gladiator/internal/app/logger/logging"
	"github.com/dimspell/gladiator/internal/backend/packet"
	"github.com/dimspell/gladiator/internal/backend/proxy"
	"github.com/dimspell/gladiator/internal/model"
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

	OnceSelectedCharacter sync.Once
	observerDone          context.CancelFunc
	wsConn                *websocket.Conn

	State *SessionState
	Proxy proxy.ProxyClient
}

func NewSession(backendConn net.Conn) *Session {
	return &Session{
		Conn:  backendConn,
		ID:    uuid.New().String(),
		State: &SessionState{},
	}
}

func (s *Session) Stop() {
	s.observerDone()
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

func (s *Session) GetUserID() int64 { return s.UserID }

func (s *Session) SendToGame(packetType packet.Code, payload []byte) error {
	return sendPacket(s.Conn, packetType, payload)
}

func sendPacket(conn net.Conn, packetType packet.Code, payload []byte) error {
	if conn == nil {
		return fmt.Errorf("backend: invalid client connection")
	}

	data := packet.EncodePacket(packetType, payload)

	slog.Debug("Sent",
		"packetType", packetType,
		"bytes", data,
		"length", len(data),
	)

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

func (s *Session) InitObserver(registerNewObserver func(context.Context) error) error {
	var err error
	s.OnceSelectedCharacter.Do(func() {
		ctx := context.TODO()

		err = s.JoinLobby(ctx)
		if err != nil {
			return
		}
		err = registerNewObserver(ctx)
		if err != nil {
			return
		}
	})
	return err
}

func (s *Session) StartObserver(ctx context.Context, observe func(ctx context.Context, wsConn *websocket.Conn)) error {
	if s.wsConn == nil {
		return fmt.Errorf("missing websocket connection")
	}

	go observe(ctx, s.wsConn)
	return nil
}

func (s *Session) StopObserver() {
	if s.observerDone != nil {
		s.observerDone()
	}
	if s.Conn != nil {
		_ = s.Conn.Close()
	}
}

func (s *Session) ConnectOverWebsocket(ctx context.Context, user *multiv1.User, wsURL string) error {
	s.Lock()
	defer s.Unlock()

	ws, err := wire.Connect(ctx, wsURL, wire.User{
		UserID:   user.UserId,
		Username: user.Username,
		Version:  wire.ProtoVersion,
	})
	if err != nil {
		return err
	}

	ctx, s.observerDone = context.WithCancel(ctx)
	s.wsConn = ws

	go func(ctx context.Context, ws *websocket.Conn) {
		<-ctx.Done()
		ws.CloseNow()
		return
	}(ctx, ws)
	return nil
}

func (s *Session) ConsumeWebSocket(ctx context.Context) ([]byte, error) {
	_, p, err := s.wsConn.Read(ctx)
	return p, err
}

func (s *Session) RegisterNewObserver(ctx context.Context) error {
	handlers := []proxy.MessageHandler{
		NewLobbyEventHandler(s).Handle,
		s.Proxy.Handle,
	}
	observe := func(ctx context.Context, wsConn *websocket.Conn) {
		for {
			if ctx.Err() != nil {
				return
			}

			// Read the broadcast and handle them as commands.
			p, err := s.ConsumeWebSocket(ctx)
			if err != nil {
				if errors.Is(err, context.Canceled) {
					return
				}
				slog.Error("Error reading from WebSocket", "session", s.ID, logging.Error(err))
				return
			}

			// TODO: Register handlers and handle them here.
			for _, handleFn := range handlers {
				if err := handleFn(ctx, p); err != nil {
					slog.Error("Error handling message", "session", s.ID, logging.Error(err))
					return
				}
			}
		}
	}
	return s.StartObserver(ctx, observe)
}

func (s *Session) SendEvent(ctx context.Context, eventType wire.EventType, content any) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second*3)
	defer cancel()

	return wire.Write(ctx, s.wsConn, wire.Compose(
		eventType,
		wire.Message{
			From:    strconv.Itoa(int(s.GetUserID())),
			Type:    eventType,
			Content: content,
		}),
	)
}

func (s *Session) SendEventTo(ctx context.Context, eventType wire.EventType, content any, recipientId int64) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second*3)
	defer cancel()

	return wire.Write(ctx, s.wsConn, wire.Compose(
		eventType,
		wire.Message{
			From:    strconv.Itoa(int(s.GetUserID())),
			To:      strconv.Itoa(int(recipientId)),
			Type:    eventType,
			Content: content,
		}),
	)
}

func (s *Session) JoinLobby(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	if err := s.SendEvent(ctx, wire.JoinLobby, wire.Player{
		UserID:      s.UserID,
		Username:    s.Username,
		CharacterID: s.CharacterID,
		ClassType:   byte(s.ClassType),
	}); err != nil {
		return fmt.Errorf("failed send a join lobby message: %w for %d", err, s.UserID)
	}

	// Expect to receive the joined message.
	_, response, err := s.wsConn.Read(ctx)
	if err != nil {
		return fmt.Errorf("failed receive a join lobby message: %w", err)
	}
	if len(response) != 1 || wire.EventType(response[0]) != wire.JoinedLobby {
		return fmt.Errorf("expected joined message, got: %s (len=%d)", string(response), len(response))
	}
	return nil
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

func (s *Session) SendRTCICECandidate(ctx context.Context, candidate webrtc.ICECandidateInit, recipientId int64) error {
	return s.SendEventTo(ctx, wire.RTCICECandidate, candidate, recipientId)
}

func (s *Session) SendRTCOffer(ctx context.Context, offer webrtc.SessionDescription, recipientId int64) error {
	return s.SendEventTo(ctx, wire.RTCOffer, wire.Offer{
		CreatorID:   s.UserID,
		RecipientID: recipientId,
		Offer:       offer,
	}, recipientId)
}

func (s *Session) SendRTCAnswer(ctx context.Context, answer webrtc.SessionDescription, recipientId int64) error {
	return s.SendEventTo(ctx, wire.RTCAnswer, wire.Offer{
		CreatorID:   s.UserID,
		RecipientID: recipientId,
		Offer:       answer,
	}, recipientId)
}
