package backend

import (
	"bytes"
	"fmt"

	"github.com/dispel-re/dispel-multi/model"
)

func (b *Backend) HandleUpdateCharacterSpells(session *model.Session, req UpdateCharacterSpellsRequest) error {
	_, _, _, err := req.Parse()
	if err != nil {
		return err
	}

	return b.Send(session.Conn, UpdateCharacterSpells, []byte{1, 0, 0, 0})
}

type UpdateCharacterSpellsRequest []byte

func (r UpdateCharacterSpellsRequest) Parse() (user string, character string, spells []byte, err error) {
	split := bytes.SplitN(r, []byte{0}, 3)
	if len(split) != 3 {
		return user, character, spells, fmt.Errorf("packet-73: no enough arguments, malformed request payload: %v", r)
	}
	if len(split[2]) != 43 {
		return user, character, spells, fmt.Errorf("packet-73: the spells array has invalid length: %v", r)
	}
	user = string(split[0])
	character = string(split[1])
	spells = split[2]
	return user, character, spells, nil
}
