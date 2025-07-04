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
	"github.com/dimspell/gladiator/internal/model"
)

func (b *Backend) HandleCreateCharacter(ctx context.Context, session *bsession.Session, req CreateCharacterRequest) error {
	if session.UserID == 0 {
		return fmt.Errorf("packet-92: user is not logged in")
	}

	data, err := req.Parse()
	if err != nil {
		slog.Warn("Invalid packet", logging.Error(err))
		return nil
	}

	respChar, err := b.characterClient.CreateCharacter(ctx,
		connect.NewRequest(&multiv1.CreateCharacterRequest{
			UserId:        session.UserID,
			CharacterName: data.CharacterName,
			Stats:         data.Info,
		}))
	if err != nil {
		slog.Error("Could not create a character", logging.Error(err))
		return session.SendToGame(packet.CreateCharacter, []byte{0, 0, 0, 0})
	}

	slog.Info("packet-92: new character created",
		"character", respChar.Msg.Character.CharacterName,
		"username", data.Username)

	return session.SendToGame(packet.CreateCharacter, []byte{1, 0, 0, 0})
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
	rd := packet.NewReader(r)

	data.Info, err = rd.ReadNBytes(56)
	if err != nil {
		return data, fmt.Errorf("packet-92: could not read character info: %w", err)
	}
	data.ParsedInfo = model.ParseCharacterInfo(data.Info)

	data.Username, err = rd.ReadString()
	if err != nil {
		return data, fmt.Errorf("packet-92: could not read username: %w", err)
	}
	data.CharacterName, err = rd.ReadString()
	if err != nil {
		return data, fmt.Errorf("packet-92: could not read character: %w", err)
	}

	return data, rd.Close()
}
