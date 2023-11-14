package backend

import (
	"context"
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

	response := []byte{}
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

// String of the room name
// /dispatcher.go:261 msg="Sent packet" packetType=69 data="\xffE\b\x00\x00\x00\x00\x00"
// /dispatcher.go:151 msg="Handle packet" packetType=69 packet="\xffE\x06\x00\x01\x00"
// /dispatcher.go:261 msg="Sent packet" packetType=69 data="\xffE\b\x00\x00\x00\x00\x00"
// /dispatcher.go:151 msg="Handle packet" packetType=34 packet="\xff\"\a\x00\x01\x00\x19"
// /dispatcher.go:261 msg="Sent packet" packetType=34 data="\xff\"\b\x00\x00\x00\x00\x00"

// /dispatcher.go:264 msg="Sent packet" packetType=69 data="/0UIAAAAAAA="
// /dispatcher.go:152 msg="Handle packet" packetType=69 packet=/0UGAAEA
// /dispatcher.go:264 msg="Sent packet" packetType=69 data="/0UIAAAAAAA="
// /dispatcher.go:152 msg="Handle packet" packetType=34 packet="/yIHAAEAGQ=="
// /dispatcher.go:264 msg="Sent packet" packetType=34 data="/yIIAAAAAAA="
// /dispatcher.go:152 msg="Handle packet" packetType=12 packet=/wwSAERJU1BFTABESVNQRUwA
// /dispatcher.go:152 msg="Handle packet" packetType=21 packet="/xUIAG9gDDE="
