package backend

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/dispel-re/dispel-multi/model"
)

// HandleSelectGame handles 0x45ff (255-69) command
func (b *Backend) HandleSelectGame(session *model.Session, req SelectGameRequest) error {
	if session.UserID == 0 {
		return fmt.Errorf("packet-69: user is not logged in")
	}

	// data, err := req.Parse()
	// if err != nil {
	// 	return err
	// }

	// respGame, err := b.GameClient.GetGame(context.TODO(),
	// 	connect.NewRequest(&multiv1.GetGameRequest{
	// 		UserId:   session.UserID,
	// 		GameName: data.RoomName,
	// 	}))
	// if err != nil {
	// 	return err
	// }

	gameRoom := SelectGameResponse{
		Lobby: model.LobbyRoom{
			HostIPAddress: [4]byte{192, 168, 121, HostIP},
			// Name:          respGame.Msg.Game.Name,
			Name:     GameRoomName,
			Password: "",
		},
		// MapID: uint32(respGame.Msg.Game.GetMapId()),
		MapID: 2,
		Players: []model.LobbyPlayer{
			{
				ClassType: model.ClassType(model.ClassTypeKnight),
				Name:      "archer",
				IPAddress: [4]byte{192, 168, 121, HostIP},
			},
			// {
			// 	ClassType: model.ClassType(model.ClassTypeMage),
			// 	Name:      "mage",
			// 	IPAddress: [4]byte{192, 168, 121, ClientIP},
			// },
		},
	}

	// gameRoom := model.GameRoom{
	// 	Lobby: model.LobbyRoom{
	// 		HostIPAddress: [4]byte{},
	// 		Name:          respGame.Msg.Game.Name,
	// 		Password:      "",
	// 	},
	// 	MapID: uint32(respGame.Msg.Game.MapId),
	// }
	// copy(gameRoom.Lobby.HostIPAddress[:], net.ParseIP(respGame.Msg.Game.HostIpAddress).To4())

	// respPlayers, err := b.GameClient.ListPlayers(context.TODO(),
	// 	connect.NewRequest(&multiv1.ListPlayersRequest{
	// 		GameRoomId: respGame.Msg.Game.GameId,
	// 	}))
	// if err != nil {
	// 	return err
	// }
	// for _, player := range respPlayers.Msg.GetPlayers() {
	// 	lobbyPlayer := model.LobbyPlayer{
	// 		ClassType: model.ClassType(player.ClassType),
	// 		Name:      player.Username,
	// 	}
	// 	copy(lobbyPlayer.IPAddress[:], net.ParseIP(player.IpAddress).To4())
	// 	gameRoom.Players = append(gameRoom.Players, lobbyPlayer)
	// }

	return b.Send(session.Conn, SelectGame, gameRoom.Details())
}

type SelectGameRequest []byte

type SelectGameRequestData struct {
	RoomName string
}

func (r SelectGameRequest) Parse() (data SelectGameRequestData, err error) {
	split := bytes.Split(r, []byte{0})
	data.RoomName = string(bytes.TrimSuffix(split[0], []byte{0}))

	return data, nil
}

type SelectGameResponse struct {
	Lobby   model.LobbyRoom
	MapID   uint32
	Players []model.LobbyPlayer
}

func (r *SelectGameResponse) Details() []byte {
	buf := []byte{}
	buf = binary.LittleEndian.AppendUint32(buf, r.MapID)
	for _, player := range r.Players {
		buf = append(buf, player.ToBytes()...)
	}
	return buf
}
