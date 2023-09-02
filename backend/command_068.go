package backend

import (
	"bytes"
	"fmt"

	"github.com/dispel-re/dispel-multi/model"
)

func (b *Backend) HandleGetCharacterInventory(session *model.Session, req GetCharacterInventoryRequest) error {
	resp := make([]byte, 207)

	for i, item := range session.Character.Inventory.Backpack {
		resp[0+i*3] = item.TypeId
		resp[1+i*3] = item.ItemId
		resp[2+i*3] = item.Unknown
	}
	for i, item := range session.Character.Inventory.Belt {
		resp[0+63+i*3] = item.TypeId
		resp[1+63+i*3] = item.ItemId
		resp[2+63+i*3] = item.Unknown
	}
	resp = append(resp, 0)

	return b.Send(session.Conn, GetCharacterInventory, resp)
}

type GetCharacterInventoryRequest []byte

func (r GetCharacterInventoryRequest) Parse() (username string, characterName string, unknown []byte, err error) {
	if bytes.Count(r, []byte{0}) != 3 {
		return username, characterName, unknown, fmt.Errorf("packet-61: malformed packet, not enough null-terminators")
	}

	split := bytes.SplitN(r, []byte{0}, 3)
	username = string(split[0])
	characterName = string(split[1])
	unknown = split[2]

	return username, characterName, unknown, nil
}
