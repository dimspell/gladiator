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

func (p *Peers) Exist(id string) bool {
	p.RLock()
	_, ok := p.peers[id]
	p.RUnlock()
	return ok
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
	p.peers = make(map[string]*Peer)
	p.Unlock()
}

func (p *Peers) Range(f func(string, *Peer)) {
	p.RLock()
	defer p.RUnlock()
	for id, peer := range p.peers {
		f(id, peer)
	}
}
