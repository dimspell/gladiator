package backend

import (
	"bytes"

	"github.com/dimspell/gladiator/model"
)

func (b *Backend) HandleSelectChannel(session *model.Session, req SelectChannelRequest) error {
	channelName, _ := req.Parse()
	if channelName == "DISPEL" {
		b.Send(session.Conn, ReceiveMessage, NewGlobalMessage("admin", "hello"))
	}

	b.Proxy.Close()

	return nil
}

type SelectChannelRequest []byte

func (r SelectChannelRequest) Parse() (string, error) {
	split := bytes.Split(r, []byte{0})
	return string(split[0]), nil
}
