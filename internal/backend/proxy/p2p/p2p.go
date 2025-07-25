package p2p

import (
	"context"
	"fmt"
	"log/slog"
	"net"

	"connectrpc.com/connect"
	multiv1 "github.com/dimspell/gladiator/gen/multi/v1"
	"github.com/dimspell/gladiator/gen/multi/v1/multiv1connect"
	"github.com/dimspell/gladiator/internal/app/logger/logging"
	"github.com/dimspell/gladiator/internal/backend/bsession"
	"github.com/dimspell/gladiator/internal/backend/proxy"
	"github.com/dimspell/gladiator/internal/backend/redirect"
	"github.com/dimspell/gladiator/internal/model"
	"github.com/pion/webrtc/v4"
)

var _ proxy.ProxyClient = (*PeerToPeer)(nil)

type ProxyP2P struct {
	ICEServers   []webrtc.ICEServer
	ProxyFactory redirect.ProxyFactory
}

func (p *ProxyP2P) Mode() model.RunMode { return model.RunModeWebRTC }

func (p *ProxyP2P) Create(session *bsession.Session, gameClient multiv1connect.GameServiceClient) proxy.ProxyClient {
	return NewPeerToPeer(session, gameClient, p.ICEServers, p.ProxyFactory)
}

// PeerToPeer implements the Proxy interface for WebRTC-based peer-to-peer connections.
// It manages game rooms, peer connections, and network addressing for multiplayer games
type PeerToPeer struct {
	// A custom IP address to which we will connect to.
	hostIPAddress net.IP

	WebRTCConfig webrtc.Configuration
	ProxyFactory redirect.ProxyFactory

	Session      *bsession.Session
	GameManager  *GameManager
	EventHandler *PeerToPeerMessageHandler

	HostManager       *redirect.HostManager
	GameServiceClient multiv1connect.GameServiceClient
}

// NewPeerToPeer now accepts ICEServers as a slice and ProxyFactory as a separate argument
func NewPeerToPeer(session *bsession.Session, gameClient multiv1connect.GameServiceClient, iceServers []webrtc.ICEServer, proxyFactory redirect.ProxyFactory) *PeerToPeer {
	if proxyFactory == nil {
		proxyFactory = &redirect.DefaultProxyFactory{}
	}

	config := webrtc.Configuration{}
	config.ICEServers = append(config.ICEServers, iceServers...)

	gameManager := &GameManager{
		session: session,
		config:  config,
	}

	hostManager := redirect.NewManager(net.IPv4(127, 0, 0, 1), redirect.WithProxyFactory(proxyFactory))

	p := &PeerToPeer{
		hostIPAddress:     net.IPv4(127, 0, 0, 2),
		WebRTCConfig:      config,
		ProxyFactory:      proxyFactory,
		Session:           session,
		GameManager:       gameManager,
		HostManager:       hostManager,
		GameServiceClient: gameClient,
	}

	handler := &PeerToPeerMessageHandler{
		p.Session.GetUserID(),
		p.Session,
		p.GameManager,
		proxyFactory,
		slog.With("user_id", p.Session.GetUserID()),
	}

	p.EventHandler = handler

	return p
}

func (p *PeerToPeer) CreateRoom(ctx context.Context, params proxy.CreateParams) error {
	p.Close()

	// NEW: Assign IP using HostManager
	userID := p.Session.GetUserID()
	ipStr, err := p.HostManager.AssignIP(fmt.Sprintf("%d", userID))
	if err != nil {
		return fmt.Errorf("failed to assign IP for host: %w", err)
	}
	ipAddr := net.ParseIP(ipStr)
	hostPlayer := p.Session.ToPlayer(ipAddr)

	gameRoom := &Game{
		ID:    params.GameID,
		Host:  hostPlayer,
		Peers: map[int64]*Peer{}, // FIXME: Add size limit
	}

	_, err = p.GameServiceClient.CreateGame(ctx, connect.NewRequest(&multiv1.CreateGameRequest{
		GameName:      params.GameID,
		Password:      params.Password,
		MapId:         multiv1.GameMap(params.MapId),
		HostUserId:    p.Session.UserID,
		HostIpAddress: ipStr,
	}))
	if err != nil {
		return fmt.Errorf("could not create game room: %w", err)
	}

	p.GameManager.Game = gameRoom
	return nil
}

func (p *PeerToPeer) SetRoomReady(ctx context.Context, params proxy.CreateParams) error {
	_, err := p.GameServiceClient.GetGame(ctx, connect.NewRequest(&multiv1.GetGameRequest{
		GameRoomId: params.GameID,
	}))
	if err != nil {
		slog.Info("Failed to get a game room", logging.Error(err))
		return err
	}

	if p.GameManager.Game == nil || p.GameManager.Game.ID != params.GameID {
		return fmt.Errorf("no game room found")
	}

	if err := p.Session.SendSetRoomReady(ctx, params.GameID); err != nil {
		return fmt.Errorf("could not send set room ready: %w", err)
	}

	return nil
}

func (p *PeerToPeer) ListGames(ctx context.Context) ([]model.LobbyRoom, error) {
	ipv4 := net.IPv4(127, 0, 0, 2)

	resp, err := p.GameServiceClient.ListGames(ctx, connect.NewRequest(&multiv1.ListGamesRequest{}))
	if err != nil {
		return nil, fmt.Errorf("could not list games: %w", err)
	}

	var lobbyRooms []model.LobbyRoom
	for _, room := range resp.Msg.GetGames() {
		lobbyRooms = append(lobbyRooms, model.LobbyRoom{
			Name:          room.Name,
			Password:      room.Password,
			HostIPAddress: ipv4,
		})
	}
	return lobbyRooms, nil
}

