package backend

import (
	"bytes"
	"fmt"

	"github.com/dispel-re/dispel-multi/model"
)

func (b *Backend) HandleDeleteCharacter(session *model.Session, req DeleteCharacterRequest) error {
	_, characterName, err := req.Parse()
	if err != nil {
		return err
	}

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

func (r DeleteCharacterRequest) Parse() (username string, characterName string, err error) {
	if bytes.Count(r, []byte{0}) < 2 {
		return "", "", fmt.Errorf("packet-61: malformed packet, not enough null-terminators")
	}

	split := bytes.SplitN(r, []byte{0}, 3)
	username = string(split[0])
	characterName = string(split[1])
	return username, characterName, nil
}
