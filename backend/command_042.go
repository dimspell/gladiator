package backend

import (
	"bytes"
	"context"
	"encoding/binary"
	"log/slog"

	"github.com/dispel-re/dispel-multi/internal/database"
	"github.com/dispel-re/dispel-multi/model"
)

// 008-JP1-20001
func (b *Backend) HandleCreateNewAccount(session *model.Session, req CreateNewAccountRequest) error {
	data, err := req.Parse()
	if err != nil {
		return err
	}

	password, err := hashPassword(data.Password)
	if err != nil {
		slog.Warn("packet-42: could not hash the password", "err", err)
		return b.Send(session.Conn, CreateNewAccount, []byte{0, 0, 0, 0})
	}

	user, err := b.DB.CreateUser(context.TODO(), database.CreateUserParams{
		Username: data.Username,
		Password: password,
	})
	if err != nil {
		slog.Warn("packet-42: could not save new user into database", "err", err)
		return b.Send(session.Conn, CreateNewAccount, []byte{0, 0, 0, 0})
	}

	slog.Info("packet-42: new user created", "user", user.Username)

	return b.Send(session.Conn, CreateNewAccount, []byte{1, 0, 0, 0})
}

type CreateNewAccountRequest []byte

type CreateNewAccountRequestData struct {
	CDKey    uint32
	Username string
	Password string
	Unknown  []byte
}

func (r CreateNewAccountRequest) Parse() (data CreateNewAccountRequestData, err error) {
	data.CDKey = binary.LittleEndian.Uint32(r[0:4])
	split := bytes.SplitN(r[4:], []byte{0}, 3)
	data.Password = string(split[0])
	data.Username = string(split[1])
	data.Unknown = split[2]
	return data, err
}

type CreateNewAccountResponse [4]byte
