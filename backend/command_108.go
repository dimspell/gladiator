package backend

import (
	"bytes"

	"github.com/dispel-re/dispel-multi/model"
)

func (b *Backend) HandleUpdateCharacterStats(session *model.Session, req UpdateCharacterStatsRequest) error {
	// _ = CharacterFromBytes(buf)
	return b.Send(session.Conn, UpdateCharacterStats, []byte{})
}

type UpdateCharacterStatsRequest []byte

func (r UpdateCharacterStatsRequest) Parse() (info model.CharacterInfo, user string, character string, unknown []byte, err error) {
	info = model.NewCharacterInfo(r[:56])

	split := bytes.SplitN(r[56:], []byte{0}, 3)
	user = string(split[0])
	character = string(split[1])
	unknown = split[2]

	return info, user, character, unknown, err
}
