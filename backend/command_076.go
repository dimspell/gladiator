package backend

import (
	"bytes"

	"github.com/dispel-re/dispel-multi/model"
)

func (b *Backend) HandleSelectCharacter(session *model.Session, req SelectCharacterRequest) error {
	_, characterName := req.UserAndCharacterName()

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

func (r SelectCharacterRequest) UserAndCharacterName() (username string, characterName string) {
	split := bytes.SplitN(r, []byte{0}, 3)
	username = string(split[0])
	characterName = string(split[1])
	return username, characterName
}
