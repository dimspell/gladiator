package backend

import (
	"context"
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

func (g *GameRoom) ToWire() wire.LobbyRoom {
	g.RLock()
	defer g.RUnlock()

	players := make([]wire.Player, 0, len(g.Players))
	for _, player := range g.Players {
		players = append(players, player)
	}

	return wire.LobbyRoom{
		Ready:      g.Ready,
		Name:       g.Name,
		HostPlayer: g.Host,
		Players:    players,
	}
}

// func (us *Session) SendCreateRoom(ctx context.Context, gameRoom *GameRoom) error {
// 	if err := wire.Write(ctx, us.wsConn, wire.ComposeTyped(
// 		wire.CreateRoom,
// 		wire.MessageContent[wire.LobbyRoom]{
// 			From:    us.GetUserID(),
// 			Type:    wire.CreateRoom,
// 			Content: gameRoom.ToWire(),
// 		}),
// 	); err != nil {
// 		return err
// 	}
// 	return nil
// }

func (s *Session) SendSetRoomReady(ctx context.Context, gameRoomId string) error {
	if err := wire.Write(ctx, s.wsConn, wire.ComposeTyped(
		wire.SetRoomReady,
		wire.MessageContent[wire.LobbyRoom]{
			From:    s.GetUserID(),
			Type:    wire.SetRoomReady,
			Content: wire.LobbyRoom{Ready: true, Name: gameRoomId, ID: gameRoomId},
		}),
	); err != nil {
		return err
	}
	return nil
}

// func (us *Session) SendJoinRoom(ctx context.Context, gameRoom *GameRoom) error {
// 	if err := wire.Write(ctx, us.wsConn, wire.ComposeTyped(
// 		wire.JoinRoom,
// 		wire.MessageContent[wire.LobbyRoom]{
// 			From:    us.GetUserID(),
// 			Type:    wire.JoinRoom,
// 			Content: gameRoom.ToWire(),
// 		}),
// 	); err != nil {
// 		return err
// 	}
// 	return nil
// }

func (s *Session) SendLeaveRoom(ctx context.Context, gameRoom *GameRoom) error {
	if err := wire.Write(ctx, s.wsConn, wire.ComposeTyped(
		wire.LeaveRoom,
		wire.MessageContent[wire.LobbyRoom]{
			From:    s.GetUserID(),
			Type:    wire.LeaveRoom,
			Content: gameRoom.ToWire(),
		}),
	); err != nil {
		return err
	}
	return nil
}
