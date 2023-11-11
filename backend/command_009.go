package backend

import (
	"context"
	"log/slog"
	"net"

	"github.com/dispel-re/dispel-multi/model"
)

// HandleListGames handles 0x9ff (255-9) command
func (b *Backend) HandleListGames(session *model.Session, req ListGamesRequest) error {
	response := []byte{}

	gameRooms, err := b.DB.ListGameRooms(context.TODO())
	if err != nil {
		slog.Error("packet-09: could not list game rooms")
		return nil
	}

	for _, room := range gameRooms {
		lobby := model.LobbyRoom{
			Name:     room.Name,
			Password: room.Password.String,
		}
		copy(lobby.HostIPAddress[:], net.ParseIP(room.HostIpAddress))

		response = append(response, lobby.ToBytes()...)
	}
	return b.Send(session.Conn, ListGames, response)
}

type ListGamesRequest []byte
