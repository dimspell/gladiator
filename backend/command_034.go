package backend

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"log/slog"

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

	respGame, err := b.gameClient.GetGame(context.TODO(), connect.NewRequest(&multiv1.GetGameRequest{
		UserId:   session.UserID,
		GameName: data.RoomName,
	}))
	if err != nil {
		return err
	}

	hostIP, err := b.Proxy.Join(
		respGame.Msg.GetGame().GetName(),
		session.Username,
		session.Username,
		respGame.Msg.GetGame().HostIpAddress,
	)
	if err != nil {
		slog.Error("Cannot get proxy address", "error", err)
		return nil
	}

	_, err = b.gameClient.JoinGame(context.TODO(), connect.NewRequest(&multiv1.JoinGameRequest{
		UserId:      session.UserID,
		CharacterId: session.CharacterID,
		GameRoomId:  respGame.Msg.Game.GetGameId(),
		IpAddress:   session.LocalIpAddress,
	}))
	if err != nil {
		slog.Error("Could not join game room", "error", err)
		return nil
	}

	respPlayers, err := b.gameClient.ListPlayers(context.TODO(), connect.NewRequest(&multiv1.ListPlayersRequest{
		GameRoomId: respGame.Msg.Game.GameId,
	}))
	if err != nil {
		slog.Error("Cannot list players", "error", err)
		return nil
	}

	gameRoom := JoinGameResponse{
		Lobby: model.LobbyRoom{
			HostIPAddress: hostIP.To4(),
			Name:          respGame.Msg.Game.Name,
			Password:      "",
		},
		MapID:   uint16(respGame.Msg.Game.GetMapId()),
		Players: []model.LobbyPlayer{},
	}

	for _, player := range respPlayers.Msg.GetPlayers() {
		proxyIP, err := b.Proxy.Exchange(respGame.Msg.GetGame().GetName(), player.Username, player.IpAddress)
		if err != nil {
			return err
		}
		if bytes.Equal(proxyIP, []byte{0, 0, 0, 0}) {
			return fmt.Errorf("packet-34: incorrect proxy for %v", player.IpAddress)
		}

		// TODO: make sure the host is the first one
		lobbyPlayer := model.LobbyPlayer{
			ClassType: model.ClassType(player.ClassType),
			Name:      player.Username,
			IPAddress: proxyIP.To4(),
		}
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
	rd := NewPacketReader(r)

	data.RoomName, err = rd.ReadString()
	if err != nil {
		return data, fmt.Errorf("packet-34: could not read room name: %w", err)
	}

	// TODO: Read password if given

	// TODO: 216 byte at the end of the packet

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
