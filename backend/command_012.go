package backend

import (
	"bytes"

	"github.com/dispel-re/dispel-multi/model"
)

func (b *Backend) HandleSelectChannel(session *model.Session, req SelectChannelRequest) error {
	if req.ChannelName() == "DISPEL" {
		// b.Send(session.Conn, ReceiveMessage, NewMessage())
	}

	return nil
}

type SelectChannelRequest []byte

func (r SelectChannelRequest) ChannelName() string {
	split := bytes.Split(r, []byte{0})
	return string(split[0])
}
