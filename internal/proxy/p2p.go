package proxy

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/dimspell/gladiator/internal/proxy/p2p"
)

var _ Proxy = (*PeerToPeer)(nil)

type PeerToPeer struct {
	SignalServerURL string
	Client          *p2p.PeerToPeer
	mtxClient       sync.Mutex
	hostIPAddress   net.IP

	stop context.CancelFunc
}

func NewPeerToPeer(signalServerURL string) *PeerToPeer {
	return &PeerToPeer{
		// A custom IP address to which we will connect to.
		hostIPAddress: net.IPv4(127, 0, 1, 2),

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

	p.Client, err = p2p.DialSignalServer(p.SignalServerURL, params.HostUserID, params.GameID, true)
	if err != nil {
		return fmt.Errorf("failed to connect to the signal server: %w", err)
	}

	go p.Client.Run(ctx, params.HostUserID)

	return nil
}

func (p *PeerToPeer) GetHostIP(_ string) net.IP { return p.hostIPAddress }

func (p *PeerToPeer) GetPlayerAddr(params GetPlayerAddrParams) (net.IP, error) {
	peer, ok := p.Client.Peers.Get(params.UserID)
	if !ok {
		return nil, fmt.Errorf("user %s not found", params.UserID)
	}
	return peer.IP, nil
}

func (p *PeerToPeer) Join(params JoinParams) (err error) {
	p.mtxClient.Lock()

	ctx, cancel := context.WithCancel(context.Background())
	p.stop = cancel

	// TODO: It is called twice, leading to the deadlock and memory usage
	p.Client, err = p2p.DialSignalServer(p.SignalServerURL, params.CurrentUserID, params.GameID, false)
	if err != nil {
		p.mtxClient.Unlock()
		return fmt.Errorf("failed to connect to the signal server: %w", err)
	}

	go p.Client.Run(ctx, params.HostUserIP)

	return err
}

func (p *PeerToPeer) Close() {
	defer p.mtxClient.Unlock()

	if p.stop != nil {
		p.stop()
	}
	if p.Client != nil {
		p.Client.Close()
		p.Client = nil
	}
}
