package backend

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"

	"connectrpc.com/connect"
	multiv1 "github.com/dispel-re/dispel-multi/gen/multi/v1"
	"github.com/dispel-re/dispel-multi/model"
)

// HandleUpdateCharacterStats handles 0x6cff (255-108) command.
//
// It can be received by the game server in multiple scenarios:
//   - .
func (b *Backend) HandleUpdateCharacterStats(session *model.Session, req UpdateCharacterStatsRequest) error {
	if session.UserID == 0 {
		return fmt.Errorf("packet-108: user is not logged in")
	}

	data, err := req.Parse()
	if err != nil {
		return fmt.Errorf("packet-108: could not parse request: %w", err)
	}

	_, err = b.CharacterClient.PutStats(context.TODO(),
		connect.NewRequest(&multiv1.PutStatsRequest{
			UserId:        session.UserID,
			CharacterName: data.Character,
			Stats:         data.Info,
		}))
	if err != nil {
		return err
	}

	return b.Send(session.Conn, UpdateCharacterStats, []byte{})
}

type UpdateCharacterStatsRequest []byte

type UpdateCharacterStatsRequestData struct {
	Info       []byte
	ParsedInfo model.CharacterInfo
	User       string
	Character  string
	Unknown    []byte
}

func (r UpdateCharacterStatsRequest) Parse() (data UpdateCharacterStatsRequestData, err error) {
	if len(r) < 56 {
		return data, fmt.Errorf("packet is too short: %s", base64.StdEncoding.EncodeToString(r))
	}
	split := bytes.SplitN(r[56:], []byte{0}, 3)
	if len(split) != 3 {
		return data, fmt.Errorf("no enough arguments, malformed request payload: %s", base64.StdEncoding.EncodeToString(r))
	}

	data.Info = r[:56]
	data.ParsedInfo = model.ParseCharacterInfo(r[:56])
	data.User = string(split[0])
	data.Character = string(split[1])
	data.Unknown = split[2]

	return data, err
}
