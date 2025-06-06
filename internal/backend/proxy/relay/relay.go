package relay

import (
	"context"
	"log/slog"
	"net"

	"github.com/dimspell/gladiator/internal/backend/bsession"
	"github.com/dimspell/gladiator/internal/backend/proxy"
)

var _ proxy.ProxyClient = (*Relay)(nil)

type ProxyRelay struct {
	// TODO: Manage a list of opened proxies and help to close them
}

func (p *ProxyRelay) Create(session *bsession.Session) proxy.ProxyClient {
	return NewRelay(nil, session)
}

type Relay struct {
	config *Config

	session *bsession.Session
	router  *PacketRouter
	manager *HostManager
}

type Config struct {
}

func NewRelay(config *Config, session *bsession.Session) *Relay {
	cfg := &Config{}

	router := &PacketRouter{
		logger:  slog.With("proxy", "relay"),
		session: session,
	}

	manager := NewManager()

	return &Relay{cfg, session, router, manager}
}

func (w *Relay) GetHostIP(ip net.IP) net.IP {
	return net.IPv4(127, 0, 0, 2)
}

func (w *Relay) CreateRoom(params proxy.CreateParams) (net.IP, error) {
	// TODO implement me
	// w.router.Reset()

	panic("implement me")
}

func (w *Relay) HostRoom(ctx context.Context, params proxy.HostParams) error {
	// TODO implement me
	panic("implement me")
}

func (w *Relay) SelectGame(data proxy.GameData) error {
	// TODO implement me
	// w.router.Reset()

	panic("implement me")
}

func (w *Relay) GetPlayerAddr(params proxy.GetPlayerAddrParams) (net.IP, error) {
	// TODO implement me
	panic("implement me")
}

func (w *Relay) Join(ctx context.Context, params proxy.JoinParams) (net.IP, error) {
	// TODO implement me
	panic("implement me")
}

func (w *Relay) ConnectToPlayer(ctx context.Context, params proxy.GetPlayerAddrParams) (net.IP, error) {
	// TODO implement me
	panic("implement me")
}

func (w *Relay) Close() {
	// TODO implement me
	panic("implement me")
}

func (w *Relay) Handle(ctx context.Context, payload []byte) error {
	return w.router.Handle(ctx, payload)
}
