package backend

import (
	"context"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"
	multiv1 "github.com/dimspell/gladiator/gen/multi/v1"
	"github.com/dimspell/gladiator/internal/app/logger/logging"
	"github.com/dimspell/gladiator/internal/backend/bsession"
	"github.com/dimspell/gladiator/internal/backend/packet"
)

func (b *Backend) HandleDeleteCharacter(ctx context.Context, session *bsession.Session, req DeleteCharacterRequest) error {
	if session.UserID == 0 {
		return fmt.Errorf("packet-61: user is not logged in")
	}

	data, err := req.Parse()
	if err != nil {
		slog.Warn("Invalid packet", logging.Error(err))
		return nil
	}

	if _, err := b.characterClient.DeleteCharacter(ctx,
		connect.NewRequest(&multiv1.DeleteCharacterRequest{
			UserId:        session.UserID,
			CharacterName: data.CharacterName,
		}),
	); err != nil {
		return err
	}

	response := make([]byte, len(data.CharacterName)+1)
	copy(response, data.CharacterName)

	return session.SendToGame(packet.DeleteCharacter, response)
}

type DeleteCharacterRequest []byte

type DeleteCharacterRequestData struct {
	Username      string
	CharacterName string
}

func (r DeleteCharacterRequest) Parse() (data DeleteCharacterRequestData, err error) {
	rd := packet.NewReader(r)

	data.Username, err = rd.ReadString()
	if err != nil {
		return data, fmt.Errorf("packet-61: malformed username: %w", err)
	}
	data.CharacterName, err = rd.ReadString()
	if err != nil {
		return data, fmt.Errorf("packet-61: malformed character name: %w", err)
	}

	return data, rd.Close()
}
