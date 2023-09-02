package backend

import (
	"bytes"

	"github.com/dispel-re/dispel-multi/model"
)

func (b *Backend) HandleSelectChannel(session *model.Session, req SelectChannelRequest) error {
	channelName, _ := req.Parse()
	if channelName == "DISPEL" {
		// b.Send(session.Conn, ReceiveMessage, NewMessage())
	}

	return nil
}

type SelectChannelRequest []byte

func (r SelectChannelRequest) Parse() (string, error) {
	split := bytes.Split(r, []byte{0})
	return string(split[0]), nil
}
