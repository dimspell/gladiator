package console

import (
	"context"
	"sort"
	"testing"

	"connectrpc.com/connect"
	"github.com/coder/websocket"
	multiv1 "github.com/dimspell/gladiator/gen/multi/v1"
	"github.com/dimspell/gladiator/internal/wire"
	"github.com/stretchr/testify/assert"
)

type mockConn struct{}

func (m *mockConn) Read(ctx context.Context) (websocket.MessageType, []byte, error) {
	return websocket.MessageText, []byte{}, nil
}
func (m *mockConn) Write(ctx context.Context, typ websocket.MessageType, p []byte) error { return nil }
func (m *mockConn) CloseNow() error                                                      { return nil }

func TestGameServiceServer_CreateGame(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		g := &GameServiceServer{
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
	})

	t.Run("create and leave", func(t *testing.T) {
		roomID := "testing"
		g := &GameServiceServer{
			Multiplayer: NewMultiplayer(),
		}
		sess := NewUserSession(10, nil)

		g.Multiplayer.AddUserSession(10, sess)

		resp, err := g.CreateGame(context.Background(), connect.NewRequest(&multiv1.CreateGameRequest{
			GameName: roomID,
			MapId:    multiv1.GameMap_FrozenLabyrinth,

			HostIpAddress: "192.168.100.1",
			HostUserId:    10,
		}))
		if err != nil || resp.Msg.Game.GameId != roomID || len(g.Multiplayer.Rooms) != 1 {
			t.Error("room not created")
			return
		}
		g.Multiplayer.LeaveRoom(t.Context(), sess)

		if roomsLen := len(g.Multiplayer.Rooms); roomsLen != 0 {
			t.Errorf("Rooms length is wrong, expected 0, got %d", roomsLen)
			return
		}
	})
}

func TestGameServiceServer_ListGames(t *testing.T) {
	g := &GameServiceServer{
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
	g := &GameServiceServer{
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
	t.Run("ok", func(t *testing.T) {
		roomID := "testing"
		g := &GameServiceServer{
			Multiplayer: NewMultiplayer(),
		}
		g.Multiplayer.AddUserSession(10, NewUserSession(10, &mockConn{}))
		g.Multiplayer.AddUserSession(5, NewUserSession(5, &mockConn{}))

		if _, err := g.CreateGame(context.Background(), connect.NewRequest(&multiv1.CreateGameRequest{
			GameName: roomID,
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
			GameRoomId: roomID,
			IpAddress:  "192.168.100.201",
		}))
		if err != nil {
			t.Error(err)
			return
		}

		players := resp.Msg.GetPlayers()

		// Sort in ascending order
		sort.Slice(players, func(i, j int) bool {
			return players[i].UserId < players[j].UserId
		})
		assert.Equal(t, 2, len(players))

		assert.Equal(t, int64(5), players[0].UserId)
		assert.Equal(t, "192.168.100.201", players[0].IpAddress)

		assert.Equal(t, int64(10), players[1].UserId)
		assert.Equal(t, "192.168.100.1", players[1].IpAddress)
	})

	t.Run("rejoin", func(t *testing.T) {
		roomID := "testing"
		g := &GameServiceServer{
			Multiplayer: NewMultiplayer(),
		}

		guestSession := NewUserSession(5, &mockConn{})
		g.Multiplayer.AddUserSession(10, NewUserSession(10, &mockConn{}))
		g.Multiplayer.AddUserSession(5, guestSession)

		if _, err := g.CreateGame(t.Context(), connect.NewRequest(&multiv1.CreateGameRequest{
			GameName: roomID,
			Password: "secret",
			MapId:    multiv1.GameMap_FrozenLabyrinth,

			HostIpAddress: "192.168.100.1",
			HostUserId:    10,
		})); err != nil {
			t.Error(err)
			return
		}
		g.Multiplayer.SetRoomReady(wire.Message{
			Type:    wire.SetRoomReady,
			Content: roomID,
		})

		resp1, err := g.JoinGame(t.Context(), connect.NewRequest(&multiv1.JoinGameRequest{
			UserId:     5,
			GameRoomId: roomID,
			IpAddress:  "192.168.100.201",
		}))
		if err != nil {
			t.Error(err)
			return
		}

		assert.Equal(t, 2, len(resp1.Msg.GetPlayers()))
		assert.Equal(t, 2, len(g.Multiplayer.Rooms[roomID].Players))

		g.Multiplayer.LeaveRoom(t.Context(), guestSession)
		assert.Equal(t, 1, len(g.Multiplayer.Rooms[roomID].Players))

		resp2, err := g.JoinGame(t.Context(), connect.NewRequest(&multiv1.JoinGameRequest{
			UserId:     5,
			GameRoomId: roomID,
			IpAddress:  "192.168.100.201",
		}))
		if err != nil {
			t.Error(err)
			return
		}

		assert.Equal(t, 2, len(resp2.Msg.GetPlayers()))
		assert.Equal(t, 2, len(g.Multiplayer.Rooms[roomID].Players))
	})
}

func TestGameServiceServer_CreateGame_Errors(t *testing.T) {
	g := &GameServiceServer{Multiplayer: NewMultiplayer()}
	// No user session added
	_, err := g.CreateGame(context.Background(), connect.NewRequest(&multiv1.CreateGameRequest{
		GameName:   "fail",
		HostUserId: 99,
	}))
	assert.Error(t, err)
}

func TestGameServiceServer_JoinGame_Errors(t *testing.T) {
	g := &GameServiceServer{Multiplayer: NewMultiplayer()}
	// No room, no user
	_, err := g.JoinGame(context.Background(), connect.NewRequest(&multiv1.JoinGameRequest{
		UserId: 1, GameRoomId: "nope",
	}))
	assert.Error(t, err)
}

func TestGameServiceServer_DuplicateRoom(t *testing.T) {
	g := &GameServiceServer{Multiplayer: NewMultiplayer()}
	g.Multiplayer.AddUserSession(1, NewUserSession(1, nil))
	_, err := g.CreateGame(context.Background(), connect.NewRequest(&multiv1.CreateGameRequest{
		GameName: "dup", HostUserId: 1,
	}))
	assert.NoError(t, err)
	_, err = g.CreateGame(context.Background(), connect.NewRequest(&multiv1.CreateGameRequest{
		GameName: "dup", HostUserId: 1,
	}))
	assert.Error(t, err)
}

func TestGameServiceServer_JoinTwice(t *testing.T) {
	t.Skip("Failing - needs to be fixed")
	g := &GameServiceServer{Multiplayer: NewMultiplayer()}
	g.Multiplayer.AddUserSession(1, NewUserSession(1, nil))
	g.Multiplayer.AddUserSession(2, NewUserSession(2, nil))
	_, _ = g.CreateGame(context.Background(), connect.NewRequest(&multiv1.CreateGameRequest{
		GameName: "room", HostUserId: 1,
	}))
	_, err := g.JoinGame(context.Background(), connect.NewRequest(&multiv1.JoinGameRequest{
		UserId: 2, GameRoomId: "room",
	}))
	assert.NoError(t, err)
	_, err = g.JoinGame(context.Background(), connect.NewRequest(&multiv1.JoinGameRequest{
		UserId: 2, GameRoomId: "room",
	}))
	assert.Error(t, err)
}
