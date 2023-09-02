package backend

import "github.com/dispel-re/dispel-multi/model"

// HandleListGames handles 0x9ff (255-9) command
func (b *Backend) HandleListGames(session *model.Session, req ListGamesRequest) error {
	response := []byte{}
	for _, room := range b.DB.GameRooms() {
		response = append(response, room.Lobby.ToBytes()...)
	}
	return b.Send(session.Conn, ListGames, response)
}

type ListGamesRequest []byte
