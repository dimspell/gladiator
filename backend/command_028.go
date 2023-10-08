package backend

import (
	"bytes"
	"encoding/binary"

	"github.com/dispel-re/dispel-multi/model"
)

// HandleCreateGame handles 0x1cff (255-28) command
func (b *Backend) HandleCreateGame(session *model.Session, req CreateGameRequest) error {
	data, err := req.Parse()
	if err != nil {
		return err
	}

	resp := make([]byte, 4)

	switch data.State {
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

type CreateGameRequestData struct {
	State    uint32
	MapID    uint32
	RoomName string
	Password string
}

func (r CreateGameRequest) Parse() (data CreateGameRequestData, err error) {
	data.State = binary.LittleEndian.Uint32(r[0:4])
	data.MapID = binary.LittleEndian.Uint32(r[4:8])

	split := bytes.Split(r[8:], []byte{0})
	data.RoomName = string(split[0])
	data.Password = string(split[1])
	return data, nil
}
