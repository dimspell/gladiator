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

	manager *PeerToPeerPeerManager
}

func NewPeerToPeer() *PeerToPeer {
	return &PeerToPeer{
		hostIPAddress: net.IPv4(127, 0, 1, 2),
		manager:       NewPeerToPeerManager(),
	}
}

func (p *PeerToPeer) CreateRoom(params CreateParams, session *bsession.Session) (net.IP, error) {
	p.Close(session)

	ipAddr := net.IPv4(127, 0, 0, 1)
	player := session.ToPlayer(ipAddr)

	gameRoom := bsession.NewGameRoom(params.GameID, player)
	session.State.SetGameRoom(gameRoom)

	return ipAddr, nil
}

func (p *PeerToPeer) HostRoom(params HostParams, session *bsession.Session) error {
	host := &Peer{
		PeerUserID: session.GetUserID(),
		Addr:       &redirect.Addressing{IP: p.hostIPAddress},
		Mode:       redirect.CurrentUserIsHost,
	}

	room, err := session.State.GameRoom()
	if err != nil {
		return fmt.Errorf("could not get game room: %w", err)
	}
	if room.ID != params.GameID {
		return fmt.Errorf("no game room found")
	}

	// FIXME: Use function instead
	p.manager.Peers[session] = &PeersToSessionMapping{
		Game: room,
		Peers: map[string]*Peer{
			host.PeerUserID: host,
		},
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

func (p *PeerToPeer) GetPlayerAddr(params GetPlayerAddrParams, session *bsession.Session) (net.IP, error) {
	// Return the IP address of the player, if he is already in the list.
	// FIXME: Use function instead
	mapping, exist := p.manager.Peers[session]
	if exist {
		if peer, ok := mapping.Peers[params.UserID]; ok {
			return peer.Addr.IP, nil
		}
	}

	// peer := session.IpRing.NextPeerAddress(
	// 	params.UserID,
	// 	params.UserID == session.GetUserID(),
	// 	params.UserID == params.HostUserID,
	// )

	// if exist {
	// 	mapping.Peers[peer.PeerUserID] = peer
	// } else {
	// 	// FIXME: Use function instead
	// 	game, err := session.State.GameRoom()
	// 	if err != nil {
	// 		panic(err)
	// 	}
	//
	// 	p.manager.Peers[session] = &PeersToSessionMapping{
	// 		Game:  game,
	// 		Peers: map[string]*Peer{peer.PeerUserID: peer},
	// 	}
	// }
	//
	// return peer.Addr.IP, nil
	panic("could not find peer for user")
	return nil, nil
}

func (p *PeerToPeer) Join(params JoinParams, session *bsession.Session) (net.IP, error) {
	peer := &Peer{
		PeerUserID: session.GetUserID(),
		Addr:       &redirect.Addressing{IP: net.IPv4(127, 0, 0, 1)},
		Mode:       redirect.None,
	}

	// FIXME: Use function instead
	mapping, exist := p.manager.Peers[session]
	if !exist {
		return nil, fmt.Errorf("could not find current session among the peers for user ID: %s", session.GetUserID())
	}

	mapping.Peers[peer.PeerUserID] = peer

	return peer.Addr.IP, nil
}

func (p *PeerToPeer) Close(session *bsession.Session) {
	// FIXME: Use function instead
	if mapping, exists := p.manager.Peers[session]; exists {
		for _, peer := range mapping.Peers {
			if peer.Connection != nil {
				peer.Connection.Close()
			}
		}
	}

	// FIXME: Use function instead
	delete(p.manager.Peers, session)
}

func (p *PeerToPeer) ExtendWire(session *bsession.Session) MessageHandler {
	return &PeerToPeerMessageHandler{
		session: session,
		proxy:   p.manager,
	}
}
