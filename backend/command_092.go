package backend

import (
	"bytes"

	"github.com/dispel-re/dispel-multi/model"
)

func (b *Backend) HandleCreateCharacter(session *model.Session, req CreateCharacterRequest) error {
	info, _, character, err := req.Parse()
	if err != nil {
		return err
	}

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

// TODO: check if there is any additional not recognised byte at the end
type CreateCharacterRequest []byte

func (r CreateCharacterRequest) Parse() (info model.CharacterInfo, user string, character string, err error) {
	info = model.NewCharacterInfo(r[:56])

	split := bytes.SplitN(r[56:], []byte{0}, 3)
	user = string(split[0])
	character = string(split[1])

	return info, user, character, err
}
