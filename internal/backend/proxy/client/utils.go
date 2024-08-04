package client

import (
	"container/ring"
	"log/slog"
	"net"
	"sync"

	"github.com/fxamacker/cbor/v2"
)

func decodeCBOR[T any](data []byte) (v T, err error) {
	err = cbor.Unmarshal(data, &v)
	if err != nil {
		slog.Warn("Error decoding JSON", "error", err, "payload", string(data))
	}
	return
}

type IpRing struct {
	*ring.Ring
}

func NewIpRing() IpRing {
	r := ring.New(100)
	n := r.Len()
	for i := 0; i < n; i++ {
		r.Value = i + 2
		r = r.Next()
	}
	return IpRing{r}
}

func (r *IpRing) IP() net.IP {
	d := byte(r.Value.(int))
	defer r.Next()
	return net.IPv4(127, 0, 1, d)
}

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
