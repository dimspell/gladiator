package backend

import (
	"context"
	"encoding/binary"
	"fmt"
	"log/slog"
	"math"
	"net"
	"slices"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/dimspell/gladiator/internal/app/logger"
	"github.com/dimspell/gladiator/internal/model"
	"github.com/dimspell/gladiator/internal/wire"
	"github.com/google/uuid"
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

	// lobbyUsers contains list of players who are connected to lobby server.
	lobbyUsers []wire.Player

	observerDone context.CancelFunc
	wsConn       *websocket.Conn

	onceSelectedCharacter sync.Once
}

func (b *Backend) AddSession(tcpConn net.Conn) *Session {
	if b.SessionCounter == math.MaxUint64 {
		b.SessionCounter = 0
	}
	b.SessionCounter++
	id := fmt.Sprintf("%s-%d", uuid.New().String(), b.SessionCounter)
	slog.Debug("New session", "session", id)

	session := &Session{
		Conn: tcpConn,
		ID:   id,
	}
	b.ConnectedSessions.Store(session.ID, session)
	return session
}

func (b *Backend) CloseSession(session *Session) error {
	slog.Info("Session closed", "session", session.ID)

	b.Proxy.Close()

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

func (b *Backend) Send(conn net.Conn, packetType PacketType, payload []byte) error {
	if conn == nil {
		return fmt.Errorf("backend: invalid client connection")
	}

	data := encodePacket(packetType, payload)

	if logger.PacketLogger != nil {
		logger.PacketLogger.Debug("Sent",
			"packetType", packetType,
			"remoteAddr", conn.RemoteAddr().String(),
			"bytes", data,
			"length", len(data),
		)
	}

	_, err := conn.Write(data)
	return err
}

func encodePacket(packetType PacketType, payload []byte) []byte {
	length := len(payload) + 4
	packet := make([]byte, length)

	// Header
	packet[0] = 255
	packet[1] = byte(packetType)
	binary.LittleEndian.PutUint16(packet[2:4], uint16(length))

	// Data
	copy(packet[4:], payload)

	return packet
}

func (b *Backend) RegisterNewObserver(session *Session) (err error) {
	// Only one observer at the time per session.
	ctx, observerDone := context.WithCancel(context.Background())

	wsConn, err := b.ConnectToWebSocket(ctx, session)
	if err != nil {
		observerDone()
		return err
	}
	observe, err := b.createObserver(wsConn, session)
	if err != nil {
		observerDone()
		return err
	}

	session.wsConn = wsConn
	session.observerDone = observerDone

	go func() {
		<-ctx.Done()
		wsConn.CloseNow()
		return
	}()
	go observe(ctx)

	return nil
}

func (b *Backend) createObserver(wsConn *websocket.Conn, session *Session) (func(context.Context), error) {
	observer := func(ctx context.Context) {
		for {
			if ctx.Err() != nil {
				return
			}

			// Read the broadcast & handle them as commands.
			_, p, err := wsConn.Read(ctx)
			if err != nil {
				slog.Error("Error reading from WebSocket", "session", session.ID, "error", err)
				return
			}
			slog.Debug("Received packet", "session", session.ID, "packet", p)

			et := wire.ParseEventType(p)
			switch et {
			case wire.Chat:
				_, msg, err := wire.DecodeTyped[wire.ChatMessage](p)
				if err != nil {
					slog.Warn("Could not decode the message", "session", session.ID, "error", err, "event", et.String(), "payload", p)
					continue
				}
				if err := b.Send(session.Conn, ReceiveMessage, NewGlobalMessage(msg.Content.User, msg.Content.Text)); err != nil {
					slog.Error("Error writing chat message over the backend wire", "session", session.ID, "error", err)
					continue
				}
			case wire.LobbyUsers:
				// TODO: Handle it. Note: It should be sent only once.
				_, msg, err := wire.DecodeTyped[[]wire.Player](p)
				if err != nil {
					slog.Warn("Could not decode the message", "session", session.ID, "error", err, "event", et.String(), "payload", p)
					continue
				}

				session.lobbyUsers = msg.Content

				for i, player := range session.lobbyUsers {
					// TODO: It can panic, whether int value > i32.
					// TODO: It is not thread-safe.
					if err := b.Send(session.Conn, ReceiveMessage,
						AppendCharacterToLobby(player.Username, model.ClassType(player.ClassType), uint32(i)),
					); err != nil {
						slog.Warn("Error appending lobby users", "session", session.ID, "error", err)
						continue
					}
				}
			case wire.Join:
				_, msg, err := wire.DecodeTyped[wire.Player](p)
				if err != nil {
					slog.Warn("Could not decode the message", "session", session.ID, "error", err, "event", et.String(), "payload", p)
					continue
				}
				if msg.Content.UserID == session.ID {
					continue
				}

				player := msg.Content
				session.lobbyUsers = append(session.lobbyUsers, player)

				idx := uint32(len(session.lobbyUsers))
				if err := b.Send(session.Conn, ReceiveMessage,
					AppendCharacterToLobby(player.Username, model.ClassType(player.ClassType), idx),
				); err != nil {
					slog.Warn("Error appending lobby user", "session", session.ID, "error", err)
					continue
				}
			case wire.Leave:
				_, msg, err := wire.DecodeTyped[wire.Player](p)
				if err != nil {
					slog.Warn("Could not decode the message", "session", session.ID, "error", err, "event", et.String(), "payload", p)
					continue
				}

				session.lobbyUsers = slices.DeleteFunc(session.lobbyUsers, func(player wire.Player) bool {
					return msg.Content.UserID == player.UserID
				})

				if err := b.Send(session.Conn, ReceiveMessage,
					RemoveCharacterFromLobby(msg.Content.Username),
				); err != nil {
					slog.Warn("Error appending lobby user", "session", session.ID, "error", err)
					continue
				}
			default:
				// Skip and do not handle it.
			}
		}
	}
	return observer, nil
}

func (b *Backend) ConnectToWebSocket(ctx context.Context, session *Session) (*websocket.Conn, error) {
	ws, err := wire.Connect(ctx, b.SignalServerURL, wire.User{
		UserID:   fmt.Sprintf("%d", session.UserID),
		Username: session.Username,
	})
	// CharacterID:        fmt.Sprintf("%d", session.CharacterID),
	// CharacterClassType: int(session.ClassType),

	if err != nil {
		return nil, err
	}
	// TODO: Name the websocket version.
	return ws, nil
}

func (b *Backend) UpdateCharacterInfo(session *Session) error {
	ctx, cancel := context.WithTimeout(context.TODO(), time.Second*3)
	defer cancel()

	user := wire.User{
		UserID:   fmt.Sprintf("%d", session.UserID),
		Username: session.Username,
	}
	character := wire.Character{}

	err := session.wsConn.Write(ctx, websocket.MessageText,
		wire.ComposeTyped[wire.Player](wire.Join, wire.MessageContent[wire.Player]{
			From:    user.UserID,
			Type:    wire.Join,
			Content: wire.Player{User: user, Character: character},
		}))
	if err != nil {
		return err
	}

	// Expect to receive the joined message.
	_, p, err := session.wsConn.Read(ctx)
	if err != nil {
		return err
	}
	if len(p) != 1 || wire.EventType(p[0]) != wire.Joined {
		return fmt.Errorf("expected joined message, got: %s", string(p))
	}

	return nil
}
