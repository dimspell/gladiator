package backend

import "github.com/dispel-re/dispel-multi/model"

func (b *Backend) HandleUpdateCharacterInventory(session *model.Session, req UpdateCharacterInventoryRequest) error {
	// rd := bufio.NewReader(bytes.NewReader(buf[4:]))
	// _, _ = rd.ReadBytes(0)         // username
	// _, _ = rd.ReadBytes(0)         // character
	// backpack, _ := rd.ReadBytes(0) // inventory
	// printBackpack(backpack)

	return b.Send(session.Conn, UpdateCharacterInventory, []byte{1, 0, 0, 0})
}

type UpdateCharacterInventoryRequest []byte
