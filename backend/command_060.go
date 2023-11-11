package backend

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/dispel-re/dispel-multi/model"
)

func (b *Backend) HandleGetCharacters(session *model.Session, req GetCharactersRequest) error {
	if session.User == nil {
		return fmt.Errorf("packet-44: user is not logged in")
	}

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

func (r GetCharactersRequest) Parse() (username string, err error) {
	if bytes.Count(r, []byte{0}) != 1 {
		return username, fmt.Errorf("packet-44: malformed payload: %v", r)
	}

	split := bytes.SplitN(r, []byte{0}, 2)
	username = string(split[0])

	return username, err
}
