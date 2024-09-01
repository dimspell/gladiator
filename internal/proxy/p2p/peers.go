package p2p

import (
	"sync"

	"github.com/dimspell/gladiator/internal/proxy/redirect"
	"github.com/pion/webrtc/v4"
)

type Peers struct {
	sync.RWMutex

	peers map[string]*Peer
}

func NewPeers() *Peers {
	return &Peers{
		peers: make(map[string]*Peer),
	}
}

func (p *Peers) Get(id string) (*Peer, bool) {
	p.RLock()
	member, ok := p.peers[id]
	if !ok {
		return nil, false
	}
	p.RUnlock()
	return member, ok
}

func (p *Peers) Set(id string, member *Peer) {
	p.Lock()
	p.peers[id] = member
	p.Unlock()
}

func (p *Peers) Delete(id string) {
	p.Lock()
	delete(p.peers, id)
	p.Unlock()
}

func (p *Peers) Reset() {
	p.Lock()
	for id, peer := range p.peers {
		peer.Connection.Close()
		delete(p.peers, id)
	}
	p.Unlock()
}

func (p *Peers) Range(f func(string, *Peer)) {
	p.RLock()
	defer p.RUnlock()
	for id, peer := range p.peers {
		f(id, peer)
	}
}

type Peer struct {
	PeerUserID string
	Addr       *redirect.Addressing
	Mode       redirect.Mode

	Connection *webrtc.PeerConnection
}
