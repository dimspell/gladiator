package relay

import (
	"context"
	"fmt"
	"github.com/dimspell/gladiator/internal/app/logger/logging"
	"log/slog"
	"net"

	"github.com/dimspell/gladiator/internal/backend/bsession"
	"github.com/dimspell/gladiator/internal/backend/proxy"
	"github.com/dimspell/gladiator/internal/backend/redirect"
	"github.com/dimspell/gladiator/internal/model"
	"github.com/dimspell/gladiator/probe"
)

var _ proxy.ProxyClient = (*Relay)(nil)

// ProxyRelay represents the configuration for setting up a local UDP/TCP proxy
// that forwards traffic to a remote relay server.
type ProxyRelay struct {
	// Proxies []*Relay

	// RelayServerAddr is the address (IP:port) of the remote relay server to
	// which the proxy will forward all client traffic.
	RelayServerAddr string

	IPPrefix net.IP
}

func (p *ProxyRelay) Mode() model.RunMode { return model.RunModeRelay }

func (p *ProxyRelay) Create(session *bsession.Session) proxy.ProxyClient {
	px := NewRelay(p, session)

	// TODO: Manage a list of opened proxies and help to close them
	// FIXME: Not threadsafe, no closer
	// p.Proxies = append(p.Proxies, px)

	return px
}

type Relay struct {
	session *bsession.Session
	router  *PacketRouter
}

func NewRelay(config *ProxyRelay, session *bsession.Session) *Relay {
	ipPrefix := config.IPPrefix
	if ipPrefix == nil {
		ipPrefix = net.IPv4(127, 0, 0, 0)
	}

	router := &PacketRouter{
		relayAddr: config.RelayServerAddr,
		logger:    slog.With(slog.String("proxy", "relay"), slog.String("sessionId", session.ID)),
		selfID:    remoteID(session.UserID),
		session:   session,
		manager:   redirect.NewManager(ipPrefix.To4()),
	}

	return &Relay{
		session,
		router,
	}
}

func remoteID(i int64) string { return fmt.Sprintf("%d", i) }

func (r *Relay) GetHostIP(ip net.IP) net.IP {
	return net.IPv4(127, 0, 0, 2)
}

func (r *Relay) CreateRoom(params proxy.CreateParams) (net.IP, error) {
	ctx := context.Background()
	roomID := params.GameID

	r.router.Reset()
	r.router.selfID = remoteID(r.session.UserID)
	r.router.currentHostID = remoteID(r.session.UserID)
	r.router.roomID = roomID

	if err := r.router.connect(ctx, roomID); err != nil {
		return nil, fmt.Errorf("failed connect to the relay server: %w", err)
	}

	return net.IPv4(127, 0, 0, 1), nil
}

func (r *Relay) HostRoom(ctx context.Context, params proxy.HostParams) error {
	if err := r.session.SendSetRoomReady(ctx, params.GameID); err != nil {
		return fmt.Errorf("could not send set room ready: %w", err)
	}

	// A scheduled interval to keep connection to the relay server
	// Note: In case of players playing alone
	r.router.keepAliveHost(ctx)

	// Probe to check if the game server is still running
	onDisconnect := func() {
		slog.Warn("Game server went offline")
		r.router.Reset()
	}
	if err := probe.StartProbeTCP(ctx, net.JoinHostPort("127.0.0.1", "6114"), onDisconnect); err != nil {
		return fmt.Errorf("failed start the game server probe: %w", err)
	}

	return nil
}

func (r *Relay) SelectGame(data proxy.GameData) error {
	r.router.Reset()
	r.router.selfID = remoteID(r.session.UserID)
	r.router.roomID = data.Game.GameId

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

		ip, err := r.router.manager.AssignIP(peerID)
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

	ip, ok := r.router.manager.PeerIPs[peerID]
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

	for peerID, ipAddress := range r.router.manager.PeerIPs {
		onUDPMessage := func(p []byte) error {
			return r.router.sendPacket(RelayPacket{
				Type:    "udp",
				RoomID:  roomID,
				ToID:    peerID,
				Payload: p,
			})
		}

		if peerID == hostID {
			onTCPMessage := func(p []byte) error {
				return r.router.sendPacket(RelayPacket{
					Type:    "tcp",
					RoomID:  roomID,
					ToID:    peerID,
					Payload: p,
				})
			}

			onHostDisconnected := func(host *redirect.FakeHost) {
				slog.Warn("Host went offline", logging.PeerID(peerID), "ip", ipAddress)
				r.router.stop(host, peerID, ipAddress)
			}
			_, err := r.router.manager.StartHost(ctx, peerID, ipAddress, 6114, 6113, onTCPMessage, onUDPMessage, onHostDisconnected)
			if err != nil {
				return nil, err
			}

			//if err := probe.StartProbeTCP(ctx, net.JoinHostPort(ipAddress, "6114"), onHostDisconnected); err != nil {
			//	return nil, fmt.Errorf("failed start the game server probe: %w", err)
			//}
		} else {
			onHostDisconnected := func(host *redirect.FakeHost) {
				slog.Warn("Host went offline", logging.PeerID(peerID), "ip", ipAddress)
				r.router.stop(host, peerID, ipAddress)
			}
			if _, err := r.router.manager.StartHost(ctx, peerID, ipAddress, 0, 6113, nil, onUDPMessage, onHostDisconnected); err != nil {
				return nil, err
			}
		}
	}

	// go r.router.manager.CleanupInactive()

	return net.IPv4(127, 0, 0, 1), nil
}

func (r *Relay) ConnectToPlayer(ctx context.Context, params proxy.GetPlayerAddrParams) (net.IP, error) {
	return r.GetPlayerAddr(params)
}

func (r *Relay) Close() {
	r.router.Reset()
}

func (r *Relay) Handle(ctx context.Context, payload []byte) error {
	return r.router.Handle(ctx, payload)
}

func (r *Relay) Debug() any {
	hosts := r.router.manager.Hosts
	peerHosts := r.router.manager.PeerHosts
	ipToPeerID := r.router.manager.IPToPeerID
	peerIPs := r.router.manager.PeerIPs
	currentHostID := r.router.currentHostID
	selfID := r.router.selfID

	var state = struct {
		Hosts         map[string]*redirect.FakeHost
		PeerHosts     map[string]*redirect.FakeHost
		IPToPeerID    map[string]string
		PeerIPs       map[string]string
		CurrentHostID string
		SelfID        string
	}{
		Hosts:         hosts,
		PeerHosts:     peerHosts,
		IPToPeerID:    ipToPeerID,
		PeerIPs:       peerIPs,
		CurrentHostID: currentHostID,
		SelfID:        selfID,
	}

	return state
}
