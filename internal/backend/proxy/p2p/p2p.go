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
	"github.com/pion/webrtc/v4"
)

var _ proxy.Proxy = (*PeerToPeer)(nil)

// PeerToPeer implements the Proxy interface for WebRTC-based peer-to-peer connections.
// It manages game rooms, peer connections, and network addressing for multiplayer games
type PeerToPeer struct {
	// A custom IP address to which we will connect to.
	hostIPAddress net.IP

	WebRTCConfig   webrtc.Configuration
	NewTCPRedirect redirect.NewRedirect
	NewUDPRedirect redirect.NewRedirect
	SessionStore   *SessionStore
}

func NewPeerToPeer(iceServers ...webrtc.ICEServer) *PeerToPeer {
	config := webrtc.Configuration{}
	for _, server := range iceServers {
		config.ICEServers = append(config.ICEServers, server)
	}

	return &PeerToPeer{
		hostIPAddress:  net.IPv4(127, 0, 1, 2),
		WebRTCConfig:   config,
		NewTCPRedirect: redirect.NewTCPRedirect,
		NewUDPRedirect: redirect.NewUDPRedirect,
		SessionStore:   &SessionStore{sessions: make(map[*bsession.Session]*GameManager)},
	}
}

// CreateRoom creates a new game room and assigns the session as the host.
// Returns the assigned IP address for the host player
func (p *PeerToPeer) CreateRoom(params proxy.CreateParams, session *bsession.Session) (net.IP, error) {
	//p.Close(session)

	mapping, ok := p.SessionStore.GetSession(session)
	if !ok {
		return nil, fmt.Errorf("no game mananged for session: %d", session.GetUserID())
	}

	mapping.Reset()

	ipAddr := net.IPv4(127, 0, 0, 1)
	hostPlayer := session.ToPlayer(ipAddr)

	gameRoom := &Game{
		ID:     params.GameID,
		Host:   hostPlayer,
		Peers:  map[int64]*Peer{}, // FIXME: Add size limit
		IpRing: NewIpRing(),
	}

	mapping.Game = gameRoom

	return ipAddr, nil
}

func (p *PeerToPeer) HostRoom(ctx context.Context, params proxy.HostParams, session *bsession.Session) error {
	mapping, ok := p.SessionStore.sessions[session]
	if !ok {
		return fmt.Errorf("no game mananged for session: %d", session.GetUserID())
	}
	if mapping.Game == nil || mapping.Game.ID != params.GameID {
		return fmt.Errorf("no game room found")
	}

	if err := session.SendSetRoomReady(ctx, params.GameID); err != nil {
		return fmt.Errorf("could not send set room ready: %w", err)
	}

	return nil
}

func (p *PeerToPeer) GetHostIP(hostIpAddress net.IP, session *bsession.Session) net.IP {
	return p.hostIPAddress
}

func (p *PeerToPeer) SelectGame(params proxy.GameData, session *bsession.Session) error {
	//p.Close(session)

	mapping, ok := p.SessionStore.GetSession(session)
	if !ok {
		return fmt.Errorf("no game mananged for session: %d", session.GetUserID())
	}

	mapping.Reset()

	hostPlayer, err := params.FindHostUser()
	if err != nil {
		return err
	}

	gameRoom := Game{
		ID:     params.Game.GameId,
		Host:   hostPlayer,
		Peers:  map[int64]*Peer{}, // FIXME: Add size limit
		IpRing: NewIpRing(),
	}

	for _, player := range params.ToWirePlayers() {
		peerConnection, err := webrtc.NewPeerConnection(p.WebRTCConfig)
		if err != nil {
			return err
		}

		isCurrentUser := session.GetUserID() == player.UserID
		isHostUser := gameRoom.Host.UserID == player.UserID

		peer, err := NewPeer(peerConnection,
			gameRoom.IpRing,
			player.UserID,
			isCurrentUser,
			isHostUser)
		if err != nil {
			return err
		}
		gameRoom.Peers[player.UserID] = peer

		// if !isCurrentUser {
		// if err := peer.setupPeerConnection(context.TODO(), session, player, false); err != nil {
		// 	return err
		// }
	}

	return nil
}

func (p *PeerToPeer) GetPlayerAddr(params proxy.GetPlayerAddrParams, session *bsession.Session) (net.IP, error) {
	mapping, ok := p.SessionStore.GetSession(session)
	if !ok {
		return nil, fmt.Errorf("no game mananged for session: %d", session.GetUserID())
	}
	peer, ok := mapping.GetPeer(params.UserID)
	if !ok {
		return nil, fmt.Errorf("could not find peer with user ID: %d", params.UserID)
	}

	return peer.Addr.IP, nil
}

func (p *PeerToPeer) Join(ctx context.Context, params proxy.JoinParams, session *bsession.Session) (net.IP, error) {
	ip := net.IPv4(127, 0, 0, 1)

	mapping, ok := p.SessionStore.GetSession(session)
	if !ok || mapping.Game == nil {
		return nil, fmt.Errorf("no game mananged for session: %d", session.GetUserID())
	}

	peer := &Peer{
		UserID: session.GetUserID(),
		Addr:   &redirect.Addressing{IP: ip},
		Mode:   redirect.None,
	}
	mapping.AddPeer(peer)

	for _, pr := range mapping.Game.Peers {
		ch := make(chan struct{}, 1)
		pr.Connected = ch
	}

	return ip, nil
}

func (p *PeerToPeer) ConnectToPlayer(ctx context.Context, params proxy.GetPlayerAddrParams, session *bsession.Session) (net.IP, error) {
	mapping, ok := p.SessionStore.GetSession(session)
	if !ok || mapping.Game == nil {
		return nil, fmt.Errorf("no game mananged for session: %d", session.GetUserID())
	}

	peer, ok := mapping.Game.Peers[params.UserID]
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
		slog.Error("timeout waiting for peer to connect", "userID", params.UserID)
		// return nil, fmt.Errorf("could not get peer addr: %w for user ID: %s", ctx.Err(), params.UserID)
	// case <-webrtc.GatheringCompletePromise(peer.Connection):
	// 	slog.Debug("gathering complete")
	case <-peer.Connected:
		slog.Debug("peer connected, user ID", "userID", params.UserID)
	}

	return peer.Addr.IP, nil
}

func (p *PeerToPeer) Close(session *bsession.Session) {
	// if mapping, exists := p.SessionStore.GetSession(session); exists {
	// 	for _, peer := range mapping.Peers {
	// 		peer.Terminate()
	// 	}
	// }
	//
	// p.SessionStore.DeleteSession(session)
}

func (p *PeerToPeer) NewWebSocketHandler(session *bsession.Session) proxy.MessageHandler {
	gameManager := p.SessionStore.Add(session, p.WebRTCConfig)

	return &PeerToPeerMessageHandler{
		session,
		gameManager,
		p.NewTCPRedirect,
		p.NewUDPRedirect,
	}
}
