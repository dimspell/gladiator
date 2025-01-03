package proxy

import (
	"context"
	"fmt"
	"net"

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

	WebRTCConfig webrtc.Configuration
	NewRedirect  redirect.NewRedirect
	SessionStore *SessionStore
}

func NewPeerToPeer() *PeerToPeer {
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			// {
			// 	URLs: []string{"stun:stun.l.google.com:19302"},
			// },
			// {
			// 	URLs:       []string{"turn:127.0.0.1:3478"},
			// 	Username:   "username1",
			// 	Credential: "password1",
			// },
		},
	}

	return &PeerToPeer{
		hostIPAddress: net.IPv4(127, 0, 1, 2),
		WebRTCConfig:  config,
		NewRedirect:   redirect.New,
		SessionStore:  &SessionStore{sessions: make(map[*bsession.Session]*SessionMapping)},
	}
}

// CreateRoom creates a new game room and assigns the session as the host.
// Returns the assigned IP address for the host player
func (p *PeerToPeer) CreateRoom(params CreateParams, session *bsession.Session) (net.IP, error) {
	p.Close(session)

	ipAddr := net.IPv4(127, 0, 0, 1)
	hostPlayer := session.ToPlayer(ipAddr)

	gameRoom := NewGameRoom(params.GameID, hostPlayer)

	p.SessionStore.SetSession(session, &SessionMapping{
		Game:   gameRoom,
		IpRing: NewIpRing(),
		Peers: map[string]*Peer{
			hostPlayer.ID(): {
				UserID: session.GetUserID(),
				Addr:   &redirect.Addressing{IP: p.hostIPAddress},
				Mode:   redirect.CurrentUserIsHost,
			},
		},
	})

	return ipAddr, nil
}

func (p *PeerToPeer) HostRoom(params HostParams, session *bsession.Session) error {
	peers, ok := p.SessionStore.sessions[session]
	if !ok {
		return fmt.Errorf("no game mananged for session: %s", session.GetUserID())
	}
	if peers.Game == nil || peers.Game.ID != params.GameID {
		return fmt.Errorf("no game room found")
	}

	if err := session.SendSetRoomReady(context.TODO(), params.GameID); err != nil {
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
	for _, player := range params.ToWirePlayers() {
		gameRoom.SetPlayer(player)
	}

	ipRing := NewIpRing()

	peers := map[string]*Peer{}
	for _, player := range params.ToWirePlayers() {
		peer, err := NewPeer(ipRing,
			player.ID(),
			session.GetUserID() == player.ID(),
			gameRoom.Host.ID() == player.ID())
		if err != nil {
			return err
		}
		peers[player.ID()] = peer
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

func (p *PeerToPeer) Join(params JoinParams, session *bsession.Session) (net.IP, error) {
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

	mapping.Peers[peer.UserID] = peer

	gameRoom := mapping.Game
	gameRoom.SetPlayer(session.ToPlayer(peer.Addr.IP.To4()))

	return peer.Addr.IP, nil
}

func (p *PeerToPeer) Close(session *bsession.Session) {
	if mapping, exists := p.SessionStore.GetSession(session); exists {
		for _, peer := range mapping.Peers {
			peer.Terminate()
		}
	}

	p.SessionStore.DeleteSession(session)
}

func (ss *SessionStore) getOrCreatePeer(session *bsession.Session, player wire.Player) (*Peer, error) {
	mapping, ok := ss.sessions[session]
	if ok {
		peer, found := mapping.Peers[player.ID()]
		if found {
			return peer, nil
		}
	}

	gameRoom := mapping.Game
	gameRoom.SetPlayer(player)

	isHost := gameRoom.Host.UserID == player.UserID
	isCurrentUser := gameRoom.Host.UserID == session.UserID

	return NewPeer(mapping.IpRing, player.ID(), isCurrentUser, isHost)
}

func (p *PeerToPeer) ExtendWire(session *bsession.Session) MessageHandler {
	// setUpChannels sets up the peer connection channels.
	setUpChannels := func(session *bsession.Session, player wire.Player, sendRTCOffer bool, createChannels bool) (*Peer, error) {
		peerConnection, err := webrtc.NewPeerConnection(p.WebRTCConfig)
		if err != nil {
			return nil, err
		}

		ctx := context.TODO()

		peer, _ := p.SessionStore.getOrCreatePeer(session, player)
		peer.Connection = peerConnection

		if _, ok := p.SessionStore.GetPeer(session, player.ID()); !ok {
			p.SessionStore.AddPeer(session, peer)
		}

		if err := peer.setupPeerConnection(ctx, session, &player, sendRTCOffer); err != nil {
			return nil, err
		}

		if createChannels {
			if err := peer.createDataChannels(p.NewRedirect); err != nil {
				return nil, err
			}
		}

		return peer, nil
	}

	return &PeerToPeerMessageHandler{
		session:       session,
		setUpChannels: setUpChannels,
		peerManager:   p.SessionStore,
	}
}
