package backend

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	multiv1 "github.com/dispel-re/dispel-multi/gen/multi/v1"
	"github.com/dispel-re/dispel-multi/model"
)

func (b *Backend) HandleSelectCharacter(session *model.Session, req SelectCharacterRequest) error {
	if session.UserID == 0 {
		return fmt.Errorf("packet-76: user is not logged in")
	}

	data, err := req.Parse()
	if err != nil {
		return err
	}

	respChar, err := b.CharacterClient.GetCharacter(context.TODO(),
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
	split := bytes.SplitN(r, []byte{0}, 3)
	if len(split) != 3 {
		return data, fmt.Errorf("packet-76: no enough arguments, malformed request payload: %v", r)
	}
	data.Username = string(split[0])
	data.CharacterName = string(split[1])
	return data, nil
}
