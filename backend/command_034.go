package backend

import (
	"bytes"
	"context"
	"fmt"
	"net"

	"connectrpc.com/connect"
	multiv1 "github.com/dispel-re/dispel-multi/gen/multi/v1"
	"github.com/dispel-re/dispel-multi/model"
)

// HandleJoinGame handles 0x22ff (255-34) command
func (b *Backend) HandleJoinGame(session *model.Session, req JoinGameRequest) error {
	if session.UserID == 0 {
		return fmt.Errorf("packet-34: user is not logged in")
	}

	data, err := req.Parse()
	if err != nil {
		return err
	}

	respGame, err := b.GameClient.GetGame(context.TODO(), connect.NewRequest(&multiv1.GetGameRequest{
		UserId:   session.UserID,
		GameName: data.RoomName,
	}))
	if err != nil {
		return err
	}

	tcpAddr := session.Conn.RemoteAddr().(*net.TCPAddr)
	_, err = b.GameClient.JoinGame(context.TODO(), connect.NewRequest(&multiv1.JoinGameRequest{
		UserId:      session.UserID,
		CharacterId: session.CharacterID,
		GameRoomId:  respGame.Msg.Game.GameId,
		IpAddress:   tcpAddr.IP.To4().String(),
	}))
	if err != nil {
		return err
	}

	gameRoom := model.GameRoom{
		Lobby: model.LobbyRoom{
			HostIPAddress: [4]byte{},
			Name:          respGame.Msg.Game.Name,
			Password:      respGame.Msg.Game.Password,
		},
		MapID: uint32(respGame.Msg.Game.GetMapId()),
	}
	copy(gameRoom.Lobby.HostIPAddress[:], net.ParseIP(respGame.Msg.Game.HostIpAddress).To4())

	respPlayers, err := b.GameClient.ListPlayers(context.TODO(), connect.NewRequest(&multiv1.ListPlayersRequest{
		GameRoomId: respGame.Msg.Game.GameId,
	}))
	if err != nil {
		return err
	}
	for _, player := range respPlayers.Msg.GetPlayers() {
		lobbyPlayer := model.LobbyPlayer{
			ClassType: model.ClassType(player.ClassType),
			Name:      player.CharacterName,
		}
		copy(lobbyPlayer.IPAddress[:], net.ParseIP(player.IpAddress).To4())
		gameRoom.Players = append(gameRoom.Players, lobbyPlayer)
	}

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
