package backend

import (
	"context"
	"encoding/binary"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/dimspell/gladiator/backend/packet"
	multiv1 "github.com/dimspell/gladiator/gen/multi/v1"
	"github.com/dimspell/gladiator/model"
)

// HandleSelectGame handles 0x45ff (255-69) command
func (b *Backend) HandleSelectGame(session *model.Session, req SelectGameRequest) error {
	if session.UserID == 0 {
		return fmt.Errorf("packet-69: user is not logged in")
	}

	data, err := req.Parse()
	if err != nil {
		slog.Warn("Invalid packet", "error", err)
		return nil
	}

	respGame, err := b.gameClient.GetGame(context.TODO(), connect.NewRequest(&multiv1.GetGameRequest{
		GameName: data.RoomName,
	}))
	if err != nil {
		slog.Warn("No game found", "room", data.RoomName, "error", err)
		return nil
	}

	respPlayers, err := b.gameClient.ListPlayers(context.TODO(), connect.NewRequest(&multiv1.ListPlayersRequest{
		GameRoomId: respGame.Msg.Game.GameId,
	}))
	if err != nil {
		slog.Error("Cannot list players", "err", err.Error())
		return nil
	}

	hostIP, err := b.Proxy.Join(
		respGame.Msg.GetGame().GetName(),
		session.Username,
		session.Username,
		respGame.Msg.GetGame().HostIpAddress,
	)
	if err != nil {
		return err
	}

	gameRoom := SelectGameResponse{
		Lobby: model.LobbyRoom{
			HostIPAddress: hostIP.To4(),
			Name:          respGame.Msg.Game.Name,
			Password:      "",
		},
		MapID:   uint32(respGame.Msg.Game.GetMapId()),
		Players: []model.LobbyPlayer{},
	}

	response := []byte{}
	response = binary.LittleEndian.AppendUint32(response, gameRoom.MapID)

	for _, player := range respPlayers.Msg.GetPlayers() {
		if player.UserId == session.UserID {
			continue
		}

		proxyIP, err := b.Proxy.Exchange(
			respGame.Msg.GetGame().String(),
			player.Username,
			player.IpAddress,
		)
		if err != nil {
			return err
		}

		// TODO: make sure the host is the first one
		// lobbyPlayer := model.LobbyPlayer{
		//	ClassType: model.ClassType(player.ClassType),
		//	Name:      player.Username,
		//	IPAddress: proxyIP.To4(),
		// }
		// gameRoom.Players = append(gameRoom.Players, lobbyPlayer)

		response = append(response, byte(player.ClassType), 0, 0, 0) // Class type (4 bytes)
		response = append(response, proxyIP.To4()[:]...)             // IP Address (4 bytes)
		response = append(response, player.Username...)              // Player name (null terminated string)
		response = append(response, byte(0))                         // Null byte
	}

	return b.Send(session.Conn, SelectGame, response)
}

type SelectGameRequest []byte

type SelectGameRequestData struct {
	RoomName string
}

func (r SelectGameRequest) Parse() (data SelectGameRequestData, err error) {
	rd := packet.NewReader(r)
	data.RoomName, err = rd.ReadString()
	if err != nil {
		return data, fmt.Errorf("packet-69: cannot read room name: %w", err)
	}
	return data, rd.Close()
}

type SelectGameResponse struct {
	Lobby   model.LobbyRoom
	MapID   uint32
	Players []model.LobbyPlayer
}
