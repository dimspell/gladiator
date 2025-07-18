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

func (b *Backend) HandleUpdateCharacterSpells(ctx context.Context, session *bsession.Session, req UpdateCharacterSpellsRequest) error {
	if session.UserID == 0 {
		return fmt.Errorf("packet-73: user has been already logged in")
	}

	data, err := req.Parse()
	if err != nil {
		slog.Warn("Invalid packet", logging.Error(err))
		return nil
	}

	_, err = b.characterClient.PutSpells(ctx,
		connect.NewRequest(&multiv1.PutSpellsRequest{
			UserId:        session.UserID,
			CharacterName: data.CharacterName,
			Spells:        data.Spells,
		}))
	if err != nil {
		return fmt.Errorf("packet-73: could not update character spells: %s", err)
	}

	return session.SendToGame(packet.UpdateCharacterSpells, []byte{1, 0, 0, 0})
}

type UpdateCharacterSpellsRequest []byte

type UpdateCharacterSpellsRequestData struct {
	Username      string
	CharacterName string
	Spells        []byte
}

func (r UpdateCharacterSpellsRequest) Parse() (data UpdateCharacterSpellsRequestData, err error) {
	rd := packet.NewReader(r)

	data.Username, err = rd.ReadString()
	if err != nil {
		return data, fmt.Errorf("packet-73: malformed username: %w", err)
	}
	data.CharacterName, err = rd.ReadString()
	if err != nil {
		return data, fmt.Errorf("packet-73: malformed character name: %w", err)
	}
	data.Spells, err = rd.ReadNBytes(43)
	if err != nil {
		return data, fmt.Errorf("packet-73: malformed spells: %w", err)
	}

	return data, rd.Close()
}
