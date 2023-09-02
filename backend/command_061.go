package backend

import (
	"bytes"

	"github.com/dispel-re/dispel-multi/model"
)

func (b *Backend) HandleDeleteCharacter(session *model.Session, req DeleteCharacterRequest) error {
	_, characterName := req.UsernameAndCharacterName()

	var characters []model.Character
	for _, ch := range session.User.Characters {
		if ch.CharacterName != characterName {
			characters = append(characters, ch)
		}
	}
	session.User.Characters = characters

	response := make([]byte, len(characterName)+1)
	copy(response, characterName)

	return b.Send(session.Conn, DeleteCharacter, response)
}

type DeleteCharacterRequest []byte

func (r DeleteCharacterRequest) UsernameAndCharacterName() (username string, characterName string) {
	if bytes.Count(r, []byte{0}) < 2 {
		return "", ""
	}

	split := bytes.SplitN(r, []byte{0}, 3)
	username = string(split[0])
	characterName = string(split[1])
	return username, characterName
}
