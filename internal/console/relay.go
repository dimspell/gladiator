package console

import (
	"context"
	"fmt"
)

type Relay struct {
	Server *RelayServer
	cancel context.CancelFunc
}

func NewRelay(addr string, multiplayer *Multiplayer) (*Relay, error) {
	server, err := NewQUICRelay(
		addr,
		multiplayer,
		WithVerifyFunc(verify),
		WithEventHooks(multiplayer.HandleRelayJoin, multiplayer.HandleRelayLeave, multiplayer.HandleRelayDelete),
	)
	if err != nil {
		return nil, fmt.Errorf("relay failed to listen: %v", err)
	}
	return &Relay{Server: server}, nil
}

func (r *Relay) Start(ctx context.Context) error {
	if r == nil || r.Server == nil {
		return nil
	}

	ctx, r.cancel = context.WithCancel(ctx)
	// go r.Server.cleanupPeers()

	return r.Server.Start(ctx)
}

func (r *Relay) Stop(ctx context.Context) error {
	if r == nil || r.Server == nil {
		return nil
	}

	if r.cancel != nil {
		r.cancel()
	}

	return nil
}
