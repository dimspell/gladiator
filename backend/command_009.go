package backend

import (
	"context"
	"encoding/binary"
	"fmt"
	"log/slog"
	"net"

	"connectrpc.com/connect"
	multiv1 "github.com/dispel-re/dispel-multi/gen/multi/v1"
	"github.com/dispel-re/dispel-multi/model"
)

// HandleListGames handles 0x9ff (255-9) command
func (b *Backend) HandleListGames(session *model.Session, req ListGamesRequest) error {
	if session.UserID == 0 {
		return fmt.Errorf("packet-09: user is not logged in")
	}

	resp, err := b.GameClient.ListGames(context.TODO(), connect.NewRequest(&multiv1.ListGamesRequest{}))
	if err != nil {
		slog.Error("packet-09: could not list game rooms")
		return nil
	}

	var response []byte
	response = binary.LittleEndian.AppendUint32(response, uint32(len(resp.Msg.GetGames())))

	for _, room := range resp.Msg.GetGames() {
		lobby := model.LobbyRoom{
			Name:     room.Name,
			Password: room.Password,
		}
		copy(lobby.HostIPAddress[:], net.ParseIP(room.HostIpAddress).To4())
		response = append(response, lobby.ToBytes()...)
	}
	return b.Send(session.Conn, ListGames, response)
}

type ListGamesRequest []byte
