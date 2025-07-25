package console

import (
	"context"
	"fmt"
)

type RelayService struct {
	Server *RelayServer
	cancel context.CancelFunc
}

func NewRelayService(addr string, multiplayer *RoomService) (*RelayService, error) {
	server, err := NewQUICRelay(
		addr,
		multiplayer,
		WithVerifyFunc(verifyRelayPacket),
		WithEventHooks(multiplayer.HandleRelayJoin, multiplayer.HandleRelayLeave, multiplayer.HandleRelayDelete),
	)
	if err != nil {
		return nil, fmt.Errorf("relay failed to listen: %v", err)
	}
	return &RelayService{Server: server}, nil
}

func (r *RelayService) Start(ctx context.Context) error {
	if r == nil || r.Server == nil {
		return nil
	}

	ctx, r.cancel = context.WithCancel(ctx)
	// go r.Server.cleanupPeers()

	r.Server.Start(ctx)
	return nil
}

func (r *RelayService) Stop(ctx context.Context) error {
	if r == nil || r.Server == nil {
		return nil
	}

	if r.cancel != nil {
		r.cancel()
	}

	return nil
}
