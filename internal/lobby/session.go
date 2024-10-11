package lobby

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/coder/websocket"
	"github.com/dimspell/gladiator/internal/icesignal"
)

type UserSession struct {
	UserID    string    `json:"userID,omitempty"`
	GameID    string    `json:"gameID,omitempty"`
	Connected bool      `json:"connected,omitempty"`
	LastSeen  time.Time `json:"lastSeen"`

	wsConn ConnReadWriter

	Player icesignal.Player
}

func NewUserSession(id string, conn ConnReadWriter) *UserSession {
	return &UserSession{
		UserID:    id,
		Connected: true,
		LastSeen:  time.Now().In(time.UTC),
		wsConn:    conn,
	}
}

func (us *UserSession) ReadNext(ctx context.Context) ([]byte, error) {
	if !us.Connected {
		return nil, fmt.Errorf("not connected")
	}
	_, payload, err := us.wsConn.Read(ctx)
	if err != nil {
		slog.Warn("Could not read the message", "error", err, "closeError", websocket.CloseStatus(err))
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

	slog.Debug("Sending a signal message", "to", us.UserID, "type", icesignal.EventType(payload[0]).String())

	if err := us.wsConn.Write(ctx, websocket.MessageText, payload); err != nil {
		slog.Warn("Could not send a WS message", "to", us.UserID, "error", err)
		us.Connected = false
		// TODO: There is no logic to disconnect and remove the failing session
	}
}

func (us *UserSession) SendMessage(ctx context.Context, msgType icesignal.EventType, msg icesignal.Message) {
	us.Send(ctx, icesignal.Compose(msgType, msg))
}

var _ ConnReadWriter = (*websocket.Conn)(nil)

type ConnReadWriter interface {
	Read(ctx context.Context) (websocket.MessageType, []byte, error)
	Write(ctx context.Context, typ websocket.MessageType, p []byte) error
	CloseNow() error
	// TODO: Add Close function
}
