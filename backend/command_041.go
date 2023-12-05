package backend

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"
	multiv1 "github.com/dispel-re/dispel-multi/gen/multi/v1"
	"github.com/dispel-re/dispel-multi/model"
)

func (b *Backend) HandleClientAuthentication(session *model.Session, req ClientAuthenticationRequest) error {
	if session.UserID != 0 {
		return fmt.Errorf("packet-41: user has been already logged in")
	}

	data, err := req.Parse()
	if err != nil {
		return err
	}

	user, err := b.UserClient.AuthenticateUser(context.TODO(), connect.NewRequest(&multiv1.AuthenticateUserRequest{
		Username: data.Username,
		Password: data.Password,
	}))
	if err != nil {
		slog.Debug("packet-41: could not sign in", "err", err)
		return b.Send(session.Conn, ClientAuthentication, []byte{0, 0, 0, 0})
	}

	// Assign user into session
	session.UserID = user.Msg.User.UserId
	session.Username = user.Msg.User.Username

	return b.Send(session.Conn, ClientAuthentication, []byte{1, 0, 0, 0})
}

type ClientAuthenticationRequest []byte

type ClientAuthenticationRequestData struct {
	Unknown  uint32
	Username string
	Password string
}

func (r ClientAuthenticationRequest) Parse() (data ClientAuthenticationRequestData, err error) {
	data.Unknown = binary.LittleEndian.Uint32(r[0:4])

	split := bytes.SplitN(r[4:], []byte{0}, 3)
	data.Password = string(split[0])
	data.Username = string(split[1])

	return data, nil
}
