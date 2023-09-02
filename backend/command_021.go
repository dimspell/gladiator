package backend

import (
	"encoding/binary"

	"github.com/dispel-re/dispel-multi/model"
)

func (b *Backend) HandlePing(session *model.Session, req PingRequest) error {
	return b.Send(session.Conn, PingClockTime, []byte{1, 0, 0, 0})
}

type PingRequest []byte

func (r PingRequest) Milliseconds() uint32 {
	return binary.LittleEndian.Uint32(r[0:4])
}