func (p *PeerToPeer) GetGame(ctx context.Context, roomID string) (*model.LobbyRoom, []model.LobbyPlayer, error) {
	p.Close()

	respGame, err := p.GameServiceClient.GetGame(ctx, connect.NewRequest(&multiv1.GetGameRequest{GameRoomId: roomID}))
	if err != nil {
		return nil, nil, fmt.Errorf("could not get game room: %w", err)
	}

	hostPlayer, err := proxy.FindPlayer(respGame.Msg.Players, respGame.Msg.Game.HostUserId)
	if err != nil {
		return nil, nil, fmt.Errorf("could not find the host player: %w", err)
	}

	gameRoom := &Game{
		ID:    roomID,
		Host:  hostPlayer,
		Peers: map[int64]*Peer{}, // FIXME: Add size limit
	}

	lobbyRoom := &model.LobbyRoom{
		Name:          respGame.Msg.Game.Name,
		Password:      respGame.Msg.Game.Password,
		HostIPAddress: net.IPv4(127, 0, 0, 2),
		MapID:         multiv1.GameMap(respGame.Msg.Game.MapId),
	}

	var lobbyPlayers []model.LobbyPlayer
	for _, player := range respGame.Msg.GetPlayers() {
		peerConnection, err := webrtc.NewPeerConnection(p.WebRTCConfig)
		if err != nil {
			return nil, nil, err
		}

		// Assign IP using HostManager
		ipStr, err := p.HostManager.AssignIP(fmt.Sprintf("%d", player.UserId))
		if err != nil {
			return nil, nil, fmt.Errorf("failed to assign IP for user %d: %w", player.UserId, err)
		}
		ipAddr := net.ParseIP(ipStr)

		peer := &Peer{
			UserID:     player.UserId,
			Addr:       &redirect.Addressing{IP: ipAddr},
			Mode:       redirect.None, // TODO: Get rid of the Mode field
			Connection: peerConnection,
		}
		gameRoom.Peers[player.UserId] = peer

		lobbyPlayers = append(lobbyPlayers, model.LobbyPlayer{
			ClassType: player.ClassType,
			IPAddress: ipAddr.To4(),
			Name:      player.Username,
		})
	}

	p.GameManager.Game = gameRoom

	return lobbyRoom, lobbyPlayers, nil
}

func (p *PeerToPeer) JoinGame(ctx context.Context, roomID string, password string) ([]model.LobbyPlayer, error) {
	respJoin, err := p.GameServiceClient.JoinGame(ctx, connect.NewRequest(&multiv1.JoinGameRequest{
		UserId:     p.Session.UserID,
		GameRoomId: roomID,
		IpAddress:  "",
	}))
	if err != nil {
		return nil, fmt.Errorf("could not join game room: %w", err)
	}

	if p.GameManager.Game == nil {
		return nil, fmt.Errorf("no game mananged for session: %d", p.Session.GetUserID())
	}

	// Assign IP using HostManager
	userID := p.Session.GetUserID()
	ipStr, err := p.HostManager.AssignIP(fmt.Sprintf("%d", userID))
	if err != nil {
		return nil, fmt.Errorf("failed to assign IP for joining user: %w", err)
	}
	ip := net.ParseIP(ipStr)

	peer := &Peer{
		UserID: userID,
		Addr:   &redirect.Addressing{IP: ip},
		Mode:   redirect.None, // TODO: Get rid of the Mode field
	}
	p.GameManager.AddPeer(peer)

	var lobbyPlayers []model.LobbyPlayer
	for _, player := range respJoin.Msg.GetPlayers() {
		if player.UserId == p.Session.UserID {
			continue
		}

		// peer, ok := p.GameManager.GetPeer(player.UserId)
		// if !ok {
		// 	continue
		// }
		peerID := fmt.Sprintf("%d", player.UserId)
		ipStr, ok := p.HostManager.PeerIPs[peerID]
		if !ok {
			continue
		}

		lobbyPlayers = append(lobbyPlayers, model.LobbyPlayer{
			ClassType: player.ClassType,
			IPAddress: net.ParseIP(ipStr).To4(),
			Name:      player.Username,
		})
	}

	panic("implement me")
}

// func (p *PeerToPeer) ConnectToPlayer(ctx context.Context, params proxy.GetPlayerAddrParams) (net.IP, error) {
// 	gameManager, ok := p.GameManager, p.GameManager != nil
// 	if !ok || gameManager.Game == nil {
// 		return nil, fmt.Errorf("no game mananged for session: %d", p.Session.GetUserID())
// 	}
//
// 	peer, ok := gameManager.Game.Peers[params.UserID]
// 	if !ok {
// 		return nil, fmt.Errorf("could not find peer with user ID: %d", params.UserID)
// 	}
//
// 	if peer.Connected == nil {
// 		return nil, fmt.Errorf("peer does not have a connection channel")
// 	}
//
// 	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
// 	defer cancel()
//
// 	select {
// 	case <-ctx.Done():
// 		slog.Error("timeout waiting for peer to connect", "user_id", params.UserID)
// 	case <-peer.Connected:
// 		slog.Debug("peer connected, user ID", "user_id", params.UserID)
// 	}
//
// 	return peer.Addr.IP, nil
// }

// Close closes the connection for a session.
func (p *PeerToPeer) Close() {
	gameManager, ok := p.GameManager, p.GameManager != nil
	if !ok {
		return
	}

	gameManager.Reset()

	// Cleanup all fake hosts/proxies
	if p.HostManager != nil {
		p.HostManager.StopAll()
	}
}

func (p *PeerToPeer) Handle(ctx context.Context, payload []byte) error {
	return p.EventHandler.Handle(ctx, payload)
}
