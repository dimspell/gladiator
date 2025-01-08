package backend

import (
	"context"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"
	multiv1 "github.com/dimspell/gladiator/gen/multi/v1"
	"github.com/dimspell/gladiator/internal/backend/bsession"
	"github.com/dimspell/gladiator/internal/backend/packet"
)

// 008-JP1-20001
func (b *Backend) HandleCreateNewAccount(ctx context.Context, session *bsession.Session, req CreateNewAccountRequest) error {
	data, err := req.Parse()
	if err != nil {
		slog.Warn("Invalid packet", "error", err)
		return nil
	}

	respUser, err := b.userClient.CreateUser(ctx, connect.NewRequest(&multiv1.CreateUserRequest{
		Username: data.Username,
		Password: data.Password,
	}))
	if err != nil {
		slog.Warn("packet-42: could not save new user into database", "err", err)
		return session.Send(packet.CreateNewAccount, []byte{0, 0, 0, 0})
	}

	slog.Info("packet-42: new user created", "user", respUser.Msg.User.Username)

	return session.Send(packet.CreateNewAccount, []byte{1, 0, 0, 0})
}

type CreateNewAccountRequest []byte

type CreateNewAccountRequestData struct {
	CDKey    uint32
	Username string
	Password string
	Unknown  []byte
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
	data.Unknown, _ = rd.ReadRestBytes()

	return data, rd.Close()
}

type CreateNewAccountResponse [4]byte
