package backend

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/dispel-re/dispel-multi/model"
)

// HandleJoinGame handles 0x22ff (255-34) command
func (b *Backend) HandleJoinGame(session *model.Session, req JoinGameRequest) error {
	if session.UserID == 0 {
		return fmt.Errorf("packet-34: user is not logged in")
	}

	// data, err := req.Parse()
	// if err != nil {
	// 	return err
	// }

	// respGame, err := b.GameClient.GetGame(context.TODO(), connect.NewRequest(&multiv1.GetGameRequest{
	// 	UserId:   session.UserID,
	// 	GameName: data.RoomName,
	// }))
	// if err != nil {
	// 	return err
	// }

	// // clientIpAddress := session.Conn.RemoteAddr().(*net.TCPAddr).IP.To4().String()
	// clientIpAddress := "192.168.121.212" // 169
	// // clientIpAddress := "127.0.0.34"

	// _, err = b.GameClient.JoinGame(context.TODO(), connect.NewRequest(&multiv1.JoinGameRequest{
	// 	UserId:      session.UserID,
	// 	CharacterId: session.CharacterID,
	// 	GameRoomId:  respGame.Msg.Game.GameId,
	// 	IpAddress:   clientIpAddress,
	// }))
	// if err != nil {
	// 	return err
	// }

	gameRoom := JoinGameResponse{
		Lobby: model.LobbyRoom{
			HostIPAddress: [4]byte{192, 168, 121, HostIP},
			Name:          GameRoomName,
			// Name:          respGame.Msg.Game.Name,
			Password: "",
		},
		// MapID: uint32(respGame.Msg.Game.GetMapId()),
		MapID: 2,
		Players: []model.LobbyPlayer{
			{
				ClassType: model.ClassType(model.ClassTypeWarrior),
				Name:      "archer",
				IPAddress: [4]byte{192, 168, 121, HostIP},
			},
			{
				ClassType: model.ClassType(model.ClassTypeMage),
				Name:      "mage",
				IPAddress: [4]byte{192, 168, 121, ClientIP},
			},
		},
	}

	// gameRoom := model.GameRoom{
	// 	Lobby: model.LobbyRoom{
	// 		HostIPAddress: [4]byte{},
	// 		Name:          respGame.Msg.Game.Name,
	// 		Password:      "",
	// 	},
	// 	MapID: uint32(respGame.Msg.Game.GetMapId()),
	// }
	// copy(gameRoom.Lobby.HostIPAddress[:], net.ParseIP(respGame.Msg.Game.HostIpAddress).To4())
	// // gameRoom.Lobby.HostIPAddress = [4]byte{127, 21, 37, 28}

	// respPlayers, err := b.GameClient.ListPlayers(context.TODO(), connect.NewRequest(&multiv1.ListPlayersRequest{
	// 	GameRoomId: respGame.Msg.Game.GameId,
	// }))
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

	return b.Send(session.Conn, JoinGame, gameRoom.Details())
}

type JoinGameRequest []byte

type JoinGameRequestData struct {
	RoomName string
	Password string
}

func (r JoinGameRequest) Parse() (data JoinGameRequestData, err error) {
	split := bytes.Split(r, []byte{0})

	data.RoomName = string(split[0])
	data.Password = string(bytes.TrimSuffix(split[1], []byte{0}))

	return data, nil
}

type JoinGameResponse struct {
	Lobby   model.LobbyRoom
	MapID   uint16
	Players []model.LobbyPlayer
}

func (r *JoinGameResponse) Details() []byte {
	buf := []byte{}
	buf = binary.LittleEndian.AppendUint16(buf, r.MapID)
	for _, player := range r.Players {
		buf = append(buf, player.ToBytes()...)
	}
	return buf
}
