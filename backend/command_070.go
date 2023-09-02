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

func NewRankingRequest(classType model.ClassType, resultsOffset uint32, userName string, characterName string) RankingRequest {
	userNameLength := len(userName)
	characterNameLength := len(characterName)

	buf := make([]byte, 4+4+userNameLength+1+characterNameLength+1)

	// 0:4 Class type
	buf[0] = byte(classType)

	// 4:8 Offset used in pagination
	binary.LittleEndian.PutUint32(buf[4:8], resultsOffset)

	// Username (null-terminated string)
	copy(buf[8:], userName)
	buf[8+userNameLength] = 0

	// Character name (null-terminated string)
	copy(buf[8+userNameLength+1:], characterName)

	return buf
}

func (r RankingRequest) ClassType() model.ClassType {
	return model.ClassType(r[0])
}

func (r RankingRequest) Offset() uint32 {
	return binary.LittleEndian.Uint32(r[4:8])
}

func (r RankingRequest) UserAndCharacterName() (user string, character string) {
	split := bytes.Split(r[8:], []byte{0})
	user = string(split[0])
	character = string(split[1])
	return user, character
}
