package backend

import (
	"encoding/binary"

	"github.com/dispel-re/dispel-multi/model"
)

// HandleAuthorizationHandshake handles 0x6ff (255-6) command
func (b *Backend) HandleAuthorizationHandshake(session *model.Session, req AuthorizationHandshakeRequest) error {
	authKey, _, err := req.Parse()
	if err != nil {
		return err
	}
	if authKey != "68XIPSID" {
		return b.Send(session.Conn, AuthorizationHandshake, []byte{0, 0, 0, 0})
	}

	response := append([]byte("ENET"), 0)
	return b.Send(session.Conn, AuthorizationHandshake, response)
}

type AuthorizationHandshakeRequest []byte

func (r AuthorizationHandshakeRequest) Parse() (authKey string, unknown uint32, err error) {
	authKey = string(r[:8])
	unknown = binary.LittleEndian.Uint32(r[8:12])
	return authKey, unknown, err
}
