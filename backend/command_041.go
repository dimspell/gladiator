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

func (b *Backend) HandleClientAuthentication(session *model.Session, req ClientAuthenticationRequest) error {
	if session.UserID != 0 {
		return fmt.Errorf("packet-41: user has been already logged in")
	}

	data, err := req.Parse()
	if err != nil {
		slog.Warn("Invalid packet", "error", err)
		return nil
	}

	user, err := b.userClient.AuthenticateUser(context.TODO(), connect.NewRequest(&multiv1.AuthenticateUserRequest{
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
	rd := packet.NewReader(r)

	data.Unknown, err = rd.ReadUint32()
	if err != nil {
		return data, fmt.Errorf("packet-41: malformed unknown: %w", err)
	}
	data.Password, err = rd.ReadString()
	if err != nil {
		return data, fmt.Errorf("packet-41: malformed password: %w", err)
	}
	data.Username, err = rd.ReadString()
	if err != nil {
		return data, fmt.Errorf("packet-41: malformed username: %w", err)
	}

	return data, rd.Close()
}
