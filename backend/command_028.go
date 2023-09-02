package backend

import (
	"bytes"
	"encoding/binary"

	"github.com/dispel-re/dispel-multi/model"
)

// HandleCreateGame handles 0x1cff (255-28) command
func (b *Backend) HandleCreateGame(session *model.Session, req CreateGameRequest) error {
	state, _, _, _, _ := req.Parse()

	resp := make([]byte, 4)

	switch state {
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

func (r CreateGameRequest) Parse() (state uint32, mapId uint32, roomName string, password string, err error) {
	state = binary.LittleEndian.Uint32(r[0:4])
	mapId = binary.LittleEndian.Uint32(r[4:8])

	split := bytes.Split(r[8:], []byte{0})
	roomName = string(split[0])
	password = string(split[1])

	return
}
