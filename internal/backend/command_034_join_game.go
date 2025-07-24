package backend

import (
	"context"
	"encoding/binary"
	"fmt"
	"log/slog"

	"github.com/dimspell/gladiator/internal/app/logger/logging"
	"github.com/dimspell/gladiator/internal/backend/bsession"
	"github.com/dimspell/gladiator/internal/backend/packet"
	"github.com/dimspell/gladiator/internal/model"
)

// HandleJoinGame handles 0x22ff (255-34) command
func (b *Backend) HandleJoinGame(ctx context.Context, session *bsession.Session, req JoinGameRequest) error {
	if session.UserID == 0 {
		return fmt.Errorf("packet-34: user is not logged in")
	}

	data, err := req.Parse()
	if err != nil {
		slog.Warn("Invalid packet", logging.Error(err))
		return nil
	}

	players, err := session.Proxy.JoinGame(ctx, data.RoomName, data.Password)
	if err != nil {
		slog.Error("Could not join game room", logging.Error(err))
		return nil
	}

	// Add info that the player is able to join game
	response := []byte{model.GameStateStarted, 0}

	for _, player := range players {
		if player.Name == session.Username {
			continue
		}

		response = append(response, byte(player.ClassType), 0, 0, 0) // Class type (4 bytes)
		response = append(response, player.IPAddress.To4()[:]...)    // IP Address (4 bytes)
		response = append(response, player.Name...)                  // Player name (null terminated string)
		response = append(response, byte(0))                         // Null byte
	}

	return session.SendToGame(packet.JoinGame, response)
}

type JoinGameRequest []byte

type JoinGameRequestData struct {
	RoomName string
	Password string
}

func (r JoinGameRequest) Parse() (data JoinGameRequestData, err error) {
	rd := packet.NewReader(r)

	data.RoomName, err = rd.ReadString()
	if err != nil {
		return data, fmt.Errorf("packet-34: could not read room name: %w", err)
	}

	// TODO: Read password if given

	// TODO: 216 byte at the end of the packet

	return data, rd.Close()
}

type JoinGameResponse struct {
	Lobby     model.LobbyRoom
	GameState uint16
	Players   []model.LobbyPlayer
}

func (r *JoinGameResponse) Details() []byte {
	buf := []byte{}
	buf = binary.LittleEndian.AppendUint16(buf, r.GameState)
	for _, player := range r.Players {
		buf = append(buf, player.ToBytes()...)
	}
	return buf
}
