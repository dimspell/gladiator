package backend

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"
	multiv1 "github.com/dimspell/gladiator/gen/multi/v1"
	"github.com/dimspell/gladiator/internal/backend/packet"
)

func (b *Backend) HandleGetCharacterInventory(session *Session, req GetCharacterInventoryRequest) error {
	if session.UserID == 0 {
		return fmt.Errorf("packet-68: user is not logged in")
	}
	var err error
	session.observerOnce.Do(func() {
		// TODO: It would be better to have it in the 41, but then change the character here.
		err = b.RegisterNewObserver(session)
	})
	if err != nil {
		slog.Debug("packet-68: could not register observer", "error", err)
		return b.Send(session.Conn, GetCharacterInventory, []byte{0, 0, 0, 0})
	}

	data, err := req.Parse()
	if err != nil {
		slog.Warn("Invalid packet", "error", err)
		return nil
	}

	resp, err := b.characterClient.GetCharacter(context.TODO(),
		connect.NewRequest(&multiv1.GetCharacterRequest{
			UserId:        session.UserID,
			CharacterName: data.CharacterName,
		}))

	if err != nil {
		_ = b.Send(session.Conn, ReceiveMessage, NewGlobalMessage("system", "Inventory fetch failed, please try sign-in again"))

		var connectError *connect.Error
		if errors.As(err, &connectError) {
			if connectError.Code() == connect.CodeNotFound {
				return nil
			}
		}
		return fmt.Errorf("packet-68: could not fetch character %s: %s", data.CharacterName, err)
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
	rd := packet.NewReader(r)
	data.Username, err = rd.ReadString()
	if err != nil {
		return data, fmt.Errorf("packet-68: malformed username: %w", err)
	}
	data.CharacterName, err = rd.ReadString()
	if err != nil {
		return data, fmt.Errorf("packet-68: malformed character name: %w", err)
	}
	data.Unknown, _ = rd.ReadRestBytes()
	return data, rd.Close()
}
