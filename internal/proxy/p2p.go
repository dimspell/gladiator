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
		hostIPAddress:   net.IPv4(127, 0, 0, 1),
		SignalServerURL: signalServerURL,
	}
}

func (p *PeerToPeer) GetHostIP(_ string) net.IP { return p.hostIPAddress }

func (p *PeerToPeer) Create(params CreateParams) (net.IP, error) {
	// p.mtxClient.Lock()

	if p.Client != nil {
		return nil, fmt.Errorf("already connected to the signal server")
	}

	client, err := p2p.DialSignalServer(p.SignalServerURL, params.HostUserID, params.GameID, true)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to the signal server: %w", err)
	}
	p.Client = client

	return net.IPv4(127, 0, 0, 1), nil
}

func (p *PeerToPeer) Host(params HostParams) error {
	// ctx, cancel := context.WithCancel(context.Background())
	// p.stop = cancel
	// go p.Client.Run(ctx, params.HostUserID)
	// return nil
	return nil
}

func (p *PeerToPeer) Join(params JoinParams) error {
	if p.Client != nil {
		return nil
	}

	// p.mtxClient.Lock()
	// TODO: It is called twice, leading to the deadlock and memory usage
	client, err := p2p.DialSignalServer(p.SignalServerURL, params.CurrentUserID, params.GameID, false)
	if err != nil {
		return fmt.Errorf("failed to connect to the signal server: %w", err)
	}
	p.Client = client

	// close(p.done)
	// p.done = make(chan struct{}, 1)

	go p.Client.Run(ctx params.HostUserIP)

	// select {
	// case <-time.After(5 * time.Second):
	// 	return nil, fmt.Errorf("timeout")
	// 	// case ip := <-p.chanMyIP:
	// 	// 	return ip, nil
	// }

	return nil
}

func (p *PeerToPeer) GetPlayerAddr(params GetPlayerAddrParams) (net.IP, error) {
	peer, ok := p.Client.Peers.Get(params.UserID)
	if !ok {
		return nil, fmt.Errorf("user %s not found", params.UserID)
	}
	return peer.IP, nil
}

func (p *PeerToPeer) Close() {

	if p.stop != nil {
		p.stop()
	}

	if p.Client == nil {
		return
	}
	p.Client.Close()
	p.Client = nil
}
