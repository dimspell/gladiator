package backend

import (
	"context"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"
	multiv1 "github.com/dimspell/gladiator/gen/multi/v1"
	"github.com/dimspell/gladiator/internal/app/logger/logging"
	"github.com/dimspell/gladiator/internal/backend/bsession"
	"github.com/dimspell/gladiator/internal/backend/packet"
)

func (b *Backend) HandleUpdateCharacterInventory(ctx context.Context, session *bsession.Session, req UpdateCharacterInventoryRequest) error {
	if session.UserID == 0 {
		return fmt.Errorf("packet-44: user is not logged in")
	}
	data, err := req.Parse()
	if err != nil {
		slog.Warn("Invalid packet", logging.Error(err))
		return nil
	}

	_, err = b.characterClient.PutInventoryCharacter(ctx,
		connect.NewRequest(&multiv1.PutInventoryRequest{
			UserId:        session.UserID,
			CharacterName: data.CharacterName,
			Inventory:     data.Inventory,
		}))
	if err != nil {
		return err
	}

	return session.SendToGame(packet.UpdateCharacterInventory, []byte{1, 0, 0, 0})
}

type UpdateCharacterInventoryRequest []byte

type UpdateCharacterInventoryRequestData struct {
	Username      string
	CharacterName string
	Inventory     []byte
}

func (r UpdateCharacterInventoryRequest) Parse() (data UpdateCharacterInventoryRequestData, err error) {
	rd := packet.NewReader(r)

	data.Username, err = rd.ReadString()
	if err != nil {
		return data, fmt.Errorf("packet-44: malformed username: %w", err)
	}
	data.CharacterName, err = rd.ReadString()
	if err != nil {
		return data, fmt.Errorf("packet-44: malformed character name: %w", err)
	}
	data.Inventory, err = rd.ReadNBytes(207)
	if err != nil {
		return data, fmt.Errorf("packet-44: malformed inventory: %w", err)
	}

	return data, rd.Close()
}
