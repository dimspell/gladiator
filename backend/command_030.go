package backend

import (
	"fmt"

	"github.com/dimspell/gladiator/backend/packet"
)

// HandleClientHostAndUsername handles 0x1eff (255-30) command
func (b *Backend) HandleClientHostAndUsername(session *Session, req ClientHostAndUsernameRequest) error {
	return b.Send(session.Conn, ClientHostAndUsername, []byte{1, 0, 0, 0})
}

type ClientHostAndUsernameRequest []byte

type ClientHostAndUsernameRequestData struct {
	ComputerHostname string
	ComputerUsername string
}

func (r ClientHostAndUsernameRequest) Parse() (data ClientHostAndUsernameRequestData, err error) {
	rd := packet.NewReader(r)

	data.ComputerHostname, err = rd.ReadString()
	if err != nil {
		return data, fmt.Errorf("packet-30: malformed hostname: %w", err)
	}
	data.ComputerUsername, err = rd.ReadString()
	if err != nil {
		return data, fmt.Errorf("packet-30: malformed computer user: %w", err)
	}

	return data, rd.Close()
}
