package backend

import (
	"context"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"
	multiv1 "github.com/dimspell/gladiator/gen/multi/v1"
	"github.com/dimspell/gladiator/internal/backend/bsession"
	"github.com/dimspell/gladiator/internal/backend/packet"
)

func (b *Backend) HandleGetCharacterSpells(ctx context.Context, session *bsession.Session, req GetCharacterSpellsRequest) error {
	if session.UserID == 0 {
		return fmt.Errorf("packet-72: user is not logged in")
	}

	data, err := req.Parse()
	if err != nil {
		slog.Warn("Invalid packet", "error", err)
		return nil
	}

	respChar, err := b.characterClient.GetCharacter(ctx, connect.NewRequest(&multiv1.GetCharacterRequest{
		UserId:        session.UserID,
		CharacterName: data.CharacterName,
	}))
	if err != nil {
		return err
	}

	character := respChar.Msg.Character

	if len(character.Spells) != 43 {
		slog.Warn("packet-72: spells array should be 43-chars long", "spells", character.Spells, "err", err)
		return nil
	}
	for i := 0; i < 41; i++ {
		if character.Spells[i] == 0 {
			character.Spells[i] = 1
		}
	}

	return session.SendFromBackend(packet.GetCharacterSpells, character.Spells)
}

type GetCharacterSpellsRequest []byte

type GetCharacterSpellsRequestData struct {
	Username      string
	CharacterName string
}

func (r GetCharacterSpellsRequest) Parse() (data GetCharacterSpellsRequestData, err error) {
	rd := packet.NewReader(r)

	data.Username, err = rd.ReadString()
	if err != nil {
		return data, fmt.Errorf("packet-72: malformed username: %w", err)
	}
	data.CharacterName, err = rd.ReadString()
	if err != nil {
		return data, fmt.Errorf("packet-72: malformed character name: %w", err)
	}

	return data, rd.Close()
}
