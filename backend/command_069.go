package backend

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/binary"
	"errors"
	"fmt"
	"log/slog"

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

	respGame, err := b.gameClient.GetGame(context.TODO(),
		connect.NewRequest(&multiv1.GetGameRequest{
			UserId:   session.UserID,
			GameName: data.RoomName,
		}))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			fmt.Println("packet-69: game room does not exist")
			return nil
		}
		return err
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
		lobbyPlayer := model.LobbyPlayer{
			ClassType: model.ClassType(player.ClassType),
			Name:      player.Username,
			IPAddress: proxyIP.To4(),
		}
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
