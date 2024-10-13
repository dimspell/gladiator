package backend

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/dimspell/gladiator/internal/proxy/p2p"
	"github.com/dimspell/gladiator/internal/proxy/redirect"
)

var _ Proxy = (*PeerToPeer)(nil)

type PeerToPeer struct {
	SignalServerURL string

	hostIPAddress net.IP

	ipRing        *p2p.IpRing
	gatheredPeers []*p2p.Peer

	mtxClient sync.Mutex
	p2pClient *p2p.PeerToPeer

	stop context.CancelFunc
}

func NewPeerToPeer(signalServerURL string) *PeerToPeer {
	return &PeerToPeer{
		// A custom IP address to which we will connect to.
		hostIPAddress: net.IPv4(127, 0, 1, 2),
		ipRing:        p2p.NewIpRing(),

		SignalServerURL: signalServerURL,
	}
}

func (p *PeerToPeer) Create(params CreateParams) (net.IP, error) {
	return net.IPv4(127, 0, 0, 1), nil
}

func (p *PeerToPeer) Host(params HostParams) (err error) {
	p.mtxClient.Lock()

	ctx, cancel := context.WithCancel(context.Background())
	p.stop = cancel

	p.p2pClient, err = p2p.DialSignalServer(p.SignalServerURL, params.HostUserID, params.GameID, true)
	if err != nil {
		return fmt.Errorf("failed to connect to the signal server: %w", err)
	}

	host := &p2p.Peer{
		PeerUserID: params.HostUserID,
		Addr:       &redirect.Addressing{IP: p.hostIPAddress},
		Mode:       redirect.CurrentUserIsHost,
	}
	p.p2pClient.Peers.Set(host.PeerUserID, host)
	p.p2pClient.IpRing = p.ipRing

	go p.p2pClient.Run(ctx)

	return nil
}

func (p *PeerToPeer) GetHostIP(_ string) net.IP { return p.hostIPAddress }

func (p *PeerToPeer) GetPlayerAddr(params GetPlayerAddrParams) (net.IP, error) {
	// Return the IP address of the player, if he is already in the list.
	for _, peer := range p.gatheredPeers {
		if peer.PeerUserID == params.UserID {
			return peer.Addr.IP, nil
		}
	}

	peer := p.ipRing.NextPeerAddress(
		params.UserID,
		params.UserID == params.CurrentUserID,
		params.UserID == params.HostUserID,
	)
	p.gatheredPeers = append(p.gatheredPeers, peer)

	return peer.Addr.IP, nil
}

func (p *PeerToPeer) Join(params JoinParams) (err error) {
	p.mtxClient.Lock()

	ctx, cancel := context.WithCancel(context.Background())
	p.stop = cancel

	p.p2pClient, err = p2p.DialSignalServer(p.SignalServerURL, params.CurrentUserID, params.GameID, false)
	if err != nil {
		p.mtxClient.Unlock()
		return fmt.Errorf("failed to connect to the signal server: %w", err)
	}
	for _, peer := range p.gatheredPeers {
		p.p2pClient.Peers.Set(peer.PeerUserID, peer)
	}
	current := &p2p.Peer{
		PeerUserID: params.CurrentUserID,
		Addr:       &redirect.Addressing{IP: net.IPv4(127, 0, 0, 1)},
		Mode:       redirect.None,
	}
	p.p2pClient.Peers.Set(current.PeerUserID, current)
	p.p2pClient.IpRing = p.ipRing

	go p.p2pClient.Run(ctx)

	// TODO: Add a timer to check if the connection is established and running

	return err
}

func (p *PeerToPeer) Close() {
	defer p.mtxClient.Unlock()

	if p.stop != nil {
		p.stop()
		// TODO: Wait for the connection to be closed
	}
	if p.p2pClient != nil {
		p.p2pClient.Close()
		p.p2pClient = nil
	}

	// TODO: Clear the list of gathered players
	clear(p.gatheredPeers)
	p.ipRing = p2p.NewIpRing()
}
