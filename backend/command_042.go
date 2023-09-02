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

func (r CreateNewAccountRequest) CDKey() uint32 {
	return binary.LittleEndian.Uint32(r[0:4])
}

func (r CreateNewAccountRequest) UsernameAndPassword() (username string, password string) {
	split := bytes.SplitN(r[4:], []byte{0}, 3)
	password = string(split[0])
	username = string(split[1])
	return username, password
}

func (r CreateNewAccountRequest) Unknown() []byte {
	split := bytes.SplitN(r[4:], []byte{0}, 3)
	return split[2]
}

type CreateNewAccountResponse [4]byte
