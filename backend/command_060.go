package backend

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"

	"connectrpc.com/connect"
	multiv1 "github.com/dispel-re/dispel-multi/gen/multi/v1"
	"github.com/dispel-re/dispel-multi/model"
)

func (b *Backend) HandleGetCharacters(session *model.Session, req GetCharactersRequest) error {
	if session.UserID == 0 {
		return fmt.Errorf("packet-60: user is not logged in")
	}

	resp, err := b.CharacterClient.ListCharacters(context.TODO(),
		connect.NewRequest(&multiv1.ListCharactersRequest{
			UserId: session.UserID,
		}))

	if err != nil {
		return err
	}

	if len(resp.Msg.GetCharacters()) == 0 {
		return b.Send(session.Conn, GetCharacters, []byte{0, 0, 0, 0})
	}

	response := []byte{1, 0, 0, 0}
	response = binary.LittleEndian.AppendUint32(response, uint32(len(resp.Msg.GetCharacters())))
	for _, character := range resp.Msg.GetCharacters() {
		response = append(response, character.CharacterName...)
		response = append(response, 0)
	}
	return b.Send(session.Conn, GetCharacters, response)
}

type GetCharactersRequest []byte

func (r GetCharactersRequest) Parse() (username string, err error) {
	if bytes.Count(r, []byte{0}) != 1 {
		return username, fmt.Errorf("packet-44: malformed payload: %v", r)
	}

	split := bytes.SplitN(r, []byte{0}, 2)
	username = string(split[0])

	return username, err
}
