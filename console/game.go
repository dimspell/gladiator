package console

import (
	"context"
	"database/sql"

	"connectrpc.com/connect"
	"github.com/dispel-re/dispel-multi/console/database"
	multiv1 "github.com/dispel-re/dispel-multi/gen/multi/v1"
	"github.com/dispel-re/dispel-multi/gen/multi/v1/multiv1connect"
)

var _ multiv1connect.GameServiceHandler = (*gameServiceServer)(nil)

type gameServiceServer struct {
	DB *database.Queries
}

func (s *gameServiceServer) ListGames(ctx context.Context, req *connect.Request[multiv1.ListGamesRequest]) (*connect.Response[multiv1.ListGamesResponse], error) {
	gameRooms, err := s.DB.ListGameRooms(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	games := make([]*multiv1.Game, len(gameRooms))
	for i, room := range gameRooms {
		games[i] = &multiv1.Game{
			GameId:        room.ID,
			Name:          room.Name,
			Password:      room.Password.String,
			HostIpAddress: room.HostIpAddress,
			MapId:         room.MapID,
		}
	}

	resp := connect.NewResponse(&multiv1.ListGamesResponse{Games: games})
	return resp, nil
}

func (s gameServiceServer) GetGame(ctx context.Context, req *connect.Request[multiv1.GetGameRequest]) (*connect.Response[multiv1.GetGameResponse], error) {
	room, err := s.DB.GetGameRoom(ctx, req.Msg.GameName)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	resp := connect.NewResponse(&multiv1.GetGameResponse{Game: &multiv1.Game{
		GameId:        room.ID,
		Name:          room.Name,
		Password:      room.Password.String,
		HostIpAddress: room.HostIpAddress,
		MapId:         room.MapID,
	}})
	return resp, nil
}

func (s *gameServiceServer) CreateGame(ctx context.Context, req *connect.Request[multiv1.CreateGameRequest]) (*connect.Response[multiv1.CreateGameResponse], error) {
	input := req.Msg

	game, err := s.DB.CreateGameRoom(ctx, database.CreateGameRoomParams{
		Name:          input.GameName,
		Password:      sql.NullString{String: input.Password, Valid: len(input.Password) > 0},
		HostIpAddress: input.HostIpAddress,
		MapID:         input.MapId,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	resp := connect.NewResponse(&multiv1.CreateGameResponse{
		Game: &multiv1.Game{
			GameId:        game.ID,
			Name:          game.Name,
			Password:      game.Password.String,
			HostIpAddress: game.HostIpAddress,
			MapId:         game.MapID,
		},
	})
	return resp, nil
}

func (s gameServiceServer) JoinGame(ctx context.Context, req *connect.Request[multiv1.JoinGameRequest]) (*connect.Response[multiv1.JoinGameResponse], error) {
	err := s.DB.AddPlayerToRoom(ctx, database.AddPlayerToRoomParams{
		GameRoomID:  req.Msg.GameRoomId,
		CharacterID: req.Msg.CharacterId,
		IpAddress:   req.Msg.IpAddress,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	resp := connect.NewResponse(&multiv1.JoinGameResponse{})
	return resp, nil
}

func (s gameServiceServer) ListPlayers(ctx context.Context, req *connect.Request[multiv1.ListPlayersRequest]) (*connect.Response[multiv1.ListPlayersResponse], error) {
	roomPlayers, err := s.DB.GetGameRoomPlayers(ctx, req.Msg.GameRoomId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	players := make([]*multiv1.Player, len(roomPlayers))
	for i, player := range roomPlayers {
		players[i] = &multiv1.Player{
			CharacterName: player.CharacterName,
			ClassType:     player.ClassType,
			IpAddress:     player.IpAddress,
		}
	}

	resp := connect.NewResponse(&multiv1.ListPlayersResponse{Players: players})
	return resp, nil
}
