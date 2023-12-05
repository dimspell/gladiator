package console

import (
	"context"

	"connectrpc.com/connect"
	multiv1 "github.com/dispel-re/dispel-multi/gen/multi/v1"
	"github.com/dispel-re/dispel-multi/gen/multi/v1/multiv1connect"
	"github.com/dispel-re/dispel-multi/internal/database"
)

type rankingServiceServer struct {
	multiv1connect.UnimplementedRankingServiceHandler

	DB *database.Queries
}

func (s *userServiceServer) GetRanking(ctx context.Context, req *connect.Request[multiv1.GetRankingRequest]) (*connect.Response[multiv1.GetRankingResponse], error) {

	// positions, err := s.DB.SelectRanking(context.TODO(), database.SelectRankingParams{
	// 	ClassType: int64(data.ClassType),
	// 	Offset:    int64(data.Offset),
	// })
	// if err != nil {
	// 	return err
	// }
	// currentPlayer, err := s.DB.GetCurrentUser(context.TODO(), database.GetCurrentUserParams{
	// 	Username:      data.Username,
	// 	CharacterName: data.CharacterName,
	// })
	// if err != nil {
	// 	return err
	// }
	//
	// rankingPositions := make([]model.RankingPosition, len(positions))
	// for i, position := range positions {
	// 	rankingPositions[i] = model.RankingPosition{
	// 		Rank:          uint32(position.Position.(int64)),
	// 		Points:        uint32(position.ScorePoints),
	// 		Username:      position.Username,
	// 		CharacterName: position.CharacterName,
	// 	}
	// }
	//
	// ranking := model.Ranking{
	// 	Players: rankingPositions,
	// 	CurrentPlayer: model.RankingPosition{
	// 		Rank:          uint32(currentPlayer.Position.(int64)),
	// 		Points:        uint32(currentPlayer.ScorePoints),
	// 		Username:      currentPlayer.Username,
	// 		CharacterName: currentPlayer.CharacterName,
	// 	},
	// }
	return nil, nil
}
