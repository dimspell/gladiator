package backend

import (
	"bytes"
	"context"
	"fmt"

	"connectrpc.com/connect"
	multiv1 "github.com/dispel-re/dispel-multi/gen/multi/v1"
	"github.com/dispel-re/dispel-multi/model"
)

func (b *Backend) HandleDeleteCharacter(session *model.Session, req DeleteCharacterRequest) error {
	if session.UserID == 0 {
		return fmt.Errorf("packet-61: user is not logged in")
	}

	data, err := req.Parse()
	if err != nil {
		return err
	}

	if _, err := b.CharacterClient.DeleteCharacter(context.TODO(),
		connect.NewRequest(&multiv1.DeleteCharacterRequest{
			UserId:        session.UserID,
			CharacterName: data.CharacterName,
		}),
	); err != nil {
		return err
	}

	response := make([]byte, len(data.CharacterName)+1)
	copy(response, data.CharacterName)

	return b.Send(session.Conn, DeleteCharacter, response)
}

type DeleteCharacterRequest []byte

type DeleteCharacterRequestData struct {
	Username      string
	CharacterName string
}

func (r DeleteCharacterRequest) Parse() (data DeleteCharacterRequestData, err error) {
	if bytes.Count(r, []byte{0}) < 2 {
		return data, fmt.Errorf("packet-61: malformed packet, not enough null-terminators")
	}

	split := bytes.SplitN(r, []byte{0}, 3)
	data.Username = string(split[0])
	data.CharacterName = string(split[1])

	return data, nil
}
