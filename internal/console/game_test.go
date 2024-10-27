package console

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	multiv1 "github.com/dimspell/gladiator/gen/multi/v1"
	"github.com/dimspell/gladiator/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestGameServiceServer_(t *testing.T) {
	//
}

func TestGameServiceServer_CreateGame(t *testing.T) {
	g := &gameServiceServer{
		Multiplayer: NewMultiplayer(),
	}

	gameId := "Game Room"

	resp, err := g.CreateGame(context.Background(), connect.NewRequest(&multiv1.CreateGameRequest{
		GameName: gameId,
		Password: "secret",
		MapId:    3,

		Host: &multiv1.Player{
			UserId:      10,
			Username:    "user",
			CharacterId: 12,
			ClassType:   int32(model.ClassTypeMage),
			IpAddress:   "192.168.100.1",
		},
	}))
	if err != nil {
		t.Error(err)
		return
	}
	if resp.Msg.Game.GameId != gameId {
		t.Errorf("Name of the game room is wrong, expected %s, got %s", gameId, resp.Msg.Game.GameId)
		return
	}

	if len(g.Multiplayer.Rooms) != 1 {
		t.Errorf("Rooms length is wrong, expected 1, got %d", len(g.Multiplayer.Rooms))
		return
	}
	room, ok := g.Multiplayer.Rooms[gameId]
	if !ok {
		t.Errorf("Game room not found, expected %s, got %s", gameId, resp.Msg.Game.GameId)
		return
	}

	assert.Equal(t, false, room.Ready)
	assert.Equal(t, gameId, room.ID)
	assert.Equal(t, gameId, room.Name)
	assert.Equal(t, "secret", room.Password)
	assert.Equal(t, int64(3), room.MapID)

	assert.Equal(t, int64(10), room.HostPlayer.UserID)
	assert.Equal(t, int64(10), room.CreatedBy.UserID)
	assert.Equal(t, int64(10), room.Players[0].UserID)
}

func TestGameServiceServer_ListGames(t *testing.T) {
	g := &gameServiceServer{
		Multiplayer: NewMultiplayer(),
	}

	gameId := "Game Room"

	_, err := g.CreateGame(context.Background(), connect.NewRequest(&multiv1.CreateGameRequest{
		GameName: gameId,
		Password: "secret",
		MapId:    3,

		Host: &multiv1.Player{
			UserId:      10,
			Username:    "user",
			CharacterId: 12,
			ClassType:   int32(model.ClassTypeMage),
			IpAddress:   "192.168.100.1",
		},
	}))
	if err != nil {
		t.Error(err)
		return
	}

	resp, err := g.ListGames(context.Background(), connect.NewRequest(&multiv1.ListGamesRequest{}))
	if err != nil {
		t.Error(err)
		return
	}

	games := resp.Msg.GetGames()
	if len(games) != 1 {
		t.Errorf("Game list length is wrong, expected 1, got %d", len(games))
		return
	}

	room := games[0]
	assert.Equal(t, gameId, room.GameId)
	assert.Equal(t, gameId, room.Name)
	assert.Equal(t, "secret", room.Password)
	assert.Equal(t, int64(3), room.MapId)

	assert.Equal(t, int64(10), room.HostUserId)
	assert.Equal(t, int64(10), room.CreatedBy)
	// assert.Equal(t, int64(10), room.Players[0].UserID)
}
