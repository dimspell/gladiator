package backend

import (
	"bytes"
	"fmt"

	"github.com/dispel-re/dispel-multi/model"
)

func (b *Backend) HandleSelectCharacter(session *model.Session, req SelectCharacterRequest) error {
	data, err := req.Parse()
	if err != nil {
		return err
	}

	for _, c := range session.User.Characters {
		if c.CharacterName == data.CharacterName {
			session.Character = &c
			break
		}
	}

	// No characters owned by player
	if session.Character == nil || session.Character.CharacterName != data.CharacterName {
		session.Character = nil
		return b.Send(session.Conn, SelectCharacter, []byte{0, 0, 0, 0})
	}

	// Provide stats of the selected character
	info := session.Character.Info
	response := make([]byte, 60)
	response[0] = 1
	copy(response[4:], info.ToBytes())

	return b.Send(session.Conn, SelectCharacter, response)
}

type SelectCharacterRequest []byte

type SelectCharacterRequestData struct {
	Username      string
	CharacterName string
}

func (r SelectCharacterRequest) Parse() (data SelectCharacterRequestData, err error) {
	split := bytes.SplitN(r, []byte{0}, 3)
	if len(split) != 3 {
		return data, fmt.Errorf("packet-76: no enough arguments, malformed request payload: %v", r)
	}
	data.Username = string(split[0])
	data.CharacterName = string(split[1])
	return data, nil
}
