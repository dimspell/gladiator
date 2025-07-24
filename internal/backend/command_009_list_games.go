package backend

import (
	"context"
	"encoding/binary"
	"fmt"
	"log/slog"

	"github.com/dimspell/gladiator/internal/backend/bsession"
	"github.com/dimspell/gladiator/internal/backend/packet"
)

// HandleListGames handles 0x9ff (255-9) command
func (b *Backend) HandleListGames(ctx context.Context, session *bsession.Session, req ListGamesRequest) error {
	if session.UserID == 0 {
		return fmt.Errorf("packet-09: user is not logged in")
	}

	games, err := session.Proxy.ListGames(ctx)
	if err != nil {
		slog.Error("packet-09: could not list game rooms")
		return nil
	}

	var response []byte
	response = binary.LittleEndian.AppendUint32(response, uint32(len(games)))

	for _, lobby := range games {
		response = append(response, lobby.HostIPAddress[:]...) // Host IP Address (4 bytes)
		response = append(response, lobby.Name...)             // Room name (null terminated string)
		response = append(response, byte(0))                   // Null byte
		response = append(response, lobby.Password...)         // Room password (null terminated string)
		response = append(response, byte(0))                   // Null byte
	}

	return session.SendToGame(packet.ListGames, response)
}

type ListGamesRequest []byte
