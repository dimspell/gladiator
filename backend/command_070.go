package backend

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"

	"github.com/dispel-re/dispel-multi/internal/database"
	"github.com/dispel-re/dispel-multi/model"
)

// HandleShowRanking handles 0x46ff (255-70) command
func (b *Backend) HandleShowRanking(session *model.Session, req RankingRequest) error {
	if session.UserID == 0 {
		return fmt.Errorf("packet-70: user is not logged in")
	}

	data, err := req.Parse()
	if err != nil {
		return err
	}

	positions, err := b.DB.SelectRanking(context.TODO(), database.SelectRankingParams{
		ClassType: int64(data.ClassType),
		Offset:    int64(data.Offset),
	})
	if err != nil {
		return err
	}
	currentPlayer, err := b.DB.GetCurrentUser(context.TODO(), database.GetCurrentUserParams{
		Username:      data.Username,
		CharacterName: data.CharacterName,
	})
	if err != nil {
		return err
	}

	rankingPositions := make([]model.RankingPosition, len(positions))
	for i, position := range positions {
		rankingPositions[i] = model.RankingPosition{
			Rank:          uint32(position.Position.(int64)),
			Points:        uint32(position.ScorePoints),
			Username:      position.Username,
			CharacterName: position.CharacterName,
		}
	}

	ranking := model.Ranking{
		Players: rankingPositions,
		CurrentPlayer: model.RankingPosition{
			Rank:          uint32(currentPlayer.Position.(int64)),
			Points:        uint32(currentPlayer.ScorePoints),
			Username:      currentPlayer.Username,
			CharacterName: currentPlayer.CharacterName,
		},
	}

	return b.Send(session.Conn, ShowRanking, ranking.ToBytes())
}

type RankingRequest []byte

type RankingRequestData struct {
	ClassType     model.ClassType
	Offset        uint32
	Username      string
	CharacterName string
}

func NewRankingRequest(data RankingRequestData) RankingRequest {
	userNameLength := len(data.Username)
	characterNameLength := len(data.CharacterName)

	buf := make([]byte, 4+4+userNameLength+1+characterNameLength+1)

	// 0:4 Class type
	buf[0] = byte(data.ClassType)

	// 4:8 Offset used in pagination
	binary.LittleEndian.PutUint32(buf[4:8], data.Offset)

	// Username (null-terminated string)
	copy(buf[8:], data.Username)
	buf[8+userNameLength] = 0

	// Character name (null-terminated string)
	copy(buf[8+userNameLength+1:], data.CharacterName)

	return buf
}

func (r RankingRequest) Parse() (data RankingRequestData, err error) {
	data.ClassType = model.ClassType(r[0])
	data.Offset = binary.LittleEndian.Uint32(r[4:8])

	split := bytes.Split(r[8:], []byte{0})
	data.Username = string(split[0])
	data.CharacterName = string(split[1])

	return data, nil
}
