package backend

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/dimspell/gladiator/backend/packet"
	multiv1 "github.com/dimspell/gladiator/gen/multi/v1"
	"github.com/dimspell/gladiator/model"
)

// HandleJoinGame handles 0x22ff (255-34) command
func (b *Backend) HandleJoinGame(session *model.Session, req JoinGameRequest) error {
	if session.UserID == 0 {
		return fmt.Errorf("packet-34: user is not logged in")
	}

	data, err := req.Parse()
	if err != nil {
		slog.Warn("Invalid packet", "error", err)
		return nil
	}

	respGame, err := b.gameClient.GetGame(context.TODO(), connect.NewRequest(&multiv1.GetGameRequest{
		UserId:   session.UserID,
		GameName: data.RoomName,
	}))
	if err != nil {
		return err
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

	response := []byte{model.GameStateStarted, 0}
	for _, player := range respPlayers.Msg.GetPlayers() {
		proxyIP, err := b.Proxy.Exchange(respGame.Msg.GetGame().GetName(), player.Username, player.IpAddress)
		if err != nil {
			return err
		}
		if bytes.Equal(proxyIP, []byte{0, 0, 0, 0}) {
			return fmt.Errorf("packet-34: incorrect proxy for %v", player.IpAddress)
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

	return b.Send(session.Conn, JoinGame, response)
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
