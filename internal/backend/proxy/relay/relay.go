// Package relay provides the implementation of a relay-based packet router for multiplayer networking.
package relay

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"sync"

	"connectrpc.com/connect"
	multiv1 "github.com/dimspell/gladiator/gen/multi/v1"
	"github.com/dimspell/gladiator/gen/multi/v1/multiv1connect"
	"github.com/dimspell/gladiator/internal/app/logger/logging"
	"github.com/dimspell/gladiator/internal/backend/bsession"
	"github.com/dimspell/gladiator/internal/backend/proxy"
	"github.com/dimspell/gladiator/internal/backend/redirect"
	"github.com/dimspell/gladiator/internal/model"
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

func (p *ProxyRelay) Create(session *bsession.Session, client multiv1connect.GameServiceClient) proxy.ProxyClient {
	px := NewRelay(p, client, session)

	// TODO: Manage a list of opened proxies and help to close them
	// FIXME: Not threadsafe, no closer
	// p.Proxies = append(p.Proxies, px)

	return px
}

type Relay struct {
	mu                sync.Mutex
	session           *bsession.Session
	router            *PacketRouter
	GameServiceClient multiv1connect.GameServiceClient
}

func NewRelay(config *ProxyRelay, client multiv1connect.GameServiceClient, session *bsession.Session) *Relay {
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
		session:           session,
		router:            router,
		GameServiceClient: client,
	}
}

func (r *Relay) CreateRoom(ctx context.Context, params proxy.CreateParams) error {
	roomID := params.GameID

	r.router.Reset()
	r.router.selfID = remoteID(r.session.UserID)
	r.router.currentHostID = remoteID(r.session.UserID)
	r.router.roomID = roomID

	if err := r.router.connect(ctx, roomID); err != nil {
		return fmt.Errorf("failed connect to the relay server: %w", err)
	}

	_, err := r.GameServiceClient.CreateGame(ctx, connect.NewRequest(&multiv1.CreateGameRequest{
		GameName:      params.GameID,
		Password:      params.Password,
		MapId:         multiv1.GameMap(params.MapId),
		HostUserId:    r.session.UserID,
		HostIpAddress: "",
	}))
	if err != nil {
		return fmt.Errorf("could not create game room: %w", err)
	}

	return nil
}

func (r *Relay) SetRoomReady(ctx context.Context, params proxy.CreateParams) error {
	respGame, err := r.GameServiceClient.GetGame(ctx, connect.NewRequest(&multiv1.GetGameRequest{
		GameRoomId: params.GameID,
	}))
	if err != nil {
		slog.Info("Failed to get a game room", logging.Error(err))
		return err
	}

	if respGame.Msg.Game.MapId != multiv1.GameMap(params.MapId) {
		return fmt.Errorf("incorrect map id: %d", respGame.Msg.Game.MapId)
	}

	if err := r.session.SendSetRoomReady(ctx, params.GameID); err != nil {
		return fmt.Errorf("could not send set room ready: %w", err)
	}

	// A scheduled interval to keep connection to the relay server
	// Note: In case of players playing alone
	// r.router.keepAliveHost(ctx)

	// Probe to check if the game server is still running
	// onDisconnect := func() {
	//	slog.Warn("Game server went offline")
	//	r.router.Reset()
	//	r.router.disconnect()
	// }
	// if err := probe.StartProbeTCP(ctx, net.JoinHostPort("127.0.0.1", "6114"), onDisconnect); err != nil {
	//	return fmt.Errorf("failed start the game server probe: %w", err)
	// }
	return nil
}

func (r *Relay) ListGames(ctx context.Context) ([]model.LobbyRoom, error) {
	resp, err := r.GameServiceClient.ListGames(ctx, connect.NewRequest(&multiv1.ListGamesRequest{}))
	if err != nil {
		return nil, fmt.Errorf("could not list games: %w", err)
	}

	var lobbyRooms []model.LobbyRoom
	for _, room := range resp.Msg.GetGames() {
		lobbyRooms = append(lobbyRooms, model.LobbyRoom{
			Name:          room.Name,
			Password:      room.Password,
			HostIPAddress: net.IPv4(127, 0, 0, 2).To4(),
		})
	}
	return lobbyRooms, nil
}

