package backend

import (
	"bytes"
	"context"
	"fmt"

	"connectrpc.com/connect"
	multiv1 "github.com/dispel-re/dispel-multi/gen/multi/v1"
	"github.com/dispel-re/dispel-multi/model"
)

func (b *Backend) HandleUpdateCharacterInventory(session *model.Session, req UpdateCharacterInventoryRequest) error {
	if session.UserID == 0 {
		return fmt.Errorf("packet-44: user is not logged in")
	}
	data, err := req.Parse()
	if err != nil {
		return err
	}

	_, err = b.CharacterClient.PutInventoryCharacter(context.TODO(),
		connect.NewRequest(&multiv1.PutInventoryRequest{
			UserId:        session.UserID,
			CharacterName: data.CharacterName,
			Inventory:     data.Inventory,
		}))
	if err != nil {
		return err
	}

	return b.Send(session.Conn, UpdateCharacterInventory, []byte{1, 0, 0, 0})
}

type UpdateCharacterInventoryRequest []byte

type UpdateCharacterInventoryRequestData struct {
	Username      string
	CharacterName string
	Inventory     []byte
}

func (r UpdateCharacterInventoryRequest) Parse() (data UpdateCharacterInventoryRequestData, err error) {
	if bytes.Count(r, []byte{0}) != 3 {
		return data, fmt.Errorf("packet-44: malformed payload: %v", r)
	}

	split := bytes.SplitN(r, []byte{0}, 4)
	if len(split[2]) != 207 {
		return data, fmt.Errorf("packet-44: invalid length of inventory array")
	}

	data.Username = string(split[0])
	data.CharacterName = string(split[1])
	data.Inventory = split[2]

	return data, err
}
