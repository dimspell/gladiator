package backend

import (
	"encoding/binary"
	"fmt"
	"time"

	"github.com/dispel-re/dispel-multi/model"
)

func (b *Backend) HandlePing(session *model.Session, req PingRequest) error {
	return b.Send(session.Conn, PingClockTime, []byte{1, 0, 0, 0})
}

type PingRequest []byte

func (r PingRequest) Parse() (uint32, error) {
	if len(r) != 4 {
		return 0, fmt.Errorf("packet-21: invalid length")
	}
	return binary.LittleEndian.Uint32(r[0:4]), nil
}

func (r PingRequest) ParseDate() (time.Time, error) {
	msec, err := r.Parse()
	if err != nil {
		return time.Time{}, err
	}
	return time.UnixMilli(int64(msec)).In(time.UTC), nil
}
