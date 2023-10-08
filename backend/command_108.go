package backend

import (
	"bytes"
	"encoding/base64"
	"fmt"

	"github.com/dispel-re/dispel-multi/model"
)

// HandleUpdateCharacterStats handles 0x6cff (255-108) command.
//
// It can be received by the game server in multiple scenarios:
//   - .
func (b *Backend) HandleUpdateCharacterStats(session *model.Session, req UpdateCharacterStatsRequest) error {
	// _ = CharacterFromBytes(buf)
	return b.Send(session.Conn, UpdateCharacterStats, []byte{})
}

type UpdateCharacterStatsRequest []byte

type UpdateCharacterStatsRequestData struct {
	CharacterInfo model.CharacterInfo
	User          string
	Character     string
	Unknown       []byte
}

func (r UpdateCharacterStatsRequest) Parse() (data UpdateCharacterStatsRequestData, err error) {
	if len(r) < 56 {
		return data, fmt.Errorf("packet-108: packet is too short: %s", base64.StdEncoding.EncodeToString(r))
	}
	split := bytes.SplitN(r[56:], []byte{0}, 3)
	if len(split) != 3 {
		return data, fmt.Errorf("packet-108: no enough arguments, malformed request payload: %s", base64.StdEncoding.EncodeToString(r))
	}

	data.CharacterInfo = model.NewCharacterInfo(r[:56])
	data.User = string(split[0])
	data.Character = string(split[1])
	data.Unknown = split[2]

	return data, err
}
