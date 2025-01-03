package backend

import (
	"context"
	"encoding/binary"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"
	multiv1 "github.com/dimspell/gladiator/gen/multi/v1"
	"github.com/dimspell/gladiator/internal/backend/packet"
	"github.com/dimspell/gladiator/internal/wire"
)

// HandleSelectGame handles 0x45ff (255-69) command
func (b *Backend) HandleSelectGame(session *Session, req SelectGameRequest) error {
	if session.UserID == 0 {
		return fmt.Errorf("packet-69: user is not logged in")
	}

	data, err := req.Parse()
	if err != nil {
		slog.Warn("Invalid packet", "error", err)
		return nil
	}

	b.Proxy.Close(session)

	respGame, err := b.gameClient.GetGame(context.TODO(), connect.NewRequest(&multiv1.GetGameRequest{
		GameRoomId: data.RoomName,
	}))
	if err != nil {
		slog.Warn("No game found", "room", data.RoomName, "error", err)
		return nil
	}

	gameRoom := &GameRoom{
		Players: map[string]wire.Player{},
		ID:      respGame.Msg.GetGame().GetName(),
		Name:    respGame.Msg.GetGame().GetName(),
	}
	for _, player := range respGame.Msg.Players {
		gameRoom.SetPlayer(wire.Player{
			UserID:      player.UserId,
			Username:    player.Username,
			CharacterID: player.CharacterId,
			ClassType:   byte(player.ClassType),
			IPAddress:   player.IpAddress,
		})
		if respGame.Msg.Game.HostUserId == player.UserId {
			gameRoom.SetHost(wire.Player{
				UserID:      player.UserId,
				Username:    player.Username,
				CharacterID: player.CharacterId,
				ClassType:   byte(player.ClassType),
				IPAddress:   player.IpAddress,
			})
		}
	}
	session.State.SetGameRoom(gameRoom)

	response := []byte{}
	response = binary.LittleEndian.AppendUint32(response, uint32(respGame.Msg.Game.GetMapId()))

	for _, player := range respGame.Msg.GetPlayers() {
		if player.UserId == session.UserID {
			continue
		}

		ps := GetPlayerAddrParams{
			GameID:     respGame.Msg.GetGame().GetName(),
			UserID:     fmt.Sprintf("%d", player.UserId),
			IPAddress:  player.IpAddress,
			HostUserID: fmt.Sprintf("%d", respGame.Msg.GetGame().HostUserId),
		}
		proxyIP, err := b.Proxy.GetPlayerAddr(ps, session)

		if err != nil {
			slog.Warn("Not found a player with the provided ID",
				"player", player.Username,
				"proxyIP", proxyIP,
				"error", err,
				"gameID", ps.GameID,
				"userId", ps.UserID,
				"ipAddress", ps.IPAddress,
			)
			// return err
			// continue
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

	return session.Send(SelectGame, response)
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

//	gameRoom := SelectGameResponse{
//		Lobby: model.LobbyRoom{
//			HostIPAddress: hostIP.To4(),
//			Name:          respGame.Msg.Game.Name,
//			Password:      "",
//		},
//		MapID: uint32(respGame.Msg.Game.GetMapId()),
//		// Players: []model.LobbyPlayer{}, // Unused
//	}
// type SelectGameResponse struct {
//	Lobby   model.LobbyRoom
//	MapID   uint32
//	Players []model.LobbyPlayer
// }
