package console

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/coder/websocket"
	"github.com/dimspell/gladiator/internal/app/logger/logging"
	"github.com/dimspell/gladiator/internal/wire"
)

type UserSession struct {
	UserID    int64  `json:"userID,omitempty"`
	GameID    string `json:"gameID,omitempty"`
	Connected bool   `json:"connected,omitempty"`

	ConnectedAt time.Time `json:"connectedAt,omitempty"`
	JoinedAt    time.Time `json:"joinedAt,omitempty"`

	// TODO: It is never provided
	IPAddress string `json:"ip"`

	wsConn ConnReadWriter

	User      wire.User
	Character wire.Character
}

func NewUserSession(id int64, conn ConnReadWriter) *UserSession {
	return &UserSession{
		UserID:      id,
		Connected:   true,
		ConnectedAt: time.Now().In(time.UTC),
		wsConn:      conn,
	}
}

func (us *UserSession) ReadNext(ctx context.Context) ([]byte, error) {
	if !us.Connected {
		return nil, fmt.Errorf("not connected")
	}
	_, payload, err := us.wsConn.Read(ctx)
	if err != nil {
		// TODO: Make the log more clear that the user has disconnected
		slog.Warn("Could not read the message", logging.Error(err), "closeError", websocket.CloseStatus(err))
		return nil, err
	}
	return payload, nil
}

func (us *UserSession) Send(ctx context.Context, payload []byte) {
	if len(payload) < 1 {
		slog.Debug("payload is too short", "length", len(payload))
		return
	}
	if !us.Connected {
		slog.Debug("not connected", "userId", us.UserID)
		return
	}

	if err := wire.Write(ctx, us.wsConn, payload); err != nil {
		slog.Warn("Could not send a WS message", "to", us.UserID, logging.Error(err))
		us.Connected = false
		// TODO: There is no logic to disconnect and remove the failing session
	}
}

func (us *UserSession) SendMessage(ctx context.Context, msgType wire.EventType, msg wire.Message) {
	us.Send(ctx, wire.Compose(msgType, msg))
}

func (us *UserSession) ToPlayer() wire.Player {
	return wire.Player{
		UserID:      us.User.UserID,
		Username:    us.User.Username,
		CharacterID: us.Character.CharacterID,
		ClassType:   us.Character.ClassType,
	}
}

var _ ConnReadWriter = (*websocket.Conn)(nil)

type ConnReadWriter interface {
	Read(ctx context.Context) (websocket.MessageType, []byte, error)
	Write(ctx context.Context, typ websocket.MessageType, p []byte) error
	CloseNow() error
	// TODO: Add Close function
}
