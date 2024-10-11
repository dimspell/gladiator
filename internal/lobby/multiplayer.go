package lobby

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/dimspell/gladiator/internal/icesignal"
)

type Multiplayer struct {
	done context.CancelFunc

	// Sessions
	sessionMutex sync.RWMutex
	sessions     map[string]*UserSession

	// Presence chan UserSession
	Messages chan icesignal.Message
}

func NewMultiplayer(ctx context.Context) *Multiplayer {
	ctx, done := context.WithCancel(ctx)

	mp := &Multiplayer{
		sessions: make(map[string]*UserSession),
		Messages: make(chan icesignal.Message),
		done:     done,
	}

	go mp.Run(ctx)
	return mp
}

func (mp *Multiplayer) Close() { mp.done() }

func (mp *Multiplayer) Reset() {
	mp.forEachSession(func(userSession *UserSession) bool {
		_ = userSession.Conn.CloseNow()
		return true
	})
	clear(mp.sessions)
	close(mp.Messages)
}

func (mp *Multiplayer) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			mp.Reset()
			return
		case msg, ok := <-mp.Messages:
			if !ok {
				return
			}

			slog.Debug("Received a signal message", "type", msg.Type.String(), "from", msg.From, "to", msg.To)

			switch msg.Type {
			case icesignal.Chat:
				mp.BroadcastMessage(ctx, compose(icesignal.Chat, icesignal.Message{
					From:    msg.From,
					Content: msg.Content,
				}))
			case icesignal.HandshakeRequest:
				mp.SendJoin(ctx, msg)
			case icesignal.RTCOffer, icesignal.RTCAnswer, icesignal.RTCICECandidate:
				mp.ForwardRTCMessage(ctx, msg)
			default:
				// Do nothing
			}
		}
	}
}

func (mp *Multiplayer) HandleSession(ctx context.Context, session *UserSession) error {
	// Add user to the list of connected players.
	mp.SetPlayerConnected(session)

	// Remove the player
	defer mp.SetPlayerDisconnected(session)

	// Handle all the incoming messages.
	for {
		// Register that the user is still being active.
		session.LastSeen = time.Now().In(time.UTC)

		payload, err := session.ReadNext(ctx)
		if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
			return err
		}
		if err != nil {
			slog.Error("Could not handle the message", "error", err)
			return err
		}
		if err := mp.EnqueueMessage(payload); err != nil {
			return err
		}
	}
}

func (mp *Multiplayer) SendJoin(ctx context.Context, msg icesignal.Message) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*3)
	defer cancel()

	if user, ok := mp.getSession(msg.From); ok {
		user.SendMessage(ctx, icesignal.HandshakeResponse, icesignal.Message{
			Type:    icesignal.HandshakeResponse,
			To:      msg.From,
			Content: msg.From,
		})
	}
	mp.BroadcastMessage(ctx, compose(icesignal.Join, icesignal.Message{
		Type:    icesignal.Join,
		Content: msg.Content,
	}))
}

func (mp *Multiplayer) ForwardRTCMessage(ctx context.Context, msg icesignal.Message) {
	slog.Debug("Forwarding RTC message", "type", msg.Type.String(), "from", msg.From, "to", msg.To)

	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	if user, ok := mp.getSession(msg.To); ok {
		user.SendMessage(ctx, msg.Type, msg)
	}
}

func (mp *Multiplayer) EnqueueMessage(payload []byte) error {
	if len(payload) == 0 {
		return io.ErrShortBuffer
	}

	// if len(payload) == 1 && payload[0] == 0x00 {
	// 	ctx, cancel := context.WithTimeout(ctx, time.Second)
	// 	defer cancel()
	//
	// 	if err := conn.Write(ctx, websocket.MessageText, []byte{0x00}); err != nil {
	// 		return err
	// 	}
	// }

	var m icesignal.Message
	if err := icesignal.DefaultCodec.Unmarshal(payload[1:], &m); err != nil {
		return err
	}
	mp.Messages <- m
	return nil
}
