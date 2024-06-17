package backend

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/dimspell/gladiator/backend/packet"
	multiv1 "github.com/dimspell/gladiator/gen/multi/v1"
	"github.com/dimspell/gladiator/model"
)

func (b *Backend) HandleSelectCharacter(session *model.Session, req SelectCharacterRequest) error {
	if session.UserID == 0 {
		return fmt.Errorf("packet-76: user is not logged in")
	}

	data, err := req.Parse()
	if err != nil {
		slog.Warn("Invalid packet", "error", err)
		return nil
	}

	respChar, err := b.characterClient.GetCharacter(context.TODO(),
		connect.NewRequest(&multiv1.GetCharacterRequest{
			UserId:        session.UserID,
			CharacterName: data.CharacterName,
		}))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return b.Send(session.Conn, SelectCharacter, []byte{0, 0, 0, 0})
		}
		return fmt.Errorf("packet-76: no characters found owned by player: %s", err)
	}

	response := make([]byte, 60)
	response[0] = 1
	copy(response[4:], respChar.Msg.Character.Stats)

	session.CharacterID = respChar.Msg.Character.CharacterId

	return b.Send(session.Conn, SelectCharacter, response)
}

type SelectCharacterRequest []byte

type SelectCharacterRequestData struct {
	Username      string
	CharacterName string
}

func (r SelectCharacterRequest) Parse() (data SelectCharacterRequestData, err error) {
	rd := packet.NewReader(r)

	data.Username, err = rd.ReadString()
	if err != nil {
		return data, fmt.Errorf("packet-76: malformed username: %w", err)
	}
	data.CharacterName, err = rd.ReadString()
	if err != nil {
		return data, fmt.Errorf("packet-76: malformed character name: %w", err)
	}

	return data, rd.Close()
}
