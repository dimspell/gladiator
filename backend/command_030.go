package backend

import (
	"bytes"
	"fmt"

	"github.com/dispel-re/dispel-multi/model"
)

// HandleClientHostAndUsername handles 0x1eff (255-30) command
func (b *Backend) HandleClientHostAndUsername(session *model.Session, req ClientHostAndUsernameRequest) error {
	return b.Send(session.Conn, ClientHostAndUsername, []byte{1, 0})
}

type ClientHostAndUsernameRequest []byte

func (r ClientHostAndUsernameRequest) Parse() (hostName string, hostUser string, err error) {
	if bytes.Count(r, []byte{0}) < 2 {
		return "", "", fmt.Errorf("packet-30: not enough null terminators")
	}

	split := bytes.SplitN(r, []byte{0}, 3)
	hostName = string(split[0])
	hostUser = string(split[1])
	return hostName, hostUser, nil
}
