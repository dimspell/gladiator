package backend

import (
	"context"
	"encoding/binary"
	"fmt"
	"log/slog"
	"net"

	"github.com/dispel-re/dispel-multi/model"
)

// HandleListGames handles 0x9ff (255-9) command
func (b *Backend) HandleListGames(session *model.Session, req ListGamesRequest) error {
	if session.UserID == 0 {
		return fmt.Errorf("packet-09: user is not logged in")
	}

	gameRooms, err := b.DB.ListGameRooms(context.TODO())
	if err != nil {
		slog.Error("packet-09: could not list game rooms")
		return nil
	}

	var response []byte
	response = binary.LittleEndian.AppendUint32(response, uint32(len(gameRooms)))

	for _, room := range gameRooms {
		lobby := model.LobbyRoom{
			Name:     room.Name,
			Password: room.Password.String,
		}
		copy(lobby.HostIPAddress[:], net.ParseIP(room.HostIpAddress).To4())
		response = append(response, lobby.ToBytes()...)
	}
	return b.Send(session.Conn, ListGames, response)
}

type ListGamesRequest []byte
