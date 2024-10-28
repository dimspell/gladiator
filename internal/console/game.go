package console

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	multiv1 "github.com/dimspell/gladiator/gen/multi/v1"
	"github.com/dimspell/gladiator/gen/multi/v1/multiv1connect"
	"github.com/dimspell/gladiator/internal/wire"
)

var _ multiv1connect.GameServiceHandler = (*gameServiceServer)(nil)

type gameServiceServer struct {
	Multiplayer *Multiplayer
}

// ListGames returns a list of all open games.
func (s *gameServiceServer) ListGames(_ context.Context, req *connect.Request[multiv1.ListGamesRequest]) (*connect.Response[multiv1.ListGamesResponse], error) {
	s.Multiplayer.roomsMutex.RLock()
	defer s.Multiplayer.roomsMutex.RUnlock()

	games := make([]*multiv1.Game, 0, len(s.Multiplayer.Rooms))
	for _, room := range s.Multiplayer.Rooms {
		games = append(games, &multiv1.Game{
			GameId:        room.ID,
			Name:          room.Name,
			Password:      room.Password,
			MapId:         room.MapID,
			HostUserId:    room.HostPlayer.UserID,
			HostIpAddress: room.HostPlayer.IPAddress,
		})
	}

	resp := connect.NewResponse(&multiv1.ListGamesResponse{Games: games})
	return resp, nil
}

// GetGame finds the game room by name.
func (s *gameServiceServer) GetGame(_ context.Context, req *connect.Request[multiv1.GetGameRequest]) (*connect.Response[multiv1.GetGameResponse], error) {
	s.Multiplayer.roomsMutex.RLock()
	defer s.Multiplayer.roomsMutex.RUnlock()

	room, found := s.Multiplayer.Rooms[req.Msg.GetGameRoomId()]
	if !found {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("game %s not found", req.Msg.GetGameRoomId()))
	}

	players := make([]*multiv1.Player, 0, len(room.Players))
	for _, player := range room.Players {
		players = append(players, &multiv1.Player{
			UserId:      player.UserID,
			Username:    player.Username,
			CharacterId: player.CharacterID,
			ClassType:   multiv1.ClassType(player.ClassType),
			IpAddress:   player.IPAddress,
		})
	}
	resp := connect.NewResponse(&multiv1.GetGameResponse{
		Game: &multiv1.Game{
			GameId:        room.ID,
			Name:          room.Name,
			Password:      room.Password,
			MapId:         room.MapID,
			HostUserId:    room.HostPlayer.UserID,
			HostIpAddress: room.HostPlayer.IPAddress,
		},
		Players: players,
	})
	return resp, nil
}

// CreateGame creates a new game.
func (s *gameServiceServer) CreateGame(_ context.Context, req *connect.Request[multiv1.CreateGameRequest]) (*connect.Response[multiv1.CreateGameResponse], error) {
	s.Multiplayer.roomsMutex.Lock()
	defer s.Multiplayer.roomsMutex.Unlock()

	gameId := req.Msg.GetGameName()
	player := wire.Player{
		UserID:      req.Msg.GetHost().UserId,
		Username:    req.Msg.GetHost().Username,
		CharacterID: req.Msg.GetHost().CharacterId,
		ClassType:   byte(req.Msg.GetHost().ClassType),
		IPAddress:   req.Msg.GetHost().IpAddress,
	}
	room := wire.LobbyRoom{
		Ready:    false,
		ID:       gameId,
		Name:     req.Msg.GetGameName(),
		Password: req.Msg.GetPassword(),
		MapID:    req.Msg.MapId,

		// TODO
		HostPlayer: player,
		CreatedBy:  player,
		Players:    []wire.Player{player},
	}

	s.Multiplayer.Rooms[gameId] = &room

	resp := connect.NewResponse(&multiv1.CreateGameResponse{
		Game: &multiv1.Game{
			GameId:        room.ID,
			Name:          room.Name,
			Password:      room.Password,
			MapId:         room.MapID,
			HostUserId:    room.HostPlayer.UserID,
			HostIpAddress: room.HostPlayer.IPAddress,
		},
	})
	return resp, nil
}

// JoinGame tries to get the player to join a game.
func (s *gameServiceServer) JoinGame(_ context.Context, req *connect.Request[multiv1.JoinGameRequest]) (*connect.Response[multiv1.JoinGameResponse], error) {
	s.Multiplayer.roomsMutex.Lock()
	defer s.Multiplayer.roomsMutex.Unlock()

	gameId := req.Msg.GetGameRoomId()
	room, ok := s.Multiplayer.Rooms[gameId]
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("game %s not found", gameId))
	}

	for _, player := range room.Players {
		if player.UserID == req.Msg.GetUserId() {
			// return nil, connect.NewError(connect.CodeAlreadyExists, fmt.Errorf("player %s already exists", player.UserID))
			return connect.NewResponse(&multiv1.JoinGameResponse{}), nil
		}
	}

	player := wire.Player{
		UserID:      req.Msg.UserId,
		Username:    req.Msg.CharacterName,
		CharacterID: req.Msg.CharacterId,
		ClassType:   byte(req.Msg.ClassType),
		IPAddress:   req.Msg.IpAddress,
	}
	room.Players = append(room.Players, player)
	s.Multiplayer.Rooms[gameId] = room

	players := make([]*multiv1.Player, 0, len(room.Players))
	for _, player := range room.Players {
		players = append(players, &multiv1.Player{
			UserId:      player.UserID,
			Username:    player.Username,
			CharacterId: player.CharacterID,
			ClassType:   multiv1.ClassType(player.ClassType),
			IpAddress:   player.IPAddress,
		})
	}
	resp := connect.NewResponse(&multiv1.JoinGameResponse{Players: players})
	return resp, nil
}
