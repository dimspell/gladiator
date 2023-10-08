package backend

import (
	"bytes"
	"fmt"

	"github.com/dispel-re/dispel-multi/model"
)

func (b *Backend) HandleUpdateCharacterInventory(session *model.Session, req UpdateCharacterInventoryRequest) error {
	// rd := bufio.NewReader(bytes.NewReader(buf[4:]))
	// _, _ = rd.ReadBytes(0)         // username
	// _, _ = rd.ReadBytes(0)         // character
	// backpack, _ := rd.ReadBytes(0) // inventory
	// printBackpack(backpack)

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
