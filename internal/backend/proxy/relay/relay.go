package relay

import (
	"context"
	"fmt"
	"log/slog"
	"net"

	"github.com/dimspell/gladiator/internal/backend/bsession"
	"github.com/dimspell/gladiator/internal/backend/proxy"
)

var _ proxy.ProxyClient = (*Relay)(nil)

type ProxyRelay struct {
	// Proxies []*Relay
	RelayAddr string
}

func (p *ProxyRelay) Create(session *bsession.Session) proxy.ProxyClient {
	px := NewRelay(&Config{RelayAddr: p.RelayAddr}, session)
	// TODO: Manage a list of opened proxies and help to close them
	// FIXME: Not threadsafe, no closer
	// p.Proxies = append(p.Proxies, px)
	return px
}

type Relay struct {
	config *Config

	session *bsession.Session
	router  *PacketRouter
	manager *HostManager
	players map[string]net.IP
}

type Config struct {
	RelayAddr string
}

func NewRelay(config *Config, session *bsession.Session) *Relay {
	cfg := &Config{}

	manager := NewManager()

	router := &PacketRouter{
		relayAddr: config.RelayAddr,
		logger:    slog.With(slog.String("proxy", "relay"), slog.String("sessionId", session.ID)),
		selfID:    remoteID(session.UserID),
		session:   session,
		manager:   manager,
	}

	return &Relay{
		cfg,
		session,
		router,
		manager,
		make(map[string]net.IP),
	}
}

func remoteID(i int64) string { return fmt.Sprintf("%d", i) }

func (r *Relay) GetHostIP(ip net.IP) net.IP {
	return net.IPv4(127, 0, 0, 2)
}

func (r *Relay) CreateRoom(params proxy.CreateParams) (net.IP, error) {
	r.router.Reset()
	r.router.selfID = remoteID(r.session.UserID)
	r.router.currentHostID = remoteID(r.session.UserID)

	ctx := context.Background()
	roomID := params.GameID

	if err := r.router.connect(ctx, roomID); err != nil {
		return nil, fmt.Errorf("failed connect to the relay server: %w", err)
	}

	return net.IPv4(127, 0, 0, 1), nil
}

func (r *Relay) HostRoom(ctx context.Context, params proxy.HostParams) error {
	if err := r.session.SendSetRoomReady(ctx, params.GameID); err != nil {
		return fmt.Errorf("could not send set room ready: %w", err)
	}
	return nil
}

func (r *Relay) SelectGame(data proxy.GameData) error {
	r.router.Reset()
	r.router.selfID = remoteID(r.session.UserID)

	host, err := data.FindHostUser()
	if err != nil {
		return err
	}
	r.router.currentHostID = remoteID(host.UserID)

	for _, player := range data.Players {
		peerID := remoteID(player.UserId)
		if peerID == r.router.selfID {
			continue
		}

		ip, err := r.router.manager.assignIP(peerID)
		if err != nil {
			return err
		}

		r.router.logger.Debug("assigned IP to a player", slog.Int64("remoteID", player.UserId), slog.String("player", player.Username), slog.String("ip", ip))
	}

	return nil
}

func (r *Relay) GetPlayerAddr(params proxy.GetPlayerAddrParams) (net.IP, error) {
	peerID := remoteID(params.UserID)
	if peerID == r.router.selfID {
		return net.IPv4(127, 0, 0, 1), nil
	}

	ip, ok := r.router.manager.peerIPs[peerID]
	if !ok {
		return nil, fmt.Errorf("not found the IP for a peer with ID %s", peerID)
	}
	ipv4 := net.ParseIP(ip)
	if ipv4 == nil {
		return nil, fmt.Errorf("invalid IP %s", ip)
	}
	return ipv4, nil
}

func (r *Relay) Join(ctx context.Context, params proxy.JoinParams) (net.IP, error) {
	roomID := params.GameID
	if err := r.router.connect(ctx, roomID); err != nil {
		return nil, fmt.Errorf("failed connect to the relay server: %w", err)
	}

	if err := r.router.sendPacket(RelayPacket{
		Type:    "broadcast",
		RoomID:  roomID,
		Payload: []byte("Hello everyone!"),
	}); err != nil {
		return nil, err
	}

	hostID := remoteID(params.HostUserID)

	for peerID, ipAddress := range r.router.manager.peerIPs {
		if peerID == hostID {
			onTCPMessage := func(p []byte) error {
				return r.router.sendPacket(RelayPacket{
					Type:    "tcp",
					RoomID:  roomID,
					ToID:    peerID,
					Payload: p,
				})
			}
			onUDPMessage := func(p []byte) error {
				return r.router.sendPacket(RelayPacket{
					Type:    "udp",
					RoomID:  roomID,
					ToID:    peerID,
					Payload: p,
				})
			}

			if _, err := r.router.manager.StartHost(peerID, ipAddress, 6114, 6113, onTCPMessage, onUDPMessage); err != nil {
				return nil, err
			}
		} else {
			onUDPMessage := func(p []byte) error {
				return r.router.sendPacket(RelayPacket{
					Type:    "udp",
					RoomID:  roomID,
					ToID:    peerID,
					Payload: p,
				})
			}

			if _, err := r.router.manager.StartHost(peerID, ipAddress, 0, 6113, nil, onUDPMessage); err != nil {
				return nil, err
			}
		}
	}

	return net.IPv4(127, 0, 0, 1), nil
}

func (r *Relay) ConnectToPlayer(ctx context.Context, params proxy.GetPlayerAddrParams) (net.IP, error) {
	return r.GetPlayerAddr(params)
}

func (r *Relay) Close() {
	r.players = make(map[string]net.IP)
	r.router.Reset()
}

func (r *Relay) Handle(ctx context.Context, payload []byte) error {
	return r.router.Handle(ctx, payload)
}
