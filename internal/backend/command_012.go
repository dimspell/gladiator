package backend

import (
	"bytes"
)

func (b *Backend) HandleSelectChannel(session *Session, req SelectChannelRequest) error {
	channelName, _ := req.Parse()
	if channelName == "DISPEL" {
		//b.Send(session.Conn, ReceiveMessage, NewGlobalMessage("admin", "hello"))
	}

	b.Send(session.Conn, SelectedChannel, SetChannelName("channel"))

	//b.Send(session.Conn, ReceiveMessage, AppendCharacterToLobby(session.Username, model.ClassTypeKnight, 1))
	//b.Send(session.Conn, ReceiveMessage, SetChannelName("channel"))

	//b.Proxy.Close()

	return nil
}

type SelectChannelRequest []byte

func (r SelectChannelRequest) Parse() (string, error) {
	split := bytes.Split(r, []byte{0})
	return string(split[0]), nil
}
