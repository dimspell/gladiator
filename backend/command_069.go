package backend

import "github.com/dispel-re/dispel-multi/model"

// HandleSelectGame handles 0x45ff (255-69) command
func (b *Backend) HandleSelectGame(session *model.Session, req SelectGameRequest) error {
	gameRoom := b.DB.GameRooms()[0]
	return b.Send(session.Conn, SelectGame, gameRoom.Details())
}

type SelectGameRequest []byte
