package backend

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"

	"github.com/dispel-re/dispel-multi/internal/database"
	"github.com/dispel-re/dispel-multi/model"
)

func (b *Backend) HandleGetCharacterInventory(session *model.Session, req GetCharacterInventoryRequest) error {
	// resp := make([]byte, 207)
	//
	// inventory := model.CharacterInventory{
	// 	Backpack: [63]model.InventoryItem{
	// 		{TypeId: 4, ItemId: 1, Unknown: 17},
	// 		{TypeId: 1, ItemId: 8, Unknown: 33},
	// 		{TypeId: 1, ItemId: 8, Unknown: 33},
	// 		{TypeId: 1, ItemId: 8, Unknown: 65},
	// 		{TypeId: 1, ItemId: 8, Unknown: 65},
	// 		{TypeId: 1, ItemId: 15, Unknown: 97},
	// 		{TypeId: 1, ItemId: 15, Unknown: 97},
	//
	// 		{TypeId: 11, ItemId: 101, Unknown: 121},
	// 		{TypeId: 1, ItemId: 8, Unknown: 33},
	// 		{TypeId: 1, ItemId: 8, Unknown: 33},
	// 		{TypeId: 1, ItemId: 8, Unknown: 65},
	// 		{TypeId: 1, ItemId: 8, Unknown: 65},
	// 		{TypeId: 11, ItemId: 101, Unknown: 97},
	// 		{TypeId: 11, ItemId: 101, Unknown: 97},
	//
	// 		{TypeId: 11, ItemId: 101, Unknown: 19},
	// 		{TypeId: 1, ItemId: 8, Unknown: 33},
	// 		{TypeId: 1, ItemId: 8, Unknown: 33},
	// 		{TypeId: 1, ItemId: 8, Unknown: 65},
	// 		{TypeId: 1, ItemId: 8, Unknown: 65},
	// 		{TypeId: 11, ItemId: 101, Unknown: 83},
	// 		{TypeId: 11, ItemId: 101, Unknown: 121},
	//
	// 		{TypeId: 11, ItemId: 101, Unknown: 19},
	// 		{TypeId: 11, ItemId: 101, Unknown: 19},
	// 		{TypeId: 11, ItemId: 101, Unknown: 51},
	// 		{TypeId: 11, ItemId: 101, Unknown: 51},
	// 		{TypeId: 11, ItemId: 101, Unknown: 83},
	// 		{TypeId: 11, ItemId: 101, Unknown: 83},
	// 		{TypeId: 11, ItemId: 101, Unknown: 121},
	//
	// 		{TypeId: 11, ItemId: 101, Unknown: 21},
	// 		{TypeId: 11, ItemId: 101, Unknown: 21},
	// 		{TypeId: 11, ItemId: 101, Unknown: 53},
	// 		{TypeId: 11, ItemId: 101, Unknown: 53},
	// 		{TypeId: 11, ItemId: 101, Unknown: 85},
	// 		{TypeId: 11, ItemId: 101, Unknown: 85},
	// 		{TypeId: 11, ItemId: 101, Unknown: 121},
	//
	// 		{TypeId: 11, ItemId: 101, Unknown: 21},
	// 		{TypeId: 11, ItemId: 101, Unknown: 21},
	// 		{TypeId: 11, ItemId: 101, Unknown: 53},
	// 		{TypeId: 11, ItemId: 101, Unknown: 53},
	// 		{TypeId: 11, ItemId: 101, Unknown: 85},
	// 		{TypeId: 11, ItemId: 101, Unknown: 85},
	// 		{TypeId: 11, ItemId: 101, Unknown: 121},
	//
	// 		{TypeId: 11, ItemId: 101, Unknown: 121},
	// 		{TypeId: 11, ItemId: 101, Unknown: 121},
	// 		{TypeId: 11, ItemId: 101, Unknown: 121},
	// 		{TypeId: 11, ItemId: 101, Unknown: 121},
	// 		{TypeId: 11, ItemId: 101, Unknown: 121},
	// 		{TypeId: 11, ItemId: 101, Unknown: 121},
	// 		{TypeId: 11, ItemId: 101, Unknown: 121},
	//
	// 		{TypeId: 11, ItemId: 101, Unknown: 121},
	// 		{TypeId: 11, ItemId: 101, Unknown: 121},
	// 		{TypeId: 11, ItemId: 101, Unknown: 121},
	// 		{TypeId: 11, ItemId: 101, Unknown: 121},
	// 		{TypeId: 11, ItemId: 101, Unknown: 121},
	// 		{TypeId: 11, ItemId: 101, Unknown: 121},
	// 		{TypeId: 11, ItemId: 101, Unknown: 121},
	//
	// 		{TypeId: 11, ItemId: 101, Unknown: 121},
	// 		{TypeId: 11, ItemId: 101, Unknown: 121},
	// 		{TypeId: 11, ItemId: 101, Unknown: 121},
	// 		{TypeId: 11, ItemId: 101, Unknown: 121},
	// 		{TypeId: 11, ItemId: 101, Unknown: 121},
	// 		{TypeId: 11, ItemId: 101, Unknown: 121},
	// 		{TypeId: 11, ItemId: 101, Unknown: 121},
	// 	},
	// 	Belt: [6]model.InventoryItem{
	// 		{TypeId: 11, ItemId: 101, Unknown: 97},
	// 		{TypeId: 11, ItemId: 101, Unknown: 97},
	// 		{TypeId: 11, ItemId: 101, Unknown: 97},
	// 		{TypeId: 11, ItemId: 101, Unknown: 97},
	// 		{TypeId: 11, ItemId: 101, Unknown: 97},
	// 		{TypeId: 11, ItemId: 101, Unknown: 97},
	// 	},
	// }
	// for i, item := range inventory.Backpack {
	// 	resp[0+i*3] = item.TypeId
	// 	resp[1+i*3] = item.ItemId
	// 	resp[2+i*3] = item.Unknown
	// }
	// for i, item := range inventory.Belt {
	// 	resp[0+63+i*3] = item.TypeId
	// 	resp[1+63+i*3] = item.ItemId
	// 	resp[2+63+i*3] = item.Unknown
	// }
	// resp = append(resp, 0)
	if session.UserID == 0 {
		return fmt.Errorf("packet-68: user is not logged in")
	}

	data, err := req.Parse()
	if err != nil {
		return err
	}

	character, err := b.DB.FindCharacter(context.TODO(), database.FindCharacterParams{
		UserID:        session.UserID,
		CharacterName: data.CharacterName,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return b.Send(session.Conn, SelectCharacter, []byte{0, 0, 0, 0})
		}
		return fmt.Errorf("packet-76: no characters found owned by player: %s", err)
	}

	if !character.Inventory.Valid {
		return nil
	}
	inventory, err := base64.StdEncoding.DecodeString(character.Inventory.String)
	if len(inventory) != 207 {
		slog.Warn("packet-68: inventory array should be 207-chars long", "inventory", character.Inventory.String, "err", err)
		return nil
	}

	return b.Send(session.Conn, GetCharacterInventory, inventory)
}

type GetCharacterInventoryRequest []byte

type GetCharacterInventoryRequestData struct {
	Username      string
	CharacterName string
	Unknown       []byte
}

func (r GetCharacterInventoryRequest) Parse() (data GetCharacterInventoryRequestData, err error) {
	if bytes.Count(r, []byte{0}) != 3 {
		return data, fmt.Errorf("packet-61: malformed packet, not enough null-terminators")
	}
	split := bytes.SplitN(r, []byte{0}, 3)

	data.Username = string(split[0])
	data.CharacterName = string(split[1])
	data.Unknown = split[2]

	return data, nil
}
