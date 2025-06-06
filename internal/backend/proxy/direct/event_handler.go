package direct

import (
	"context"
	"log/slog"
	"net"

	"github.com/dimspell/gladiator/internal/backend/bsession"
	"github.com/dimspell/gladiator/internal/backend/packet"
	"github.com/dimspell/gladiator/internal/wire"
)

type LanMessageHandler struct {
	session   *bsession.Session
	BySession map[*bsession.Session]*GameRoom
}

func (l *LanMessageHandler) Handle(ctx context.Context, payload []byte) error {
	et := wire.ParseEventType(payload)

	switch et {
	case wire.JoinRoom:
		_, msg, err := wire.DecodeTyped[wire.Player](payload)
		if err != nil {
			return nil
		}

		player := msg.Content
		slog.Info("Other player is joining", "playerId", player.ID())

		gameRoom, found := l.BySession[l.session]
		if !found {
			return nil
		}

		gameRoom.SetPlayer(player)
	case wire.LeaveRoom, wire.LeaveLobby:
		_, msg, err := wire.DecodeTyped[wire.Player](payload)
		if err != nil {
			return nil
		}

		player := msg.Content
		slog.Info("Other player is leaving", "playerId", player.ID())

		gameRoom, found := l.BySession[l.session]
		if !found {
			return nil
		}

		gameRoom.DeletePlayer(player.UserID)
	case wire.HostMigration:
		_, msg, err := wire.DecodeTyped[wire.Player](payload)
		if err != nil {
			return nil
		}

		ip := net.ParseIP(msg.Content.IPAddress)
		if ip == nil {
			slog.Error("Failed to parse IP address", "ip", msg.Content.IPAddress)
			return nil
		}

		response := make([]byte, 8)
		copy(response[0:4], []byte{1, 0, 0, 0})
		copy(response[4:], ip.To4())

		if err := l.session.SendFromBackend(packet.HostMigration, response); err != nil {
			slog.Error("Failed to send host migration response", "error", err)
			return nil
		}
	default:
		//	Ignore
	}

	return nil
}
