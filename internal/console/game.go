package console

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	multiv1 "github.com/dimspell/gladiator/gen/multi/v1"
	"github.com/dimspell/gladiator/gen/multi/v1/multiv1connect"
)

var _ multiv1connect.GameServiceHandler = (*gameServiceServer)(nil)

type gameServiceServer struct {
	Multiplayer *Multiplayer
}

// ListGames returns a list of all open games.
func (s *gameServiceServer) ListGames(_ context.Context, req *connect.Request[multiv1.ListGamesRequest]) (*connect.Response[multiv1.ListGamesResponse], error) {
	rooms := s.Multiplayer.ListRooms()

	games := make([]*multiv1.Game, 0, len(rooms))
	for _, room := range rooms {
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
	room, found := s.Multiplayer.GetRoom(req.Msg.GetGameRoomId())
	if !found {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("game %s not found", req.Msg.GetGameRoomId()))
	}

	players := make([]*multiv1.Player, 0, len(room.Players))
	for _, player := range room.Players {
		players = append(players, &multiv1.Player{
			UserId:      player.UserID,
			Username:    player.User.Username,
			CharacterId: player.Character.CharacterID,
			ClassType:   multiv1.ClassType(player.Character.ClassType),
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
	gameId := req.Msg.GetGameName()

	hostSession, found := s.Multiplayer.GetUserSession(req.Msg.HostUserId)
	if !found {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("user session %d not found", req.Msg.HostUserId))
	}

	hostSession.IPAddress = req.Msg.HostIpAddress

	room := s.Multiplayer.CreateRoom(GameRoom{
		Ready:    false,
		ID:       gameId,
		Name:     req.Msg.GetGameName(),
		Password: req.Msg.GetPassword(),
		MapID:    req.Msg.MapId,

		// TODO
		HostPlayer: hostSession,
		CreatedBy:  hostSession,
		Players:    map[int64]*UserSession{hostSession.UserID: hostSession},
	})

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
	room, err := s.Multiplayer.JoinRoom(
		req.Msg.GameRoomId,
		req.Msg.UserId,
		req.Msg.IpAddress,
	)
	if err != nil {
		return nil, connect.NewError(connect.CodeCanceled, err)
	}

	// for _, session := range s.Multiplayer.sessions {
	// 	wire.Compose(wire.JoinRoom, wire.Message{
	// 		Type:    wire.JoinRoom,
	// 		From:    newPlayer.ID(),
	// 		To:      "0",
	// 		Content: newPlayer,
	// 	})
	// }

	players := make([]*multiv1.Player, 0, len(room.Players))
	for _, player := range room.Players {
		players = append(players, &multiv1.Player{
			UserId:      player.UserID,
			Username:    player.User.Username,
			CharacterId: player.Character.CharacterID,
			ClassType:   multiv1.ClassType(player.Character.ClassType),
			IpAddress:   player.IPAddress,
		})
	}
	resp := connect.NewResponse(&multiv1.JoinGameResponse{Players: players})

	return resp, nil
}
