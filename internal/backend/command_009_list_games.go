package backend

import (
	"context"
	"encoding/binary"
	"fmt"
	"log/slog"
	"net"

	"connectrpc.com/connect"
	multiv1 "github.com/dimspell/gladiator/gen/multi/v1"
	"github.com/dimspell/gladiator/internal/backend/bsession"
	"github.com/dimspell/gladiator/internal/backend/packet"
	"github.com/dimspell/gladiator/internal/model"
)

// HandleListGames handles 0x9ff (255-9) command
func (b *Backend) HandleListGames(ctx context.Context, session *bsession.Session, req ListGamesRequest) error {
	if session.UserID == 0 {
		return fmt.Errorf("packet-09: user is not logged in")
	}

	resp, err := b.gameClient.ListGames(ctx, connect.NewRequest(&multiv1.ListGamesRequest{}))
	if err != nil {
		slog.Error("packet-09: could not list game rooms")
		return nil
	}

	var response []byte
	response = binary.LittleEndian.AppendUint32(response, uint32(len(resp.Msg.GetGames())))

	for _, room := range resp.Msg.GetGames() {
		roomIP := net.ParseIP(room.HostIpAddress)
		if roomIP == nil {
			slog.Debug("packet-09: could not parse room ip address", "ip", room.HostIpAddress)
		}

		lobby := model.LobbyRoom{
			Name:          room.Name,
			Password:      room.Password,
			HostIPAddress: session.Proxy.GetHostIP(roomIP).To4(),
		}

		// response = append(response, lobby.ToBytes()...)

		response = append(response, lobby.HostIPAddress[:]...) // Host IP Address (4 bytes)
		response = append(response, lobby.Name...)             // Room name (null terminated string)
		response = append(response, byte(0))                   // Null byte
		response = append(response, lobby.Password...)         // Room password (null terminated string)
		response = append(response, byte(0))                   // Null byte
	}

	return session.SendToGame(packet.ListGames, response)
}

type ListGamesRequest []byte
