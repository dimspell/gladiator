package backend

import (
	"bytes"
	"fmt"

	"github.com/dispel-re/dispel-multi/model"
)

// HandleListGames handles 0x9ff (255-9) command
func (b *Backend) HandleListGames(session *model.Session, req ListGamesRequest) error {
	if session.UserID == 0 {
		return fmt.Errorf("packet-09: user is not logged in")
	}

	// resp, err := b.GameClient.ListGames(context.TODO(), connect.NewRequest(&multiv1.ListGamesRequest{}))
	// if err != nil {
	// 	slog.Error("packet-09: could not list game rooms")
	// 	return nil
	// }

	// var response []byte
	// response = binary.LittleEndian.AppendUint32(response, uint32(len(resp.Msg.GetGames())))

	// for _, room := range resp.Msg.GetGames() {
	// 	lobby := model.LobbyRoom{
	// 		Name: room.Name,
	// 		// Password: room.Password,
	// 		Password:      "",
	// 		HostIPAddress: [4]byte{192, 168, 121, 212},
	// 	}
	// 	// copy(lobby.HostIPAddress[:], net.ParseIP(room.HostIpAddress).To4())
	// 	response = append(response, lobby.ToBytes()...)
	// }

	buf := bytes.NewBuffer([]byte{1, 0, 0, 0})

	lobby := model.LobbyRoom{
		Name:          GameRoomName,
		Password:      "",
		HostIPAddress: [4]byte{192, 168, 121, HostIP},
	}
	buf.Write(lobby.ToBytes())
	response := buf.Bytes()

	return b.Send(session.Conn, ListGames, response)
}

type ListGamesRequest []byte
