package backend

import (
	"fmt"
	"log/slog"

	"github.com/dimspell/gladiator/internal/backend/bsession"
	"github.com/dimspell/gladiator/internal/backend/packet"
	"github.com/dimspell/gladiator/internal/model"
)

func (b *Backend) HandleSelectChannel(session *bsession.Session, req SelectChannelRequest) error {
	serverName, channelName, err := req.Parse()
	slog.Info("Selected channel", "serverName", serverName, "channelName", channelName, "error", err)
	if serverName == "DISPEL" && channelName == "DISPEL" {
		for idx, user := range session.State.GetLobbyUsers() {
			session.Send(packet.ReceiveMessage, AppendCharacterToLobby(user.Username, model.ClassType(user.ClassType), uint32(idx)))
		}
		// session.Send(ReceiveMessage, NewGlobalMessage("admin", "hello"))
	}

	// session.Send(SelectedChannel, SetChannelName("DISPEL"))

	// session.Send(ReceiveMessage, AppendCharacterToLobby(session.Username, model.ClassTypeKnight, 1))
	// session.Send(ReceiveMessage, SetChannelName("channel"))

	// b.Proxy.Close()

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
