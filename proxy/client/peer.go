package client

import (
	"log/slog"
	"sync"

	"github.com/pion/webrtc/v4"
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
func (p *Peers) Get(id string) (*Peer, bool) {
	p.RLock()
	member, ok := p.Members[id]
	if !ok {
		slog.Error("not exist")
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

type Peer struct {
	ID   string
	Name string

	Connection *webrtc.PeerConnection
	ChannelTCP *webrtc.DataChannel
	ChannelUDP *webrtc.DataChannel
}
