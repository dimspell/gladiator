package backend

import (
	"bytes"
	"encoding/base64"
	"fmt"

	"github.com/dispel-re/dispel-multi/model"
)

func (b *Backend) HandleCreateCharacter(session *model.Session, req CreateCharacterRequest) error {
	if session.User == nil {
		return fmt.Errorf("packet-92: user is not logged in")
	}

	data, err := req.Parse()
	if err != nil {
		return err
	}

	newCharacter := model.Character{
		CharacterName: data.CharacterName,
		Slot:          0,
		Info:          data.CharacterInfo,
		Inventory:     model.CharacterInventory{},
		Spells:        nil,
	}
	session.User.Characters = append(session.User.Characters, newCharacter)

	return b.Send(session.Conn, CreateCharacter, []byte{1, 0, 0, 0})
}

// TODO: check if there is any additional not recognised byte at the end
type CreateCharacterRequest []byte

type CreateCharacterRequestData struct {
	CharacterInfo model.CharacterInfo
	Username      string
	CharacterName string
}

func (r CreateCharacterRequest) Parse() (data CreateCharacterRequestData, err error) {
	if len(r) < 56 {
		return data, fmt.Errorf("packet-92: packet is too short: %s", base64.StdEncoding.EncodeToString(r))
	}
	split := bytes.SplitN(r[56:], []byte{0}, 3)
	if len(split) != 3 {
		return data, fmt.Errorf("packet-92: no enough arguments, malformed request payload: %s", base64.StdEncoding.EncodeToString(r))
	}

	data.CharacterInfo = model.NewCharacterInfo(r[:56])
	data.Username = string(split[0])
	data.CharacterName = string(split[1])

	return data, err
}
