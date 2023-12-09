package backend

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"
	multiv1 "github.com/dispel-re/dispel-multi/gen/multi/v1"
	"github.com/dispel-re/dispel-multi/model"
)

func (b *Backend) HandleGetCharacterInventory(session *model.Session, req GetCharacterInventoryRequest) error {
	if session.UserID == 0 {
		return fmt.Errorf("packet-68: user is not logged in")
	}

	data, err := req.Parse()
	if err != nil {
		return err
	}

	resp, err := b.CharacterClient.GetCharacter(context.TODO(),
		connect.NewRequest(&multiv1.GetCharacterRequest{
			UserId:        session.UserID,
			CharacterName: data.CharacterName,
		}))

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return b.Send(session.Conn, SelectCharacter, []byte{0, 0, 0, 0})
		}
		return fmt.Errorf("packet-68: no characters found owned by player: %s", err)
	}

	inventory := resp.Msg.GetCharacter().GetInventory()
	if len(inventory) != 207 {
		slog.Warn("packet-68: inventory array should be 207-chars long", "inventory", inventory, "err", err)
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
