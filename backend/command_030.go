package backend

import "github.com/dispel-re/dispel-multi/model"

// HandleClientHostAndUsername handles 0x1eff (255-30) command
func (b *Backend) HandleClientHostAndUsername(session *model.Session, req ClientHostAndUsernameRequest) error {
	return b.Send(session.Conn, ClientHostAndUsername, []byte{1, 0})
}

type ClientHostAndUsernameRequest []byte
