package backend

import (
	"bytes"
	"fmt"

	"github.com/dispel-re/dispel-multi/model"
)

// HandleClientHostAndUsername handles 0x1eff (255-30) command
func (b *Backend) HandleClientHostAndUsername(session *model.Session, req ClientHostAndUsernameRequest) error {
	return b.Send(session.Conn, ClientHostAndUsername, []byte{1, 0, 0, 0})
}

type ClientHostAndUsernameRequest []byte

type ClientHostAndUsernameRequestData struct {
	ComputerHostname string
	ComputerUsername string
}

func (r ClientHostAndUsernameRequest) Parse() (data ClientHostAndUsernameRequestData, err error) {
	if bytes.Count(r, []byte{0}) < 2 {
		return data, fmt.Errorf("packet-30: not enough null terminators")
	}

	split := bytes.SplitN(r, []byte{0}, 3)
	data.ComputerHostname = string(split[0])
	data.ComputerUsername = string(split[1])
	return data, nil
}