func (r *Relay) GetGame(ctx context.Context, roomID string) (*model.LobbyRoom, []model.LobbyPlayer, error) {
	r.router.Reset()

	respGame, err := r.GameServiceClient.GetGame(ctx, connect.NewRequest(&multiv1.GetGameRequest{GameRoomId: roomID}))
	if err != nil {
		return nil, nil, fmt.Errorf("could not get game room: %w", err)
	}

	hostPlayer, err := proxy.FindPlayer(respGame.Msg.Players, respGame.Msg.Game.HostUserId)
	if err != nil {
		return nil, nil, fmt.Errorf("could not find the host player: %w", err)
	}

	var lobbyPlayers []model.LobbyPlayer
	for _, player := range respGame.Msg.Players {
		peerID := remoteID(player.UserId)
		if peerID == r.router.selfID {
			continue
		}

		ip, err := r.router.manager.AssignIP(peerID)
		if err != nil {
			return nil, nil, fmt.Errorf("could not assign ip: %w", err)
		}

		lobbyPlayers = append(lobbyPlayers, model.LobbyPlayer{
			ClassType: player.ClassType,
			IPAddress: net.ParseIP(ip).To4(),
			Name:      player.Username,
		})
	}

	r.router.selfID = remoteID(r.session.UserID)
	r.router.roomID = roomID
	r.router.currentHostID = remoteID(hostPlayer.UserID)

	lobbyRoom := &model.LobbyRoom{
		Name:          respGame.Msg.Game.Name,
		Password:      respGame.Msg.Game.Password,
		HostIPAddress: net.IPv4(127, 0, 0, 2),
		MapID:         multiv1.GameMap(respGame.Msg.Game.MapId),
	}

	return lobbyRoom, lobbyPlayers, nil
}

func (r *Relay) JoinGame(ctx context.Context, roomID string, password string) ([]model.LobbyPlayer, error) {
	respGame, err := r.GameServiceClient.GetGame(ctx, connect.NewRequest(&multiv1.GetGameRequest{GameRoomId: roomID}))
	if err != nil {
		return nil, fmt.Errorf("could not get game room: %w", err)
	}

	if err := r.router.connect(ctx, roomID); err != nil {
		return nil, fmt.Errorf("failed connect to the relay server: %w", err)
	}

	respJoin, err := r.GameServiceClient.JoinGame(ctx, connect.NewRequest(&multiv1.JoinGameRequest{
		UserId:     r.session.UserID,
		GameRoomId: roomID,
		IpAddress:  "",
	}))
	if err != nil {
		return nil, fmt.Errorf("could not join game room: %w", err)
	}

	hostPlayer, err := proxy.FindPlayer(respGame.Msg.GetPlayers(), respGame.Msg.GetGame().GetHostUserId())
	if err != nil {
		return nil, fmt.Errorf("could not find the host player: %w", err)
	}
	hostID := remoteID(hostPlayer.UserID)

	var lobbyPlayers []model.LobbyPlayer
	for _, player := range respJoin.Msg.GetPlayers() {
		if player.UserId == r.session.UserID {
			continue
		}

		peerID := remoteID(player.UserId)
		ipAddress, ok := r.router.manager.PeerIPs[peerID]
		if !ok {
			return nil, fmt.Errorf("not found the IP for a peer with ID %s", peerID)
		}
		ipv4 := net.ParseIP(ipAddress).To4()
		if ipv4 == nil {
			return nil, fmt.Errorf("invalid IP %s", ipAddress)
		}

		r.router.logger.Debug("Starting fake host for", logging.PeerID(peerID), "host", peerID == hostID)

		var tcpPort int
		if peerID == r.router.currentHostID {
			tcpPort = 6114
		}
		onTCPMessage := r.router.onTCPMessage(roomID, peerID)
		onUDPMessage := r.router.onUDPMessage(roomID, peerID)
		onHostDisconnected := func(host *redirect.FakeHost, forced bool) {
			slog.Warn("Host went offline", logging.PeerID(peerID), "ip", host.AssignedIP, "forced", forced)
			if forced {
				r.router.disconnect()
				r.router.Reset()
			} else {
				r.router.stop(host)
			}
		}

		_, err := r.router.manager.StartHost(ctx, peerID, ipAddress, tcpPort, 6113, onTCPMessage, onUDPMessage, onHostDisconnected)
		if err != nil {
			return nil, err
		}

		lobbyPlayers = append(lobbyPlayers, model.LobbyPlayer{
			ClassType: player.ClassType,
			IPAddress: net.ParseIP(ipAddress).To4(),
			Name:      player.Username,
		})
	}

	return lobbyPlayers, nil
}

func remoteID(i int64) string { return fmt.Sprintf("%d", i) }

func (r *Relay) Close() {
	r.router.Reset()
}

func (r *Relay) Handle(ctx context.Context, payload []byte) error {
	return r.router.Handle(ctx, payload)
}
