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

func (r UpdateCharacterInventoryRequest) Parse() (username string, character string, inventory []byte, err error) {
	if bytes.Count(r, []byte{0}) != 3 {
		return username, character, inventory, fmt.Errorf("packet-44: malformed payload: %v", r)
	}

	split := bytes.SplitN(r, []byte{0}, 4)
	if len(split[2]) != 207 {
		return username, character, inventory, fmt.Errorf("packet-44: invalid length of inventory array")
	}

	username = string(split[0])
	character = string(split[1])
	inventory = split[2]

	return username, character, inventory, err
}
