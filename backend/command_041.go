package backend

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"log/slog"

	"github.com/dispel-re/dispel-multi/model"
	"golang.org/x/crypto/bcrypt"
)

func (b *Backend) HandleClientAuthentication(session *model.Session, req ClientAuthenticationRequest) error {
	if session.UserID != 0 {
		return fmt.Errorf("packet-41: user has been already logged in")
	}

	data, err := req.Parse()
	if err != nil {
		return err
	}

	user, err := b.DB.GetUser(context.TODO(), data.Username)
	if err != nil {
		slog.Debug("packet-41: could not find a user", "username", data.Username)
		return b.Send(session.Conn, ClientAuthentication, []byte{0, 0, 0, 0})
	}

	if !checkPasswordHash(data.Password, user.Password) {
		slog.Debug("packet-41: incorrect password")
		return b.Send(session.Conn, ClientAuthentication, []byte{0, 0, 0, 0})
	}

	// Assign user into session
	session.UserID = user.ID
	session.Username = user.Username

	return b.Send(session.Conn, ClientAuthentication, []byte{1, 0, 0, 0})
}

// TODO: Use salt and pepper
func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func checkPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
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
