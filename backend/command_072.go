package backend

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"

	"github.com/dispel-re/dispel-multi/internal/database"
	"github.com/dispel-re/dispel-multi/model"
)

func (b *Backend) HandleGetCharacterSpells(session *model.Session, req GetCharacterSpellsRequest) error {
	if session.UserID == 0 {
		return fmt.Errorf("packet-72: user is not logged in")
	}

	data, err := req.Parse()
	if err != nil {
		return err
	}

	character, err := b.DB.FindCharacter(context.TODO(), database.FindCharacterParams{
		CharacterName: data.CharacterName,
		UserID:        session.UserID,
	})
	if err != nil {
		return err
	}

	if !character.Spells.Valid {
		return nil
	}
	spells, err := base64.StdEncoding.DecodeString(character.Spells.String)
	if len(spells) != 43 {
		slog.Warn("packet-72: spells array should be 43-chars long", "spells", character.Spells.String, "err", err)
		return nil
	}
	for i := 0; i < 41; i++ {
		if spells[i] == 0 {
			spells[i] = 1
		}
	}

	return b.Send(session.Conn, GetCharacterSpells, spells)
}

type GetCharacterSpellsRequest []byte

type GetCharacterSpellsRequestData struct {
	Username      string
	CharacterName string
}

func (r GetCharacterSpellsRequest) Parse() (data GetCharacterSpellsRequestData, err error) {
	if bytes.Count(r, []byte{0}) < 2 {
		return data, fmt.Errorf("packet-72: malformed packet, not enough null-terminators")
	}
	split := bytes.SplitN(r, []byte{0}, 3)
	data.Username = string(split[0])
	data.CharacterName = string(split[1])

	return data, nil
}
