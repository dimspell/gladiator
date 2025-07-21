package p2p

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/dimspell/gladiator/internal/backend/bsession"
	"github.com/dimspell/gladiator/internal/backend/proxy"
	"github.com/dimspell/gladiator/internal/backend/redirect"
	"github.com/dimspell/gladiator/internal/model"
	"github.com/pion/webrtc/v4"
)

var _ proxy.ProxyClient = (*PeerToPeer)(nil)

type ProxyP2P struct {
	ICEServers []webrtc.ICEServer
}

func (p *ProxyP2P) Mode() model.RunMode { return model.RunModeWebRTC }

func (p *ProxyP2P) Create(session *bsession.Session) proxy.ProxyClient {
	return NewPeerToPeer(session, p.ICEServers...)
}

// PeerToPeer implements the Proxy interface for WebRTC-based peer-to-peer connections.
// It manages game rooms, peer connections, and network addressing for multiplayer games
type PeerToPeer struct {
	// A custom IP address to which we will connect to.
	hostIPAddress net.IP

	WebRTCConfig   webrtc.Configuration
	NewTCPRedirect redirect.NewRedirect
	NewUDPRedirect redirect.NewRedirect

	Session      *bsession.Session
	GameManager  *GameManager
	EventHandler *PeerToPeerMessageHandler

	HostManager *redirect.HostManager // NEW: HostManager for IP/proxy management
}

func NewPeerToPeer(session *bsession.Session, iceServers ...webrtc.ICEServer) *PeerToPeer {
	config := webrtc.Configuration{}
	config.ICEServers = append(config.ICEServers, iceServers...)

	gameManager := &GameManager{
		session: session,
		config:  config,
	}

	// NEW: Initialize HostManager with 127.0.0.1 prefix
	hostManager := redirect.NewManager(net.IPv4(127, 0, 0, 1))

	p := &PeerToPeer{
		hostIPAddress:  net.IPv4(127, 0, 1, 2),
		WebRTCConfig:   config,
		NewTCPRedirect: redirect.NewTCPRedirect,
		NewUDPRedirect: redirect.NewUDPRedirect,
		Session:        session,
		GameManager:    gameManager,
		HostManager:    hostManager, // NEW
	}

	handler := &PeerToPeerMessageHandler{
		p.Session.GetUserID(),
		p.Session,
		p.GameManager,
		p.NewTCPRedirect,
		p.NewUDPRedirect,
		slog.With("user_id", p.Session.GetUserID()),
	}

	p.EventHandler = handler

	return p
}

// CreateRoom creates a new game room and assigns the session as the host.
// Returns the assigned IP address for the host player
func (p *PeerToPeer) CreateRoom(ctx context.Context, params proxy.CreateParams) (net.IP, error) {
	p.GameManager.Reset()

	// NEW: Assign IP using HostManager
	userID := p.Session.GetUserID()
	ipStr, err := p.HostManager.AssignIP(fmt.Sprintf("%d", userID))
	if err != nil {
		return nil, fmt.Errorf("failed to assign IP for host: %w", err)
	}
	ipAddr := net.ParseIP(ipStr)
	hostPlayer := p.Session.ToPlayer(ipAddr)

	gameRoom := &Game{
		ID:    params.GameID,
		Host:  hostPlayer,
		Peers: map[int64]*Peer{}, // FIXME: Add size limit
	}

	p.GameManager.Game = gameRoom

	return ipAddr, nil
}

func (p *PeerToPeer) HostRoom(ctx context.Context, params proxy.HostParams) error {
	if p.GameManager.Game == nil || p.GameManager.Game.ID != params.GameID {
		return fmt.Errorf("no game room found")
	}

	if err := p.Session.SendSetRoomReady(ctx, params.GameID); err != nil {
		return fmt.Errorf("could not send set room ready: %w", err)
	}

	return nil
}

func (p *PeerToPeer) GetHostIP(hostIpAddress net.IP) net.IP {
	return p.hostIPAddress
}

func (p *PeerToPeer) SelectGame(params proxy.GameData) error {
	p.GameManager.Reset()

	hostPlayer, err := params.FindHostUser()
	if err != nil {
		return err
	}

	gameRoom := &Game{
		ID:    params.Game.GameId,
		Host:  hostPlayer,
		Peers: map[int64]*Peer{}, // FIXME: Add size limit
	}

	for _, player := range params.ToWirePlayers() {
		peerConnection, err := webrtc.NewPeerConnection(p.WebRTCConfig)
		if err != nil {
			return err
		}

		// Assign IP using HostManager
		ipStr, err := p.HostManager.AssignIP(fmt.Sprintf("%d", player.UserID))
		if err != nil {
			return fmt.Errorf("failed to assign IP for user %d: %w", player.UserID, err)
		}
		ipAddr := net.ParseIP(ipStr)

		peer := &Peer{
			UserID:     player.UserID,
			Addr:       &redirect.Addressing{IP: ipAddr},
			Mode:       redirect.None, // TODO: Get rid of the Mode field
			Connection: peerConnection,
		}
		gameRoom.Peers[player.UserID] = peer
	}

	p.GameManager.Game = gameRoom

	return nil
}

func (p *PeerToPeer) GetPlayerAddr(params proxy.GetPlayerAddrParams) (net.IP, error) {
	peer, ok := p.GameManager.GetPeer(params.UserID)
	if !ok {
		return nil, fmt.Errorf("could not find peer with user ID: %d", params.UserID)
	}

	return peer.Addr.IP, nil
}

func (p *PeerToPeer) Join(ctx context.Context, params proxy.JoinParams) (net.IP, error) {
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

	for _, pr := range p.GameManager.Game.Peers {
		ch := make(chan struct{}, 1)
		pr.Connected = ch
	}

	return net.IPv4(127, 0, 0, 1), nil
}

func (p *PeerToPeer) ConnectToPlayer(ctx context.Context, params proxy.GetPlayerAddrParams) (net.IP, error) {
	gameManager, ok := p.GameManager, p.GameManager != nil
	if !ok || gameManager.Game == nil {
		return nil, fmt.Errorf("no game mananged for session: %d", p.Session.GetUserID())
	}

	peer, ok := gameManager.Game.Peers[params.UserID]
	if !ok {
		return nil, fmt.Errorf("could not find peer with user ID: %d", params.UserID)
	}

	if peer.Connected == nil {
		return nil, fmt.Errorf("peer does not have a connection channel")
	}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	select {
	case <-ctx.Done():
		slog.Error("timeout waiting for peer to connect", "user_id", params.UserID)
	case <-peer.Connected:
		slog.Debug("peer connected, user ID", "user_id", params.UserID)
	}

	return peer.Addr.IP, nil
}

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
