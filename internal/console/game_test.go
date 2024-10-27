package console

import (
	"connectrpc.com/connect"
	"context"
	"fmt"
	multiv1 "github.com/dimspell/gladiator/gen/multi/v1"
	"github.com/stretchr/testify/assert"
	"testing"
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
		UserId:        10,
		GameName:      gameId,
		Password:      "secret",
		HostIpAddress: "192.168.100.1",
		MapId:         3,
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
	fmt.Printf("%#v", room)

	assert.Equal(t, false, room.Ready)
	assert.Equal(t, gameId, room.ID)
	assert.Equal(t, gameId, room.Name)
	assert.Equal(t, "secret", room.Password)
	assert.Equal(t, int64(3), room.MapID)

	assert.Equal(t, int64(10), room.HostPlayer.UserID)
	assert.Equal(t, int64(10), room.CreatedBy.UserID)
	assert.Equal(t, int64(10), room.Players[0].UserID)
}
