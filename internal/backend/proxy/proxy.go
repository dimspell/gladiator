package proxy

import (
	"context"
	"fmt"
	"net"

	multiv1 "github.com/dimspell/gladiator/gen/multi/v1"
	"github.com/dimspell/gladiator/internal/backend/bsession"
	"github.com/dimspell/gladiator/internal/wire"
)

// Proxy is an interface that defines methods for managing game rooms and player
// connections. It provides functionality for creating and hosting game rooms,
// joining game sessions, and retrieving player IP addresses.
type Proxy interface {
	HostProxy
	SelectProxy
	JoinProxy

	Close(session *bsession.Session)

	NewWebSocketHandler(session *bsession.Session) MessageHandler
}

type HostProxy interface {
	// GetHostIP is used when game attempts to list the IP address of the game
	// room. This function can be used to override the IP address.
	GetHostIP(net.IP, *bsession.Session) net.IP

	// CreateRoom creates a new game room with the provided parameters and returns
	// the IP address of the game host.
	CreateRoom(CreateParams, *bsession.Session) (net.IP, error)

	// HostRoom creates a new game room with the provided parameters and returns
	// an error if the operation fails.
	HostRoom(context.Context, HostParams, *bsession.Session) error
}

type CreateParams struct {
	GameID string
}

type HostParams struct {
	GameID string
}

type SelectProxy interface {
	SelectGame(GameData, *bsession.Session) error
	GetPlayerAddr(GetPlayerAddrParams, *bsession.Session) (net.IP, error)
}

type JoinProxy interface {
	Join(context.Context, JoinParams, *bsession.Session) (net.IP, error)
	ConnectToPlayer(context.Context, GetPlayerAddrParams, *bsession.Session) (net.IP, error)
}

type GameData struct {
	Game    *multiv1.Game
	Players []*multiv1.Player
}

func (d *GameData) ToWirePlayers() []wire.Player {
	players := make([]wire.Player, len(d.Players))
	for i, player := range d.Players {
		players[i] = toWirePlayer(player)
	}
	return players
}

func (d *GameData) FindHostUser() (wire.Player, error) {
	player, err := findPlayer(d.Players, d.Game.HostUserId)
	if err != nil {
		return player, fmt.Errorf("host user not found")
	}
	return player, nil
}

type JoinParams struct {
	HostUserID int64
	GameID     string
	HostUserIP string
}

type GetPlayerAddrParams struct {
	GameID     string
	UserID     int64
	IPAddress  string
	HostUserID string
}

type MessageHandler interface {
	Handle(ctx context.Context, payload []byte) error
}

func toWirePlayer(player *multiv1.Player) wire.Player {
	return wire.Player{
		UserID:      player.UserId,
		Username:    player.Username,
		CharacterID: player.CharacterId,
		ClassType:   byte(player.ClassType),
		IPAddress:   player.IpAddress,
	}
}

func findPlayer(players []*multiv1.Player, needleUserId int64) (wire.Player, error) {
	for _, player := range players {
		if needleUserId == player.UserId {
			return toWirePlayer(player), nil
		}
	}
	return wire.Player{}, fmt.Errorf("user not found")
}
