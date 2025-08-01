package backend

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"
	multiv1 "github.com/dimspell/gladiator/gen/multi/v1"
	"github.com/dimspell/gladiator/internal/app/logger/logging"
	"github.com/dimspell/gladiator/internal/backend/bsession"
	"github.com/dimspell/gladiator/internal/backend/packet"
)

func (b *Backend) HandleGetCharacterInventory(ctx context.Context, session *bsession.Session, req GetCharacterInventoryRequest) error {
	if session.UserID == 0 {
		return fmt.Errorf("packet-68: user is not logged in")
	}

	// Once the character is selected (or created), the next packet will
	// be 68 (GetCharacterInventory). This is the perfect time to tell the
	// lobby server that someone has joined and is ready to chat & play.
	if err := session.InitObserver(b.RegisterNewObserver); err != nil {
		return fmt.Errorf("packet-68: could not select the character: %w", err)
	}

	data, err := req.Parse()
	if err != nil {
		slog.Warn("Invalid packet", logging.Error(err))
		return nil
	}

	resp, err := b.characterClient.GetCharacter(ctx,
		connect.NewRequest(&multiv1.GetCharacterRequest{
			UserId:        session.UserID,
			CharacterName: data.CharacterName,
		}))

	if err != nil {
		_ = session.SendToGame(packet.ReceiveMessage, NewGlobalMessage("system", "Inventory fetch failed, please try sign-in again"))

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
		slog.Warn("packet-68: inventory array should be 207-chars long", "inventory", inventory)
		return nil
	}

	return session.SendToGame(packet.GetCharacterInventory, inventory)
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
