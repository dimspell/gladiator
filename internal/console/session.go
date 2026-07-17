package console

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
	"github.com/dimspell/gladiator/internal/app/logger/logging"
	"github.com/dimspell/gladiator/internal/metrics"
	"github.com/dimspell/gladiator/internal/wire"
)

type UserSession struct {
	UserID int64  `json:"userID,omitempty"`
	GameID string `json:"gameID,omitempty"`

	ConnectedAt time.Time `json:"connectedAt,omitempty"`
	JoinedAt    time.Time `json:"joinedAt,omitempty"`
	IPAddress   string    `json:"ip"`

	WebSocket ConnReadWriter

	User      wire.User
	Character wire.Character

	// OnWriteError is invoked at most once when a websocket write fails, so the
	// owner can tear down the dead session. It runs asynchronously to avoid
	// re-entering locks held by the caller (e.g. forEachSession / LeaveRoom).
	OnWriteError func()

	// disconnecting guards OnWriteError so a single failed session triggers
	// cleanup exactly once, even across concurrent Send calls.
	disconnecting atomic.Bool

	// disconnected marks that teardown has already run (or is running), so
	// SetPlayerDisconnected is a no-op on a second call (e.g. from both the
	// OnWriteError goroutine and the HandleSession defer).
	disconnected atomic.Bool
}

func NewUserSession(id int64, conn ConnReadWriter) *UserSession {
	return &UserSession{
		UserID:      id,
		ConnectedAt: time.Now().In(time.UTC),
		WebSocket:   conn,
	}
}

func (us *UserSession) ReadNext(ctx context.Context) ([]byte, error) {
	conn := us.WebSocket
	if conn == nil {
		return nil, fmt.Errorf("not connected")
	}
	_, payload, err := conn.Read(ctx)
	if err != nil {
		// TODO: Make the log more clear that the user has disconnected
		slog.Warn("Could not read the message", logging.Error(err), "closeError", websocket.CloseStatus(err))
		return nil, err
	}
	return payload, nil
}

func (us *UserSession) Send(ctx context.Context, payload []byte) {
	conn := us.WebSocket
	if conn == nil {
		slog.Debug("not connected", "userId", us.UserID)
		metrics.FailedMessageSends.WithLabelValues(fmt.Sprintf("%d", us.UserID), "not_connected").Inc()
		return
	}
	if len(payload) < 1 {
		slog.Debug("payload is too short", "length", len(payload))
		metrics.FailedMessageSends.WithLabelValues(fmt.Sprintf("%d", us.UserID), "payload_too_short").Inc()
		return
	}

	if err := wire.Write(ctx, conn, payload); err != nil {
		slog.Warn("Could not send a WS message", "to", us.UserID, logging.Error(err))
		metrics.FailedMessageSends.WithLabelValues(fmt.Sprintf("%d", us.UserID), "write_error").Inc()
		// The socket is dead; tear down the session. Run asynchronously so we
		// don't re-enter locks the caller may hold (forEachSession / LeaveRoom).
		if us.disconnecting.CompareAndSwap(false, true) && us.OnWriteError != nil {
			go us.OnWriteError()
		}
	} else {
		metrics.MessagesSentPerPlayer.WithLabelValues(fmt.Sprintf("%d", us.UserID)).Inc()
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
}
