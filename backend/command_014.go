package backend

import (
	"bytes"
	"fmt"

	"github.com/dispel-re/dispel-multi/model"
)

func (b *Backend) HandleSendLobbyMessage(session *model.Session, req SendLobbyMessageRequest) error {
	message, err := req.Parse()
	if err != nil {
		return fmt.Errorf("packet-14: could not parse request: %w", err)
	}
	if len(message) == 0 || len(message) > 87 {
		return nil
	}
	resp := NewLobbyMessage(session.Username, message)
	return b.Send(session.Conn, ReceiveMessage, resp)
}

type SendLobbyMessageRequest []byte

func (c SendLobbyMessageRequest) Parse() (message string, err error) {
	split := bytes.SplitN(c, []byte{0}, 2)
	if len(split) != 2 {
		return "", fmt.Errorf("malformed packet, missing null terminator")
	}
	return string(split[0]), nil
}
