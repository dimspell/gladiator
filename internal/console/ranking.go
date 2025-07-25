package console

import (
	"context"

	"connectrpc.com/connect"
	multiv1 "github.com/dimspell/gladiator/gen/multi/v1"
	"github.com/dimspell/gladiator/gen/multi/v1/multiv1connect"
	"github.com/dimspell/gladiator/internal/console/database"
)

var _ multiv1connect.RankingServiceHandler = (*rankingServiceServer)(nil)

type rankingServiceServer struct {
	DB *database.SQLite
}

// GetRanking returns the ranking of the players.
func (s *rankingServiceServer) GetRanking(ctx context.Context, req *connect.Request[multiv1.GetRankingRequest]) (*connect.Response[multiv1.GetRankingResponse], error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	positions, err := s.DB.Read.SelectRanking(ctx, database.SelectRankingParams{
		ClassType: int64(req.Msg.GetClassType()),
		Offset:    int64(req.Msg.GetOffset()),
	})
	if err != nil {
		return nil, err
	}
	currentPlayer, err := s.DB.Read.GetCurrentUser(ctx, database.GetCurrentUserParams{
		ID:            req.Msg.GetUserId(),
		CharacterName: req.Msg.GetCharacterName(),
	})
	if err != nil {
		return nil, err
	}

	rankingPositions := make([]*multiv1.RankingPosition, len(positions))
	for i, position := range positions {
		rankingPositions[i] = &multiv1.RankingPosition{
			Rank:          uint32(position.Position.(int64)),
			Points:        uint32(position.ScorePoints),
			Username:      position.Username,
			CharacterName: position.CharacterName,
		}
	}

	resp := connect.NewResponse(&multiv1.GetRankingResponse{
		Players: rankingPositions,
		CurrentPlayer: &multiv1.RankingPosition{
			Rank:          uint32(currentPlayer.Position.(int64)),
			Points:        uint32(currentPlayer.ScorePoints),
			Username:      currentPlayer.Username,
			CharacterName: currentPlayer.CharacterName,
		},
	})
	return resp, nil
}
