package backend

import "github.com/dispel-re/dispel-multi/model"

// HandleAuthorizationHandshake handles 0x6ff (255-6) command
func (b *Backend) HandleAuthorizationHandshake(session *model.Session, req HandleAuthorizationHandshakeRequest) error {
	response := append([]byte("ENET"), 0)
	return b.Send(session.Conn, AuthorizationHandshake, response)
}

type HandleAuthorizationHandshakeRequest []byte
