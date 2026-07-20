package console

import (
	"github.com/dimspell/gladiator/internal/relayserver"
)

type (
	PeerConn          = relayserver.PeerConn
	Room              = relayserver.Room
	RelayStream       = relayserver.RelayStream
	RelayConn         = relayserver.RelayConn
	RelayPacket       = relayserver.RelayPacket
	RelayServerOption = relayserver.RelayServerOption
	RelayEventHook    = relayserver.RelayEventHook
	RelayEvent        = relayserver.RelayEvent
	RelayMetrics      = relayserver.RelayMetrics
)

// UserSessionProvider is the console-specific session interface that the relay
// server uses to authenticate peers during the handshake.
type UserSessionProvider interface {
	GetUserSession(id int64) (*UserSession, bool)
}

// sessionBridge adapts a UserSessionProvider to relayserver.SessionProvider.
type sessionBridge struct {
	p UserSessionProvider
}

func (b *sessionBridge) SessionExists(id int64) bool {
	if b == nil || b.p == nil {
		return true
	}
	_, ok := b.p.GetUserSession(id)
	return ok
}

// RelayServer wraps relayserver.RelayServer with console-specific session
// integration. Existing callers (RelayService, RegisterRelayHooks, tests) work
// through the promoted *relayserver.RelayServer fields and methods.
type RelayServer struct {
	*relayserver.RelayServer
	Multiplayer UserSessionProvider
}

// NewQUICRelay creates a new RelayServer that delegates to the extracted relay
// implementation in internal/relayserver.
func NewQUICRelay(addr string, multiplayer UserSessionProvider, opts ...RelayServerOption) (*RelayServer, error) {
	var bridge *sessionBridge
	if multiplayer != nil {
		bridge = &sessionBridge{p: multiplayer}
	}
	inner, err := relayserver.NewRelayServer(addr, bridge, opts...)
	if err != nil {
		return nil, err
	}
	return &RelayServer{
		RelayServer: inner,
		Multiplayer: multiplayer,
	}, nil
}

func WithVerifyFunc(f func([]byte) ([]byte, bool)) RelayServerOption {
	return relayserver.WithVerifyFunc(f)
}

func WithEventHooks(join, leave, del RelayEventHook) RelayServerOption {
	return relayserver.WithEventHooks(join, leave, del)
}
