package proxy

import (
	"context"
	"fmt"

	multiv1 "github.com/dimspell/gladiator/gen/multi/v1"
	"github.com/dimspell/gladiator/internal/model"
	"github.com/dimspell/gladiator/internal/wire"
)

// ProxyClient is an interface that defines methods for managing game rooms and
// player connections. It provides functionality for creating and hosting game
// rooms, joining game sessions, and retrieving player IP addresses.
type ProxyClient interface {
	CreateRoom(context.Context, CreateParams) error
	SetRoomReady(context.Context, CreateParams) error

	ListGames(context.Context) ([]model.LobbyRoom, error)
	GetGame(ctx context.Context, roomID string) (*model.LobbyRoom, []model.LobbyPlayer, error)
	JoinGame(ctx context.Context, roomID string, password string) ([]model.LobbyPlayer, error)

	Close()
	Handle(ctx context.Context, payload []byte) error
}

type CreateParams struct {
	GameID   string
	MapId    multiv1.GameMap
	Password string
}

type MessageHandler func(ctx context.Context, payload []byte) error

func ToWirePlayers(players []*multiv1.Player) []wire.Player {
	playersArr := make([]wire.Player, len(players))
	for i, player := range players {
		playersArr[i] = toWirePlayer(player)
	}
	return playersArr
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

func FindPlayer(players []*multiv1.Player, needleUserId int64) (wire.Player, error) {
	for _, player := range players {
		if needleUserId == player.UserId {
			return toWirePlayer(player), nil
		}
	}
	return wire.Player{}, fmt.Errorf("user not found")
}
