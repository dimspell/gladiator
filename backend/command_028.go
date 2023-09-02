package backend

import (
	"bytes"
	"encoding/binary"

	"github.com/dispel-re/dispel-multi/model"
)

// HandleCreateGame handles 0x1cff (255-28) command
func (b *Backend) HandleCreateGame(session *model.Session, req CreateGameRequest) error {
	resp := make([]byte, 4)

	switch req.State() {
	case uint32(0):
		binary.LittleEndian.PutUint32(resp[0:4], 1)
		// b.CreateGameRoom()
		break
	case uint32(1):
		binary.LittleEndian.PutUint32(resp[0:4], 2)
		break
	}

	return b.Send(session.Conn, CreateGame, resp)
}

type CreateGameRequest []byte

func (c CreateGameRequest) State() uint32 {
	return binary.LittleEndian.Uint32(c[0:4])
}

func (c CreateGameRequest) MapID() uint32 {
	return binary.LittleEndian.Uint32(c[4:8])
}

func (c CreateGameRequest) NameAndPassword() (roomName string, password string) {
	split := bytes.Split(c[8:], []byte{0})
	roomName = string(split[0])
	password = string(split[1])
	return roomName, password
}
