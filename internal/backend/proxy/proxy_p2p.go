package proxy

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/dimspell/gladiator/internal/backend/bsession"
	"github.com/dimspell/gladiator/internal/backend/redirect"
	"github.com/dimspell/gladiator/internal/wire"
	"github.com/pion/webrtc/v4"
)

var _ Proxy = (*PeerToPeer)(nil)

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
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			// {
			// 	URLs: []string{"stun:stun.l.google.com:19302"},
			// },
			// {
			// 	URLs:       []string{"turn:192.168.121.212:3478"},
			// 	Username:   "username1",
			// 	Credential: "password1",
			// },
		},
	}
	for _, server := range iceServers {
		config.ICEServers = append(config.ICEServers, server)
	}

	return &PeerToPeer{
		hostIPAddress:  net.IPv4(127, 0, 1, 2),
		WebRTCConfig:   config,
		NewTCPRedirect: redirect.NewTCPRedirect,
		NewUDPRedirect: redirect.NewUDPRedirect,
		SessionStore:   &SessionStore{sessions: make(map[*bsession.Session]*SessionMapping)},
	}
}

// CreateRoom creates a new game room and assigns the session as the host.
// Returns the assigned IP address for the host player
func (p *PeerToPeer) CreateRoom(params CreateParams, session *bsession.Session) (net.IP, error) {
	p.Close(session)

	ipAddr := net.IPv4(127, 0, 0, 1)
	hostPlayer := session.ToPlayer(ipAddr)

	gameRoom := NewGameRoom(params.GameID, hostPlayer)

	// peer, err := p.CreatePeer(session, hostPlayer)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to create peer: %w", err)
	// }

	p.SessionStore.SetSession(session, &SessionMapping{
		Game:   gameRoom,
		IpRing: NewIpRing(),
		Peers:  make(map[string]*Peer), // FIXME: Add size limit
	})

	return ipAddr, nil
}

func (p *PeerToPeer) HostRoom(ctx context.Context, params HostParams, session *bsession.Session) error {
	peers, ok := p.SessionStore.sessions[session]
	if !ok {
		return fmt.Errorf("no game mananged for session: %s", session.GetUserID())
	}
	if peers.Game == nil || peers.Game.ID != params.GameID {
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

func (p *PeerToPeer) SelectGame(params GameData, session *bsession.Session) error {
	p.Close(session)

	hostPlayer, err := params.FindHostUser()
	if err != nil {
		return err
	}
	gameRoom := NewGameRoom(params.Game.GameId, hostPlayer)

	ipRing := NewIpRing()

	peers := map[string]*Peer{}
	for _, player := range params.ToWirePlayers() {
		peerConnection, err := webrtc.NewPeerConnection(p.WebRTCConfig)
		if err != nil {
			return err
		}

		isCurrentUser := session.GetUserID() == player.ID()
		isHostUser := gameRoom.Host.ID() == player.ID()

		peer, err := NewPeer(peerConnection,
			ipRing,
			player.ID(),
			isCurrentUser,
			isHostUser)
		if err != nil {
			return err
		}
		peers[player.ID()] = peer

		// if !isCurrentUser {
		// if err := peer.setupPeerConnection(context.TODO(), session, player, false); err != nil {
		// 	return err
		// }

		gameRoom.SetPlayer(player)
	}

	p.SessionStore.SetSession(session, &SessionMapping{
		Game:   gameRoom,
		IpRing: ipRing,
		Peers:  peers,
	})

	return nil
}

func (p *PeerToPeer) GetPlayerAddr(params GetPlayerAddrParams, session *bsession.Session) (net.IP, error) {
	peer, ok := p.SessionStore.GetPeer(session, params.UserID)
	if !ok {
		return nil, fmt.Errorf("could not find peer with user ID: %s", params.UserID)
	}

	return peer.Addr.IP, nil
}

func (p *PeerToPeer) Join(ctx context.Context, params JoinParams, session *bsession.Session) (net.IP, error) {
	peer := &Peer{
		UserID: session.GetUserID(),
		Addr:   &redirect.Addressing{IP: net.IPv4(127, 0, 0, 1)},
		Mode:   redirect.None,
	}

	// FIXME: Use function instead
	mapping, exist := p.SessionStore.GetSession(session)
	if !exist {
		return nil, fmt.Errorf("could not find current session among the peers for user ID: %s", session.GetUserID())
	}

	mapping.Peers[session.GetUserID()] = peer

	for _, pr := range mapping.Peers {
		ch := make(chan struct{}, 1)
		pr.Connected = ch
	}

	gameRoom := mapping.Game
	gameRoom.SetPlayer(session.ToPlayer(peer.Addr.IP.To4()))

	return peer.Addr.IP, nil
}

func (p *PeerToPeer) ConnectToPlayer(ctx context.Context, params GetPlayerAddrParams, session *bsession.Session) (net.IP, error) {
	peer, ok := p.SessionStore.GetPeer(session, params.UserID)
	if !ok {
		return nil, fmt.Errorf("could not find peer with user ID: %s", params.UserID)
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

func (p *PeerToPeer) NewWebSocketHandler(session *bsession.Session) MessageHandler {
	return &PeerToPeerMessageHandler{
		session,
		p.SessionStore,
		p.CreatePeer,
		p.NewTCPRedirect,
		p.NewUDPRedirect,
	}
}

func (p *PeerToPeer) CreatePeer(session *bsession.Session, player wire.Player) (*Peer, error) {
	// createPeer sets up the peer connection channels.
	mapping, ok := p.SessionStore.GetSession(session)
	if !ok {
		return nil, fmt.Errorf("could not find mapping for user ID: %s", session.GetUserID())
	}
	if peer, found := mapping.Peers[player.ID()]; found {
		slog.Debug("Reusing peer", "userId", player.ID())
		return peer, nil
	}

	peerConnection, err := webrtc.NewPeerConnection(p.WebRTCConfig)
	if err != nil {
		return nil, err
	}

	gameRoom := mapping.Game
	gameRoom.SetPlayer(player)

	isHost := gameRoom.Host.UserID == player.UserID
	isCurrentUser := gameRoom.Host.UserID == session.UserID

	peer, err := NewPeer(peerConnection, mapping.IpRing, player.ID(), isCurrentUser, isHost)
	if err != nil {
		return nil, err
	}

	ch := make(chan struct{}, 1)
	peer.Connected = ch

	return peer, nil
}
