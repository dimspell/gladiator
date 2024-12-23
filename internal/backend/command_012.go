package backend

import (
	"fmt"
	"github.com/dimspell/gladiator/internal/backend/packet"
	"github.com/dimspell/gladiator/internal/model"
	"log/slog"
)

func (b *Backend) HandleSelectChannel(session *Session, req SelectChannelRequest) error {
	serverName, channelName, err := req.Parse()
	slog.Info("Selected channel", "serverName", serverName, "channelName", channelName, "error", err)
	if serverName == "DISPEL" && channelName == "DISPEL" {
		for idx, user := range session.lobbyUsers {
			b.Send(session.Conn, ReceiveMessage, AppendCharacterToLobby(user.Username, model.ClassType(user.ClassType), uint32(idx)))
		}
		//b.Send(session.Conn, ReceiveMessage, NewGlobalMessage("admin", "hello"))
	}

	//b.Send(session.Conn, SelectedChannel, SetChannelName("DISPEL"))

	//b.Send(session.Conn, ReceiveMessage, AppendCharacterToLobby(session.Username, model.ClassTypeKnight, 1))
	//b.Send(session.Conn, ReceiveMessage, SetChannelName("channel"))

	//b.Proxy.Close()

	return nil
}

type SelectChannelRequest []byte

func (r SelectChannelRequest) Parse() (serverName string, channelName string, err error) {
	rd := packet.NewReader(r)
	serverName, err = rd.ReadString()
	if err != nil {
		return "", "", fmt.Errorf("error parsing server name: %w", err)
	}
	channelName, err = rd.ReadString()
	if err != nil {
		return "", "", fmt.Errorf("error parsing server name: %w", err)
	}
	return serverName, channelName, err
}
