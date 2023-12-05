package backend

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"
	multiv1 "github.com/dispel-re/dispel-multi/gen/multi/v1"
	"github.com/dispel-re/dispel-multi/model"
)

func (b *Backend) HandleCreateCharacter(session *model.Session, req CreateCharacterRequest) error {
	if session.UserID == 0 {
		return fmt.Errorf("packet-92: user is not logged in")
	}

	data, err := req.Parse()
	if err != nil {
		return err
	}

	respChar, err := b.CharacterClient.CreateCharacter(context.TODO(),
		connect.NewRequest(&multiv1.CreateCharacterRequest{
			UserId:        session.UserID,
			CharacterName: data.CharacterName,
			Stats:         data.Info,
		}))
	if err != nil {
		slog.Error("Could not create a character", "err", err)
		return b.Send(session.Conn, CreateCharacter, []byte{0, 0, 0, 0})
	}

	slog.Info("packet-92: new character created",
		"character", respChar.Msg.Character.CharacterName,
		"username", data.Username)

	return b.Send(session.Conn, CreateCharacter, []byte{1, 0, 0, 0})
}

// TODO: check if there is any additional not recognised byte at the end like slot number
type CreateCharacterRequest []byte

type CreateCharacterRequestData struct {
	Info          []byte
	ParsedInfo    model.CharacterInfo
	Username      string
	CharacterName string
}

func (r CreateCharacterRequest) Parse() (data CreateCharacterRequestData, err error) {
	if len(r) < 56 {
		return data, fmt.Errorf("packet-92: packet is too short: %s", base64.StdEncoding.EncodeToString(r))
	}
	split := bytes.SplitN(r[56:], []byte{0}, 3)
	if len(split) != 3 {
		return data, fmt.Errorf("packet-92: no enough arguments, malformed request payload: %s", base64.StdEncoding.EncodeToString(r))
	}

	data.Info = r[:56]
	data.ParsedInfo = model.ParseCharacterInfo(r[:56])
	data.Username = string(split[0])
	data.CharacterName = string(split[1])

	return data, err
}
