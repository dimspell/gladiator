package backend

import (
	"context"
	"encoding/binary"
	"fmt"
	"log/slog"

	"github.com/dimspell/gladiator/internal/app/logger/logging"
	"github.com/dimspell/gladiator/internal/backend/bsession"
	"github.com/dimspell/gladiator/internal/backend/packet"
)

// HandleSelectGame handles 0x45ff (255-69) command
func (b *Backend) HandleSelectGame(ctx context.Context, session *bsession.Session, req SelectGameRequest) error {
	if session.UserID == 0 {
		return fmt.Errorf("packet-69: user is not logged in")
	}

	data, err := req.Parse()
	if err != nil {
		slog.Warn("Invalid packet", logging.Error(err))
		return nil
	}

	game, players, err := session.Proxy.GetGame(ctx, data.RoomName)
	if err != nil {
		return err
	}

	response := []byte{}
	response = binary.LittleEndian.AppendUint32(response, uint32(game.MapID))

	for _, player := range players {
		if player.Name == session.Username {
			continue
		}

		response = append(response, byte(player.ClassType), 0, 0, 0) // Class type (4 bytes)
		response = append(response, player.IPAddress.To4()[:]...)    // IP Address (4 bytes)
		response = append(response, player.Name...)                  // Player name (null terminated string)
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
