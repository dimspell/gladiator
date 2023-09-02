package backend

import "github.com/dispel-re/dispel-multi/model"

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

	return nil
}

type GetCharacterSpellsRequest []byte
