package console

import (
	"context"
	"database/sql"

	"connectrpc.com/connect"
	multiv1 "github.com/dispel-re/dispel-multi/gen/multi/v1"
	"github.com/dispel-re/dispel-multi/gen/multi/v1/multiv1connect"
	"github.com/dispel-re/dispel-multi/internal/database"
)

type gameServiceServer struct {
	multiv1connect.UnimplementedGameServiceHandler

	DB *database.Queries
}

func (s *gameServiceServer) CreateGame(ctx context.Context, req *connect.Request[multiv1.CreateGameRequest]) (*connect.Response[multiv1.CreateGameResponse], error) {
	input := req.Msg

	game, err := s.DB.CreateGameRoom(context.TODO(), database.CreateGameRoomParams{
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
