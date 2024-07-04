package console

import (
	"context"
	"database/sql"
	"time"

	"connectrpc.com/connect"
	"github.com/dimspell/gladiator/console/database"
	multiv1 "github.com/dimspell/gladiator/gen/multi/v1"
	"github.com/dimspell/gladiator/gen/multi/v1/multiv1connect"
)

var _ multiv1connect.GameServiceHandler = (*gameServiceServer)(nil)

type gameServiceServer struct {
	DB *database.SQLite
}

// ListGames returns a list of all open games.
func (s *gameServiceServer) ListGames(ctx context.Context, req *connect.Request[multiv1.ListGamesRequest]) (*connect.Response[multiv1.ListGamesResponse], error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	gameRooms, err := s.DB.Read.ListGameRooms(ctx)
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
			CreatedBy:     room.CreatedBy,
			HostUserId:    room.HostUserID,
		}
	}

	resp := connect.NewResponse(&multiv1.ListGamesResponse{Games: games})
	return resp, nil
}

// GetGame returns a game by name.
func (s *gameServiceServer) GetGame(ctx context.Context, req *connect.Request[multiv1.GetGameRequest]) (*connect.Response[multiv1.GetGameResponse], error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	room, err := s.DB.Read.GetGameRoom(ctx, req.Msg.GameName)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	resp := connect.NewResponse(&multiv1.GetGameResponse{Game: &multiv1.Game{
		GameId:        room.ID,
		Name:          room.Name,
		Password:      room.Password.String,
		HostIpAddress: room.HostIpAddress,
		MapId:         room.MapID,
		CreatedBy:     room.CreatedBy,
		HostUserId:    room.HostUserID,
	}})
	return resp, nil
}

// CreateGame creates a new game.
func (s *gameServiceServer) CreateGame(ctx context.Context, req *connect.Request[multiv1.CreateGameRequest]) (*connect.Response[multiv1.CreateGameResponse], error) {
	input := req.Msg

	game, err := s.DB.Write.CreateGameRoom(ctx, database.CreateGameRoomParams{
		Name:          input.GameName,
		Password:      sql.NullString{String: input.Password, Valid: len(input.Password) > 0},
		HostIpAddress: input.HostIpAddress,
		MapID:         input.MapId,
		CreatedBy:     req.Msg.UserId,
		HostUserID:    req.Msg.UserId,
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
			CreatedBy:     game.CreatedBy,
			HostUserId:    game.HostUserID,
		},
	})
	return resp, nil
}

// JoinGame tries to get the player to join a game.
func (s *gameServiceServer) JoinGame(ctx context.Context, req *connect.Request[multiv1.JoinGameRequest]) (*connect.Response[multiv1.JoinGameResponse], error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	exist, _ := s.DB.Write.ExistPlayerInRoom(ctx, database.ExistPlayerInRoomParams{
		GameRoomID:  req.Msg.GameRoomId,
		CharacterID: req.Msg.CharacterId,
	})
	if exist == 1 {
		return connect.NewResponse(&multiv1.JoinGameResponse{}), nil
	}

	err := s.DB.Write.AddPlayerToRoom(ctx, database.AddPlayerToRoomParams{
		GameRoomID:  req.Msg.GameRoomId,
		UserID:      req.Msg.UserId,
		CharacterID: req.Msg.CharacterId,
		IpAddress:   req.Msg.IpAddress,
		AddedAt:     time.Now().Unix(),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	resp := connect.NewResponse(&multiv1.JoinGameResponse{})
	return resp, nil
}

// ListPlayers returns a list of all players in a game.
func (s *gameServiceServer) ListPlayers(ctx context.Context, req *connect.Request[multiv1.ListPlayersRequest]) (*connect.Response[multiv1.ListPlayersResponse], error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	roomPlayers, err := s.DB.Read.GetGameRoomPlayers(ctx, req.Msg.GameRoomId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	players := make([]*multiv1.Player, len(roomPlayers))
	for i, player := range roomPlayers {
		players[i] = &multiv1.Player{
			UserId:        player.UserID,
			Username:      player.Username,
			CharacterName: player.CharacterName,
			ClassType:     player.ClassType,
			IpAddress:     player.IpAddress,
		}
	}

	resp := connect.NewResponse(&multiv1.ListPlayersResponse{Players: players})
	return resp, nil
}
