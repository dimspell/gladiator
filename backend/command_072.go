package backend

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"
	multiv1 "github.com/dispel-re/dispel-multi/gen/multi/v1"
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

	respChar, err := b.CharacterClient.GetCharacter(context.TODO(), connect.NewRequest(&multiv1.GetCharacterRequest{
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

	return b.Send(session.Conn, GetCharacterSpells, character.Spells)
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
