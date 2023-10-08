package backend

import (
	"bytes"
	"encoding/binary"

	"github.com/dispel-re/dispel-multi/model"
)

// HandleShowRanking handles 0x46ff (255-70) command
func (b *Backend) HandleShowRanking(session *model.Session, req RankingRequest) error {
	ranking := b.DB.Ranking()
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
