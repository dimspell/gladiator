package backend

import (
	"context"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/dimspell/gladiator/backend/packet"
	multiv1 "github.com/dimspell/gladiator/gen/multi/v1"
)

func (b *Backend) HandleUpdateCharacterSpells(session *Session, req UpdateCharacterSpellsRequest) error {
	if session.UserID == 0 {
		return fmt.Errorf("packet-73: user has been already logged in")
	}

	data, err := req.Parse()
	if err != nil {
		slog.Warn("Invalid packet", "error", err)
		return nil
	}

	_, err = b.characterClient.PutSpells(context.TODO(),
		connect.NewRequest(&multiv1.PutSpellsRequest{
			UserId:        session.UserID,
			CharacterName: data.CharacterName,
			Spells:        data.Spells,
		}))
	if err != nil {
		return fmt.Errorf("packet-73: could not update character spells: %s", err)
	}

	return b.Send(session.Conn, UpdateCharacterSpells, []byte{1, 0, 0, 0})
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
