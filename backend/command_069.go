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

// HandleSelectGame handles 0x45ff (255-69) command
func (b *Backend) HandleSelectGame(session *model.Session, req SelectGameRequest) error {
	if session.UserID == 0 {
		return fmt.Errorf("packet-69: user is not logged in")
	}

	data, err := req.Parse()
	if err != nil {
		return err
	}

	respRoom, err := b.GameClient.GetGame(context.TODO(),
		connect.NewRequest(&multiv1.GetGameRequest{
			UserId:   session.UserID,
			GameName: data.RoomName,
		}))
	if err != nil {
		return err
	}
	gameRoom := model.GameRoom{
		Lobby: model.LobbyRoom{
			HostIPAddress: [4]byte{},
			Name:          respRoom.Msg.Game.Name,
			Password:      respRoom.Msg.Game.Password,
		},
		MapID: uint32(respRoom.Msg.Game.MapId),
	}
	copy(gameRoom.Lobby.HostIPAddress[:], net.ParseIP(respRoom.Msg.Game.HostIpAddress).To4())

	respPlayers, err := b.GameClient.ListPlayers(context.TODO(),
		connect.NewRequest(&multiv1.ListPlayersRequest{
			GameRoomId: respRoom.Msg.Game.GameId,
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
