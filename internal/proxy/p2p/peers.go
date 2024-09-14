package p2p

import (
	"sync"
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
	defer p.RUnlock()

	member, ok := p.peers[id]
	if !ok {
		return nil, false
	}
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
		peer.Close()
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
