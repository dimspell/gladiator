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

// 008-JP1-20001
func (b *Backend) HandleCreateNewAccount(ctx context.Context, session *bsession.Session, req CreateNewAccountRequest) error {
	data, err := req.Parse()
	if err != nil {
		slog.Warn("Invalid packet", logging.Error(err))
		return session.SendToGame(packet.CreateNewAccount, []byte{0, 0, 0, 0})
	}

	if len(data.Username) == 0 || len(data.Username) > 8 {
		slog.Warn("Incorrect username - must be less than 9 characters", "length", len(data.Username))
		return session.SendToGame(packet.CreateNewAccount, []byte{0, 0, 0, 0})
	}
	if len(data.Password) == 0 || len(data.Password) > 8 {
		slog.Warn("Incorrect password - must be less than 9 characters", "length", len(data.Password))
		return session.SendToGame(packet.CreateNewAccount, []byte{0, 0, 0, 0})
	}

	respUser, err := b.userClient.CreateUser(ctx, connect.NewRequest(&multiv1.CreateUserRequest{
		Username: data.Username,
		Password: data.Password,
	}))
	if err != nil {
		slog.Warn("packet-42: could not save a new user into database", logging.Error(err))
		return session.SendToGame(packet.CreateNewAccount, []byte{0, 0, 0, 0})
	}

	slog.Info("packet-42: new user created", "user", respUser.Msg.User.Username)

	return session.SendToGame(packet.CreateNewAccount, []byte{1, 0, 0, 0})
}

type CreateNewAccountRequest []byte

type CreateNewAccountRequestData struct {
	CDKey    uint32
	Username string
	Password string
}

func (r CreateNewAccountRequest) Parse() (data CreateNewAccountRequestData, err error) {
	rd := packet.NewReader(r)

	data.CDKey, err = rd.ReadUint32()
	if err != nil {
		return data, fmt.Errorf("packet-42: malformed cdkey: %w", err)
	}
	data.Password, err = rd.ReadString()
	if err != nil {
		return data, fmt.Errorf("packet-42: malformed password: %w", err)
	}
	data.Username, err = rd.ReadString()
	if err != nil {
		return data, fmt.Errorf("packet-42: malformed username: %w", err)
	}

	return data, rd.Close()
}

type CreateNewAccountResponse [4]byte
