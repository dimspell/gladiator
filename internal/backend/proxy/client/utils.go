package client

import (
	"sync"
)

type Peers struct {
	sync.RWMutex

	Members map[string]*Peer
}

func NewPeers() *Peers {
	return &Peers{
		Members: make(map[string]*Peer),
	}
}

func (p *Peers) Exist(id string) bool {
	p.RLock()
	_, ok := p.Members[id]
	p.RUnlock()
	return ok
}

func (p *Peers) Get(id string) (*Peer, bool) {
	p.RLock()
	member, ok := p.Members[id]
	if !ok {
		return nil, false
	}
	p.RUnlock()
	return member, ok
}

func (p *Peers) Set(id string, member *Peer) {
	p.Lock()
	p.Members[id] = member
	p.Unlock()
}

func (p *Peers) Delete(id string) {
	p.Lock()
	// peer, ok := p.Members[id]
	// if ok {
	// peer.ChannelUDP.Close()
	// peer.ChannelTCP.Close()
	// }
	delete(p.Members, id)
	p.Unlock()
}

func (p *Peers) Range(f func(string, *Peer)) {
	p.RLock()
	defer p.RUnlock()
	for id, member := range p.Members {
		f(id, member)
	}
}
