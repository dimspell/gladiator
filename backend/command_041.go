package backend

import (
	"bytes"
	"encoding/binary"

	"github.com/dispel-re/dispel-multi/model"
)

func (b *Backend) HandleClientAuthentication(session *model.Session, req ClientAuthenticationRequest) error {
	resp := make([]byte, 4)
	ok := true
	if ok {
		resp[0] = 1
	}
	return b.Send(session.Conn, ClientAuthentication, resp)
}

type ClientAuthenticationRequest []byte

func (r ClientAuthenticationRequest) Parse() (unknown uint32, username string, password string, err error) {
	unknown = binary.LittleEndian.Uint32(r[0:4])

	split := bytes.SplitN(r[4:], []byte{0}, 3)
	password = string(split[0])
	username = string(split[1])

	return unknown, username, password, nil
}
