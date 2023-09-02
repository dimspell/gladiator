package backend

import "github.com/dispel-re/dispel-multi/model"

func (b *Backend) HandleUpdateCharacterStats(session *model.Session, req UpdateCharacterStatsRequest) error {
	// _ = CharacterFromBytes(buf)
	return b.Send(session.Conn, UpdateCharacterStats, []byte{})
}

type UpdateCharacterStatsRequest []byte
