package backend

import (
	"fmt"

	"github.com/dispel-re/dispel-multi/model"
)

// HandleJoinGame handles 0x22ff (255-34) command
func (b *Backend) HandleJoinGame(session *model.Session, req JoinGameRequest) error {
	if session.UserID != 0 {
		return fmt.Errorf("packet-34: user has been already logged in")
	}

	// gameRoom := b.DB.GameRooms()[0]
	var gameRoom model.GameRoom
	return b.Send(session.Conn, JoinGame, gameRoom.Details())
}

type JoinGameRequest []byte
