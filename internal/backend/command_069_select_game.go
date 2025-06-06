package backend

import (
	"context"
	"encoding/binary"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"
	multiv1 "github.com/dimspell/gladiator/gen/multi/v1"
	"github.com/dimspell/gladiator/internal/backend/bsession"
	"github.com/dimspell/gladiator/internal/backend/packet"
	"github.com/dimspell/gladiator/internal/backend/proxy"
)

// HandleSelectGame handles 0x45ff (255-69) command
func (b *Backend) HandleSelectGame(ctx context.Context, session *bsession.Session, req SelectGameRequest) error {
	if session.UserID == 0 {
		return fmt.Errorf("packet-69: user is not logged in")
	}

	data, err := req.Parse()
	if err != nil {
		slog.Warn("Invalid packet", "error", err)
		return nil
	}

	respGame, err := b.gameClient.GetGame(ctx, connect.NewRequest(&multiv1.GetGameRequest{
		GameRoomId: data.RoomName,
	}))
	if err != nil {
		slog.Warn("No game found", "room", data.RoomName, "error", err)
		return nil
	}

	if err := b.Proxy.SelectGame(proxy.GameData{
		Game:    respGame.Msg.GetGame(),
		Players: respGame.Msg.GetPlayers(),
	}, session); err != nil {
		return err
	}

	response := []byte{}
	response = binary.LittleEndian.AppendUint32(response, uint32(respGame.Msg.Game.GetMapId()))

	for _, player := range respGame.Msg.GetPlayers() {
		if player.UserId == session.UserID {
			continue
		}

		ps := proxy.GetPlayerAddrParams{
			GameID:     respGame.Msg.GetGame().GetName(),
			UserID:     player.UserId,
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

	return session.SendToGame(packet.SelectGame, response)
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
