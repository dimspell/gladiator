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
