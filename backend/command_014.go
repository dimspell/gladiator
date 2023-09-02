package backend

import "github.com/dispel-re/dispel-multi/model"

func (b *Backend) HandleSendLobbyMessage(session *model.Session, req SendLobbyMessageRequest) error {
	resp := NewLobbyMessage(session.Character.CharacterName, req.Message())
	return b.Send(session.Conn, ReceiveMessage, resp)
}

type SendLobbyMessageRequest []byte

func (c SendLobbyMessageRequest) Message() string {
	return string(c[:])
}
