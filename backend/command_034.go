package backend

import "github.com/dispel-re/dispel-multi/model"

// HandleJoinGame handles 0x22ff (255-34) command
func (b *Backend) HandleJoinGame(session *model.Session, req JoinGameRequest) error {
	gameRoom := b.DB.GameRooms()[0]
	return b.Send(session.Conn, JoinGame, gameRoom.Details())
}

type JoinGameRequest []byte
