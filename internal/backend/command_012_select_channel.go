package backend

import (
	"context"
	"fmt"
	"github.com/dimspell/gladiator/internal/backend/bsession"
	"github.com/dimspell/gladiator/internal/backend/packet"
	"github.com/dimspell/gladiator/internal/model"
	"log/slog"
)

func (b *Backend) HandleSelectChannel(ctx context.Context, session *bsession.Session, req SelectChannelRequest) error {
	serverName, channelName, err := req.Parse()
	slog.Info("Selected channel", "serverName", serverName, "channelName", channelName, "error", err)

	if err := session.SendFromBackend(packet.ReceiveMessage, SetChannelName(channelName)); err != nil {
		return err
	}

	if serverName == "DISPEL" && channelName == "DISPEL" {
		for idx, user := range session.State.GetLobbyUsers() {
			session.SendFromBackend(packet.ReceiveMessage, AppendCharacterToLobby(user.Username, model.ClassType(user.ClassType), uint32(idx)))
		}
		// session.Send(ReceiveMessage, NewGlobalMessage("admin", "hello"))
	}

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
		return "", "", fmt.Errorf("error parsing channel name: %w", err)
	}
	return serverName, channelName, err
}
