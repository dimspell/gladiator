package backend

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"
	multiv1 "github.com/dimspell/gladiator/gen/multi/v1"
	"github.com/dimspell/gladiator/internal/app/logger/logging"
	"github.com/dimspell/gladiator/internal/backend/bsession"
	"github.com/dimspell/gladiator/internal/backend/packet"
)

func (b *Backend) HandleSelectCharacter(ctx context.Context, session *bsession.Session, req SelectCharacterRequest) error {
	if session.UserID == 0 {
		return fmt.Errorf("packet-76: user is not logged in")
	}

	data, err := req.Parse()
	if err != nil {
		slog.Warn("Invalid packet", logging.Error(err))
		return nil
	}

	respChar, err := b.characterClient.GetCharacter(ctx,
		connect.NewRequest(&multiv1.GetCharacterRequest{
			UserId:        session.UserID,
			CharacterName: data.CharacterName,
		}))
	if err != nil {
		var connectError *connect.Error
		if errors.As(err, &connectError) {
			if connectError.Code() == connect.CodeNotFound {
				return session.SendToGame(packet.SelectCharacter, []byte{0, 0, 0, 0})
			}
		}
		return fmt.Errorf("packet-76: no characters found owned by player: %s", err)
	}

	response := make([]byte, 60)
	response[0] = 1 // Exist, flag - first 4 bytes
	copy(response[4:], respChar.Msg.Character.Stats)

	session.UpdateCharacter(respChar.Msg.Character)

	return session.SendToGame(packet.SelectCharacter, response)
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
