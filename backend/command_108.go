package backend

import (
	"context"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/dimspell/gladiator/backend/packet"
	multiv1 "github.com/dimspell/gladiator/gen/multi/v1"
	"github.com/dimspell/gladiator/model"
)

// HandleUpdateCharacterStats handles 0x6cff (255-108) command.
//
// It can be received by the game server in multiple scenarios:
//   - .
func (b *Backend) HandleUpdateCharacterStats(session *Session, req UpdateCharacterStatsRequest) error {
	if session.UserID == 0 {
		return fmt.Errorf("packet-108: user is not logged in")
	}

	data, err := req.Parse()
	if err != nil {
		slog.Warn("Invalid packet", "error", err)
		return err
	}

	_, err = b.characterClient.PutStats(context.TODO(),
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
	Username   string
	Character  string
	Unknown    []byte
}

func (r UpdateCharacterStatsRequest) Parse() (data UpdateCharacterStatsRequestData, err error) {
	rd := packet.NewReader(r)

	data.Info, err = rd.ReadNBytes(56)
	if err != nil {
		return data, fmt.Errorf("packet-108: could not read character info: %w", err)
	}
	data.ParsedInfo = model.ParseCharacterInfo(data.Info)

	data.Username, err = rd.ReadString()
	if err != nil {
		return data, fmt.Errorf("packet-108: could not read username: %w", err)
	}
	data.Character, err = rd.ReadString()
	if err != nil {
		return data, fmt.Errorf("packet-108: could not read character: %w", err)
	}
	data.Unknown, _ = rd.ReadRestBytes()

	return data, rd.Close()
}
