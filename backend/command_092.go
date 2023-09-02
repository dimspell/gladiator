package backend

import (
	"bytes"

	"github.com/dispel-re/dispel-multi/model"
)

func (b *Backend) HandleCreateCharacter(session *model.Session, req CreateCharacterRequest) error {
	info := req.CharacterInfo()
	_, character := req.UserAndCharacterName()

	newCharacter := model.Character{
		CharacterName: character,
		Slot:          0,
		Info:          info,
		Inventory:     model.CharacterInventory{},
		Spells:        nil,
	}
	session.User.Characters = append(session.User.Characters, newCharacter)

	return b.Send(session.Conn, CreateCharacter, []byte{1, 0, 0, 0})
}

type CreateCharacterRequest []byte

func (r CreateCharacterRequest) CharacterInfo() model.CharacterInfo {
	return model.NewCharacterInfo(r[:56])
}

func (r CreateCharacterRequest) UserAndCharacterName() (username string, characterName string) {
	split := bytes.SplitN(r, []byte{0}, 3)
	username = string(split[0])
	characterName = string(split[1])
	return username, characterName
}
