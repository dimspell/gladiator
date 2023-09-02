package backend

import (
	"bytes"
	"fmt"

	"github.com/dispel-re/dispel-multi/model"
)

func (b *Backend) HandleSelectCharacter(session *model.Session, req SelectCharacterRequest) error {
	_, characterName, err := req.Parse()
	if err != nil {
		return err
	}

	for _, c := range session.User.Characters {
		if c.CharacterName == characterName {
			session.Character = &c
			break
		}
	}

	// No characters owned by player
	if session.Character == nil || session.Character.CharacterName != characterName {
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

func (r SelectCharacterRequest) Parse() (user string, character string, err error) {
	split := bytes.SplitN(r, []byte{0}, 3)
	if len(split) != 3 {
		return user, character, fmt.Errorf("packet-76: no enough arguments, malformed request payload: %v", r)
	}
	user = string(split[0])
	character = string(split[1])
	return user, character, nil
}
