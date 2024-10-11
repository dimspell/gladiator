package backend

import (
	"context"
	"encoding/binary"
	"fmt"
	"log/slog"
	"math"
	"net"
	"sync"

	"github.com/coder/websocket"
	"github.com/dimspell/gladiator/internal/icesignal"
	"github.com/dimspell/gladiator/model"
	"github.com/google/uuid"
)

type Session struct {
	ID          string
	UserID      int64
	Username    string
	CharacterID int64
	ClassType   model.ClassType

	Conn         net.Conn
	onceObserver sync.Once
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

	if PacketLogger != nil {
		PacketLogger.Debug("Sent",
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

func (b *Backend) RegisterNewObserver(session *Session) error {
	// TODO: make sure the websocket is not kept forever
	// ctx, done := context.WithCancel(context.TODO())
	// session.done = done

	wsConn, err := b.ConnectToWebSocket(session)
	if err != nil {
		return err
	}
	observer := func(session *Session) {
		for {
			_, p, err := wsConn.Read(context.TODO())
			if err != nil {
				slog.Error("Error reading from WebSocket", "session", session.ID, "error", err)
				return
			}
			slog.Debug("Received packet", "session", session.ID, "packet", p)
		}
	}

	session.onceObserver.Do(func() {
		go observer(session)
	})
	return nil
}

func (b *Backend) ConnectToWebSocket(session *Session) (*websocket.Conn, error) {
	ws, err := icesignal.Connect(context.TODO(), b.SignalServerURL, icesignal.Player{
		ID:                 fmt.Sprintf("%d", session.UserID),
		CharacterID:        fmt.Sprintf("%d", session.CharacterID),
		CharacterClassType: int(session.ClassType),
	})
	if err != nil {
		return nil, err
	}
	return ws, nil
}
