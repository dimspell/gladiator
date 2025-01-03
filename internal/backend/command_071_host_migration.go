package backend

import (
	"github.com/dimspell/gladiator/internal/backend/bsession"
	"github.com/dimspell/gladiator/internal/backend/packet/command"
)

func (b *Backend) SendHostMigration(session *bsession.Session, isHost bool, newHostIP [4]byte) error {
	payload := []byte{}

	// Yes(int32 1)/No(int32 0)
	if isHost {
		payload = append(payload, 1, 0, 0, 0)
	} else {
		payload = append(payload, 0, 0, 0, 0)
	}

	// IP address in 4 bytes
	payload = append(payload, newHostIP[:]...)

	return session.Send(command.ChangeHost, payload)
}
