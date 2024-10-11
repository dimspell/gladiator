package lobby

import (
	"context"
	"sync"

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
		_ = userSession.wsConn.CloseNow()
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
			mp.HandleIncomingMessage(ctx, msg)
		}
	}
}
