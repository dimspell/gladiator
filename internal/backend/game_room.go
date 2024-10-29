package backend

import (
	"context"
	"fmt"
	"sync"

	"github.com/dimspell/gladiator/internal/wire"
)

type GameRoom struct {
	sync.RWMutex

	ID    string
	Name  string
	Ready bool

	Host    wire.Player
	Players map[string]wire.Player
}

func NewGameRoom() *GameRoom {
	return &GameRoom{
		Players: make(map[string]wire.Player),
	}
}

func (g *GameRoom) SetHost(player wire.Player) {
	g.Lock()
	g.Host = player
	g.Unlock()
}

func (g *GameRoom) SetReady() {
	g.Lock()
	g.Ready = true
	g.Unlock()
}

func (g *GameRoom) GetPlayer(id string) (wire.Player, bool) {
	g.RLock()
	defer g.RUnlock()

	player, ok := g.Players[id]
	if !ok {
		return wire.Player{}, false
	}
	return player, ok
}

func (g *GameRoom) SetPlayer(player wire.Player) {
	g.Lock()
	g.Players[player.ID()] = player
	g.Unlock()
}

func (g *GameRoom) DeletePlayer(player wire.Player) {
	g.Lock()
	delete(g.Players, player.ID())
	g.Unlock()
}

// func (p *Peers) Reset() {
// 	p.Lock()
// 	for id, peer := range p.peers {
// 		peer.Close()
// 		delete(p.peers, id)
// 	}
// 	p.Unlock()
// }
//
// func (p *Peers) Range(f func(string, *Peer)) {
// 	p.RLock()
// 	defer p.RUnlock()
// 	for id, peer := range p.peers {
// 		f(id, peer)
// 	}
// }

func (s *Session) SendSetRoomReady(ctx context.Context, gameRoomId string) error {
	err := wire.Write(ctx, s.wsConn, wire.ComposeTyped(
		wire.SetRoomReady,
		wire.MessageContent[string]{
			From:    s.GetUserID(),
			Type:    wire.SetRoomReady,
			Content: gameRoomId,
		}),
	)
	if err != nil {
		return fmt.Errorf("failed to send SetRoomReady: %w", err)
	}
	return nil
}

func (s *Session) SendLeaveRoom(ctx context.Context, gameRoom *GameRoom) error {
	if err := wire.Write(ctx, s.wsConn, wire.ComposeTyped(
		wire.LeaveRoom,
		wire.MessageContent[string]{
			From:    s.GetUserID(),
			Type:    wire.LeaveRoom,
			Content: gameRoom.ID,
		}),
	); err != nil {
		return err
	}
	return nil
}
