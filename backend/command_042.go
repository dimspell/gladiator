package backend

import (
	"bytes"
	"encoding/binary"

	"github.com/dispel-re/dispel-multi/model"
)

func (b *Backend) HandleCreateNewAccount(session *model.Session, req CreateNewAccountRequest) error {
	resp := make([]byte, 4)
	ok := true
	if ok {
		resp[0] = 1
	}
	return b.Send(session.Conn, CreateNewAccount, resp)
}

type CreateNewAccountRequest []byte

func (r CreateNewAccountRequest) Parse() (cdKey uint32, username string, password string, unknown []byte, err error) {
	cdKey = binary.LittleEndian.Uint32(r[0:4])
	split := bytes.SplitN(r[4:], []byte{0}, 3)
	password = string(split[0])
	username = string(split[1])
	unknown = split[2]
	return cdKey, username, password, unknown, err
}

type CreateNewAccountResponse [4]byte
