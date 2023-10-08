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
