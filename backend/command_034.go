package backend

import (
	"bytes"
	"context"
	"fmt"
	"net"

	"github.com/dispel-re/dispel-multi/internal/database"
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

	room, err := b.DB.GetGameRoom(context.TODO(), data.RoomName)
	if err != nil {
		return err
	}

	character, err := b.DB.FindCharacter(context.TODO(), database.FindCharacterParams{
		CharacterName: "TODO",
		UserID:        session.UserID,
	})
	if err != nil {
		return err
	}

	tcpAddr := session.Conn.RemoteAddr().(*net.TCPAddr)
	if err := b.DB.AddPlayerToRoom(context.TODO(), database.AddPlayerToRoomParams{
		GameRoomID:  room.ID,
		CharacterID: character.ID,
		IpAddress:   tcpAddr.IP.To4().String(),
	}); err != nil {
		return err
	}

	gameRoom := model.GameRoom{
		Lobby: model.LobbyRoom{
			HostIPAddress: [4]byte{},
			Name:          room.Name,
			Password:      room.Password.String,
		},
		MapID: uint32(room.MapID),
	}
	copy(gameRoom.Lobby.HostIPAddress[:], net.ParseIP(room.HostIpAddress).To4())

	players, err := b.DB.GetGameRoomPlayers(context.TODO(), data.RoomName)
	if err != nil {
		return err
	}
	for _, player := range players {
		lobbyPlayer := model.LobbyPlayer{
			ClassType: model.ClassType(player.ClassType),
			Name:      player.CharacterName,
		}
		copy(lobbyPlayer.IPAddress[:], net.ParseIP(player.IpAddress).To4())
		gameRoom.Players = append(gameRoom.Players)
	}

	return b.Send(session.Conn, JoinGame, gameRoom.Details())
}

type JoinGameRequest []byte

type JoinGameRequestData struct {
	RoomName string
}

func (r JoinGameRequest) Parse() (data JoinGameRequestData, err error) {
	if bytes.Count(r, []byte{0}) != 1 {
		return data, fmt.Errorf("packet-34: malformed packet, not enough null-terminators")
	}
	split := bytes.SplitN(r, []byte{0}, 2)

	data.RoomName = string(split[0])

	return data, nil
}
