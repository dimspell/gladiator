package backend

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/dimspell/gladiator/internal/proxy/p2p"
	"github.com/dimspell/gladiator/internal/proxy/redirect"
	"github.com/dimspell/gladiator/internal/wire"
)

var _ Proxy = (*PeerToPeer)(nil)

type PeerToPeer struct {
	// A custom IP address to which we will connect to.
	hostIPAddress net.IP

	manager *PeerToPeerPeerManager
}

func NewPeerToPeer() *PeerToPeer {
	return &PeerToPeer{
		hostIPAddress: net.IPv4(127, 0, 1, 2),
		manager:       NewPeerToPeerManager(),
	}
}

func (p *PeerToPeer) CreateRoom(params CreateParams, session *Session) (net.IP, error) {
	ipAddr := net.IPv4(127, 0, 0, 1)

	// Create player from session data
	player := wire.Player{
		UserID:      session.UserID,
		Username:    session.Username,
		CharacterID: session.CharacterID,
		ClassType:   byte(session.ClassType),
		IPAddress:   ipAddr.To4().String(),
	}

	gameRoom := NewGameRoom()
	gameRoom.ID = params.GameID
	gameRoom.Name = params.GameID
	gameRoom.SetHost(player)
	gameRoom.SetPlayer(player)

	session.State.SetGameRoom(gameRoom)

	return ipAddr, nil
}

func (p *PeerToPeer) HostRoom(params HostParams, session *Session) error {
	host := &p2p.Peer{
		PeerUserID: session.GetUserID(),
		Addr:       &redirect.Addressing{IP: p.hostIPAddress},
		Mode:       redirect.CurrentUserIsHost,
	}

	room := session.State.GameRoom()
	if room == nil || room.ID != params.GameID {
		return fmt.Errorf("no game room found")
	}

	// FIXME: Use function instead
	p.manager.Peers[session] = &PeersToSessionMapping{
		Game: room,
		Peers: map[string]*p2p.Peer{
			host.PeerUserID: host,
		},
	}

	if err := p.sendRoomReadyNotification(session, params.GameID); err != nil {
		return fmt.Errorf("could not send set room ready: %w", err)
	}

	return nil
}

func (p *PeerToPeer) sendRoomReadyNotification(session *Session, gameID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	return session.SendSetRoomReady(ctx, gameID)
}

func (p *PeerToPeer) GetHostIP(hostIpAddress string, session *Session) net.IP {
	return p.hostIPAddress
}

func (p *PeerToPeer) GetPlayerAddr(params GetPlayerAddrParams, session *Session) (net.IP, error) {
	// Return the IP address of the player, if he is already in the list.
	// FIXME: Use function instead
	mapping, exist := p.manager.Peers[session]
	if exist {
		if peer, ok := mapping.Peers[params.UserID]; ok {
			return peer.Addr.IP, nil
		}
	}

	peer := session.IpRing.NextPeerAddress(
		params.UserID,
		params.UserID == session.GetUserID(),
		params.UserID == params.HostUserID,
	)

	if exist {
		mapping.Peers[peer.PeerUserID] = peer
	} else {
		// FIXME: Use function instead
		p.manager.Peers[session] = &PeersToSessionMapping{
			Game:  session.State.GameRoom(),
			Peers: map[string]*p2p.Peer{peer.PeerUserID: peer},
		}
	}

	return peer.Addr.IP, nil
}

func (p *PeerToPeer) Join(params JoinParams, session *Session) (net.IP, error) {
	peer := &p2p.Peer{
		PeerUserID: session.GetUserID(),
		Addr:       &redirect.Addressing{IP: net.IPv4(127, 0, 0, 1)},
		Mode:       redirect.None,
	}

	// FIXME: Use function instead
	mapping, exist := p.manager.Peers[session]
	if !exist {
		return nil, fmt.Errorf("could not find current session among the peers")
	}

	mapping.Peers[peer.PeerUserID] = peer

	return peer.Addr.IP, nil
}

func (p *PeerToPeer) Close(session *Session) {
	// FIXME: Use function instead
	if mapping, exists := p.manager.Peers[session]; exists {
		for _, peer := range mapping.Peers {
			if peer.Connection != nil {
				peer.Connection.Close()
			}
		}
	}
	session.IpRing.Reset()
	// FIXME: Use function instead
	delete(p.manager.Peers, session)
}

func (p *PeerToPeer) ExtendWire(session *Session) MessageHandler {
	return &PeerToPeerMessageHandler{
		session: session,
		proxy:   p.manager,
	}
}
