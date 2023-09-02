package backend

import (
	"encoding/binary"

	"github.com/dispel-re/dispel-multi/model"
)

func (b *Backend) HandleGetCharacters(session *model.Session, req GetCharactersRequest) error {
	if len(session.User.Characters) == 0 {
		return b.Send(session.Conn, GetCharacters, []byte{0, 0, 0, 0})
	}

	response := []byte{1, 0, 0, 0}
	response = binary.LittleEndian.AppendUint32(response, uint32(len(session.User.Characters)))
	for _, character := range session.User.Characters {
		response = append(response, character.CharacterName...)
		response = append(response, 0)
	}
	return b.Send(session.Conn, GetCharacters, response)
}

type GetCharactersRequest []byte

func (r GetCharactersRequest) Username() string {
	return string(r[:len(r)-1])
}
