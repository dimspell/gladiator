package backend

import (
	"bytes"
	"fmt"

	"github.com/dispel-re/dispel-multi/model"
)

func (b *Backend) HandleGetCharacterSpells(session *model.Session, req GetCharacterSpellsRequest) error {
	// spells := make([]byte, 41)
	// for i := 0; i < len(spells); i++ {
	// 	spells[i] = 2
	// }
	// resp := []byte{255, opJoinedUpdateSpells, 0, 0}
	// resp = append(resp, spells...)
	// resp = append(resp, 0, 0)
	// binary.LittleEndian.PutUint16(resp[2:4], uint16(len(resp)))
	//
	// _, _ = conn.Write(resp)
	// _, err := conn.Write(resp)

	// GetCharacterSpells

	return nil
}

type GetCharacterSpellsRequest []byte

func (r GetCharacterSpellsRequest) Parse() (username string, characterName string, err error) {
	if bytes.Count(r, []byte{0}) < 2 {
		return "", "", fmt.Errorf("packet-72: malformed packet, not enough null-terminators")
	}

	split := bytes.SplitN(r, []byte{0}, 3)
	username = string(split[0])
	characterName = string(split[1])
	return username, characterName, nil
}
