package webrtc

import (
	"context"

	"github.com/dimspell/gladiator/gen/multi/v1/multiv1connect"
	"github.com/dimspell/gladiator/internal/backend/bsession"
	"github.com/dimspell/gladiator/internal/backend/proxy"
	"github.com/dimspell/gladiator/internal/backend/redirect"
	"github.com/dimspell/gladiator/internal/model"
	"github.com/pion/webrtc/v4"
)

type Factory struct {
	ICEServers   []webrtc.ICEServer
	ProxyFactory redirect.ProxyFactory
}

func (p *Factory) Mode() model.RunMode { return model.RunModeWebRTC }

func (p *Factory) Create(session *bsession.Session, client multiv1connect.GameServiceClient) proxy.ProxyClient {
	return &Instance{}
}

type Instance struct {
	ProxyFactory redirect.ProxyFactory

	Session *bsession.Session

	RoomID string
	Peers  map[string]*Peer
}

func (p *Instance) CreateRoom(ctx context.Context, params proxy.CreateParams) error {
	// TODO implement me
	panic("implement me")
}

func (p *Instance) SetRoomReady(ctx context.Context, params proxy.CreateParams) error {
	// TODO implement me
	panic("implement me")
}

func (p *Instance) ListGames(ctx context.Context) ([]model.LobbyRoom, error) {
	// TODO implement me
	panic("implement me")
}

func (p *Instance) GetGame(ctx context.Context, roomID string) (*model.LobbyRoom, []model.LobbyPlayer, error) {
	// TODO implement me
	panic("implement me")
}

func (p *Instance) JoinGame(ctx context.Context, roomID string, password string) ([]model.LobbyPlayer, error) {
	// TODO implement me
	panic("implement me")
}

func (p *Instance) Close() {
	// TODO implement me
	panic("implement me")
}

func (p *Instance) Handle(ctx context.Context, payload []byte) error {
	// TODO implement me
	panic("implement me")
}

type Peer struct {
	ID string `json:"id"`
}
