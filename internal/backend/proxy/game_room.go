package proxy

import (
	"sync"

	"github.com/dimspell/gladiator/internal/wire"
)

type GameRoom struct {
	sync.RWMutex

	ID   string
	Name string

	Host    wire.Player
	Players map[int64]wire.Player
}

func NewGameRoom(name string, host wire.Player) *GameRoom {
	return &GameRoom{
		Players: map[int64]wire.Player{
			host.UserID: host,
		},
		Host: host,
		ID:   name,
		Name: name,
	}
}

func (g *GameRoom) SetHost(player wire.Player) {
	g.Lock()
	g.Host = player
	g.Unlock()
}

func (g *GameRoom) GetPlayer(userId int64) (wire.Player, bool) {
	g.RLock()
	defer g.RUnlock()

	player, ok := g.Players[userId]
	if !ok {
		return wire.Player{}, false
	}
	return player, ok
}

func (g *GameRoom) SetPlayer(player wire.Player) {
	g.Lock()
	g.Players[player.UserID] = player
	g.Unlock()
}

func (g *GameRoom) DeletePlayer(userId int64) {
	g.Lock()
	delete(g.Players, userId)
	g.Unlock()
}

// func (p *SessionStore) Reset() {
// 	p.Lock()
// 	for id, peer := range p.peers {
// 		peer.Close()
// 		delete(p.peers, id)
// 	}
// 	p.Unlock()
// }
//
// func (p *SessionStore) Range(f func(string, *Peer)) {
// 	p.RLock()
// 	defer p.RUnlock()
// 	for id, peer := range p.peers {
// 		f(id, peer)
// 	}
// }
