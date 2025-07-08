package console

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/coder/websocket"
	multiv1 "github.com/dimspell/gladiator/gen/multi/v1"
	"github.com/stretchr/testify/assert"
)

type mockConn struct{}

func (m *mockConn) Read(ctx context.Context) (websocket.MessageType, []byte, error) {
	return websocket.MessageText, []byte{}, nil
}
func (m *mockConn) Write(ctx context.Context, typ websocket.MessageType, p []byte) error { return nil }
func (m *mockConn) CloseNow() error                                                      { return nil }

func TestGameServiceServer_CreateGame(t *testing.T) {
	g := &gameServiceServer{
		Multiplayer: NewMultiplayer(),
	}
	g.Multiplayer.AddUserSession(10, NewUserSession(10, nil))

	gameId := "Game Room"

	resp, err := g.CreateGame(context.Background(), connect.NewRequest(&multiv1.CreateGameRequest{
		GameName: gameId,
		Password: "secret",
		MapId:    multiv1.GameMap_FrozenLabyrinth,

		HostIpAddress: "192.168.100.1",
		HostUserId:    10,
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
	assert.Equal(t, multiv1.GameMap_FrozenLabyrinth, room.MapID)

	assert.Equal(t, int64(10), room.HostPlayer.UserID)
	assert.Equal(t, int64(10), room.CreatedBy.UserID)
	assert.Equal(t, int64(10), room.Players[10].UserID)
}

func TestGameServiceServer_ListGames(t *testing.T) {
	g := &gameServiceServer{
		Multiplayer: NewMultiplayer(),
	}
	g.Multiplayer.AddUserSession(10, NewUserSession(10, nil))

	gameId := "Game Room"
	_, err := g.CreateGame(context.Background(), connect.NewRequest(&multiv1.CreateGameRequest{
		GameName: gameId,
		Password: "secret",
		MapId:    multiv1.GameMap_FrozenLabyrinth,

		HostIpAddress: "192.168.100.1",
		HostUserId:    10,
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
	assert.Equal(t, multiv1.GameMap_FrozenLabyrinth, room.MapId)
	assert.Equal(t, "192.168.100.1", room.HostIpAddress)

	assert.Equal(t, int64(10), room.HostUserId)
}

func TestGameServiceServer_GetGame(t *testing.T) {
	g := &gameServiceServer{
		Multiplayer: NewMultiplayer(),
	}
	g.Multiplayer.AddUserSession(10, NewUserSession(10, nil))

	gameId := "Game Room"
	_, err := g.CreateGame(context.Background(), connect.NewRequest(&multiv1.CreateGameRequest{
		GameName: gameId,
		Password: "secret",
		MapId:    multiv1.GameMap_FrozenLabyrinth,

		HostIpAddress: "192.168.100.1",
		HostUserId:    10,
	}))
	if err != nil {
		t.Error(err)
		return
	}
	resp, err := g.GetGame(context.Background(), connect.NewRequest(&multiv1.GetGameRequest{
		GameRoomId: gameId,
	}))
	if err != nil {
		t.Error(err)
		return
	}

	room := resp.Msg.GetGame()
	assert.Equal(t, gameId, room.GameId)
	assert.Equal(t, gameId, room.Name)
	assert.Equal(t, "secret", room.Password)
	assert.Equal(t, multiv1.GameMap_FrozenLabyrinth, room.MapId)
	assert.Equal(t, int64(10), room.HostUserId)
	assert.Equal(t, "192.168.100.1", room.HostIpAddress)

	players := resp.Msg.GetPlayers()
	assert.Equal(t, 1, len(players))
	assert.Equal(t, int64(10), players[0].UserId)
}

func TestGameServiceServer_JoinGame(t *testing.T) {
	t.Skip("Fix me please")

	g := &gameServiceServer{
		Multiplayer: NewMultiplayer(),
	}
	g.Multiplayer.AddUserSession(10, NewUserSession(10, &mockConn{}))
	g.Multiplayer.AddUserSession(5, NewUserSession(5, &mockConn{}))

	gameId := "Game Room"
	if _, err := g.CreateGame(context.Background(), connect.NewRequest(&multiv1.CreateGameRequest{
		GameName: gameId,
		Password: "secret",
		MapId:    multiv1.GameMap_FrozenLabyrinth,

		HostIpAddress: "192.168.100.1",
		HostUserId:    10,
	})); err != nil {
		t.Error(err)
		return
	}

	resp, err := g.JoinGame(context.Background(), connect.NewRequest(&multiv1.JoinGameRequest{
		UserId:     5,
		GameRoomId: gameId,
		IpAddress:  "192.168.100.201",
	}))
	if err != nil {
		t.Error(err)
		return
	}

	players := resp.Msg.GetPlayers()
	assert.Equal(t, 2, len(players))

	assert.Equal(t, int64(10), players[0].UserId)
	assert.Equal(t, "192.168.100.1", players[0].IpAddress)

	assert.Equal(t, int64(5), players[1].UserId)
	assert.Equal(t, "192.168.100.201", players[1].IpAddress)
}
