package model

import (
	"encoding/binary"
)

type Ranking struct {
	Players       []RankingPosition
	CurrentPlayer RankingPosition
}

type RankingPosition struct {
	Rank          uint32
	Points        uint32
	Username      string
	CharacterName string
}

func (ranking *Ranking) ToBytes() []byte {
	var buf = []byte{}

	buf = binary.LittleEndian.AppendUint32(buf, uint32(len(ranking.Players)))

	for _, position := range ranking.Players {
		buf = binary.LittleEndian.AppendUint32(buf, uint32(position.Rank))
		buf = binary.LittleEndian.AppendUint32(buf, uint32(position.Points))
		buf = append(buf, position.Username...)
		buf = append(buf, 0)
		buf = append(buf, position.CharacterName...)
		buf = append(buf, 0)
	}

	buf = binary.LittleEndian.AppendUint32(buf, uint32(ranking.CurrentPlayer.Rank))
	buf = binary.LittleEndian.AppendUint32(buf, uint32(ranking.CurrentPlayer.Points))
	buf = append(buf, ranking.CurrentPlayer.Username...)
	buf = append(buf, 0)
	buf = append(buf, ranking.CurrentPlayer.CharacterName...)
	buf = append(buf, 0)

	return buf
}
