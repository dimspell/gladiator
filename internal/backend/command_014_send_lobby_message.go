package backend

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/dimspell/gladiator/internal/backend/bsession"
)

func (b *Backend) HandleSendLobbyMessage(session *bsession.Session, req SendLobbyMessageRequest) error {
	message, err := req.Parse()
	if err != nil {
		return fmt.Errorf("packet-14: could not parse request: %w", err)
	}
	if len(message) == 0 || len(message) > 87 {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.TODO(), 2*time.Second)
	defer cancel()

	if err := session.SendChatMessage(ctx, message); err != nil {
		slog.Warn("Could not send WS message", "error", fmt.Errorf("packet-14: could not send chat message: %w", err))
	}

	// resp := NewGlobalMessage(session.Username, message)
	return nil // session.Send(ReceiveMessage, resp)
}

type SendLobbyMessageRequest []byte

func (c SendLobbyMessageRequest) Parse() (message string, err error) {
	split := bytes.SplitN(c, []byte{0}, 2)
	if len(split) != 2 {
		return "", fmt.Errorf("malformed packet, missing null terminator")
	}
	return string(split[0]), nil
}
