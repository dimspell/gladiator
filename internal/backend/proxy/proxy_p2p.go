package proxy

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/dimspell/gladiator/internal/backend/bsession"
	"github.com/dimspell/gladiator/internal/backend/redirect"
)

var _ Proxy = (*PeerToPeer)(nil)

type PeerToPeer struct {
	// A custom IP address to which we will connect to.
	hostIPAddress net.IP

	Manager *PeerToPeerPeerManager
}

func NewPeerToPeer() *PeerToPeer {
	return &PeerToPeer{
		hostIPAddress: net.IPv4(127, 0, 1, 2),
		Manager:       NewPeerToPeerManager(),
	}
}

func (p *PeerToPeer) CreateRoom(params CreateParams, session *bsession.Session) (net.IP, error) {
	p.Close(session)

	ipAddr := net.IPv4(127, 0, 0, 1)
	hostPlayer := session.ToPlayer(ipAddr)

	gameRoom := NewGameRoom(params.GameID, hostPlayer)

	p.Manager.Peers[session] = &PeersToSessionMapping{
		Game:   gameRoom,
		IpRing: NewIpRing(),
		Peers: map[string]*Peer{
			hostPlayer.ID(): {
				PeerUserID: session.GetUserID(),
				Addr:       &redirect.Addressing{IP: p.hostIPAddress},
				Mode:       redirect.CurrentUserIsHost,
			},
		},
	}

	return ipAddr, nil
}

func (p *PeerToPeer) HostRoom(params HostParams, session *bsession.Session) error {
	peers, ok := p.Manager.Peers[session]
	if !ok {
		return fmt.Errorf("no game mananged for session: %s", session.GetUserID())
	}
	if peers.Game == nil || peers.Game.ID != params.GameID {
		return fmt.Errorf("no game room found")
	}

	if err := p.sendRoomReadyNotification(session, params.GameID); err != nil {
		return fmt.Errorf("could not send set room ready: %w", err)
	}

	return nil
}

func (p *PeerToPeer) sendRoomReadyNotification(session *bsession.Session, gameID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	return session.SendSetRoomReady(ctx, gameID)
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
		peer := ipRing.NextPeerAddress(
			player.ID(),
			session.GetUserID() == player.ID(),
			gameRoom.Host.ID() == player.ID())
		peers[player.ID()] = peer
	}

	p.Manager.Peers[session] = &PeersToSessionMapping{
		Game:   gameRoom,
		IpRing: ipRing,
		Peers:  peers,
	}

	return nil
}

func (p *PeerToPeer) GetPlayerAddr(params GetPlayerAddrParams, session *bsession.Session) (net.IP, error) {
	// Return the IP address of the player, if he is already in the list.
	// FIXME: Use function instead
	mapping, ok := p.Manager.Peers[session]
	if !ok {
		return nil, fmt.Errorf("no game manager for session: %s", session.GetUserID())
	}

	for _, peer := range mapping.Peers {
		if peer.PeerUserID == params.UserID {
			return peer.Addr.IP, nil
		}
	}

	return nil, fmt.Errorf("could not find player with user ID: %s", params.UserID)
}

func (p *PeerToPeer) Join(params JoinParams, session *bsession.Session) (net.IP, error) {
	peer := &Peer{
		PeerUserID: session.GetUserID(),
		Addr:       &redirect.Addressing{IP: net.IPv4(127, 0, 0, 1)},
		Mode:       redirect.None,
	}

	// FIXME: Use function instead
	mapping, exist := p.Manager.Peers[session]
	if !exist {
		return nil, fmt.Errorf("could not find current session among the peers for user ID: %s", session.GetUserID())
	}

	mapping.Peers[peer.PeerUserID] = peer

	gameRoom := mapping.Game
	gameRoom.SetPlayer(session.ToPlayer(peer.Addr.IP.To4()))

	return peer.Addr.IP, nil
}

func (p *PeerToPeer) Close(session *bsession.Session) {
	// FIXME: Use function instead
	if mapping, exists := p.Manager.Peers[session]; exists {
		for _, peer := range mapping.Peers {
			if peer.Connection != nil {
				peer.Connection.Close()
			}
		}
	}

	// FIXME: Use function instead
	delete(p.Manager.Peers, session)
}

func (p *PeerToPeer) ExtendWire(session *bsession.Session) MessageHandler {
	return &PeerToPeerMessageHandler{
		session: session,
		proxy:   p.Manager,
	}
}
