package backend

import (
	"fmt"

	"github.com/dispel-re/dispel-multi/model"
)

// HandleSelectGame handles 0x45ff (255-69) command
func (b *Backend) HandleSelectGame(session *model.Session, req SelectGameRequest) error {
	if session.UserID != 0 {
		return fmt.Errorf("packet-69: user has been already logged in")
	}

	// gameRoom := b.DB.GameRooms()[0]
	var gameRoom model.GameRoom
	return b.Send(session.Conn, SelectGame, gameRoom.Details())
}

type SelectGameRequest []byte
