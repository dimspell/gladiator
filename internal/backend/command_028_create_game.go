package backend

import (
	"context"
	"fmt"
	"github.com/dimspell/gladiator/internal/app/logger/logging"
	"log/slog"

	"connectrpc.com/connect"
	multiv1 "github.com/dimspell/gladiator/gen/multi/v1"
	"github.com/dimspell/gladiator/internal/backend/bsession"
	"github.com/dimspell/gladiator/internal/backend/packet"
	"github.com/dimspell/gladiator/internal/backend/proxy"
	"github.com/dimspell/gladiator/internal/model"
)

// HandleCreateGame handles 0x1cff (255-28) command
func (b *Backend) HandleCreateGame(ctx context.Context, session *bsession.Session, req CreateGameRequest) error {
	if session.UserID == 0 {
		return fmt.Errorf("packet-28: user is not logged in")
	}

	data, err := req.Parse()
	if err != nil {
		return err
	}

	switch data.State {
	case uint32(model.GameStateNone):
		hostIPAddress, err := session.Proxy.CreateRoom(proxy.CreateParams{GameID: data.RoomName})
		if err != nil {
			slog.Info("Failed to obtain host address when creating a game", logging.Error(err))
			return session.SendToGame(packet.CreateGame, []byte{2, 0, 0, 0})
		}

		respGame, err := b.gameClient.CreateGame(ctx, connect.NewRequest(&multiv1.CreateGameRequest{
			GameName:      data.RoomName,
			Password:      data.Password,
			MapId:         multiv1.GameMap(data.MapID),
			HostUserId:    session.UserID,
			HostIpAddress: hostIPAddress.String(),
		}))
		if err != nil {
			slog.Info("Failed to create a game", logging.Error(err))
			return session.SendToGame(packet.CreateGame, []byte{2, 0, 0, 0})
		}

		slog.Info("packet-28: created game room", "id", respGame.Msg.Game.GameId, "name", respGame.Msg.Game.Name)
		return session.SendToGame(packet.CreateGame, []byte{model.GameStateCreating, 0, 0, 0})

	case uint32(model.GameStateCreating):
		respGame, err := b.gameClient.GetGame(ctx, connect.NewRequest(&multiv1.GetGameRequest{
			GameRoomId: data.RoomName,
		}))
		if err != nil {
			slog.Info("Failed to get a game room", logging.Error(err))
			return nil // Note: It is not possible to cancel the game creation now.
		}

		if err := session.Proxy.HostRoom(ctx, proxy.HostParams{GameID: respGame.Msg.GetGame().Name}); err != nil {
			slog.Info("Failed to host a game room", logging.Error(err))
			return nil // Note: It is not possible to cancel the game creation now.
		}
		return session.SendToGame(packet.CreateGame, []byte{model.GameStateStarted, 0, 0, 0})
	}

	return fmt.Errorf("packet-28: incorrect game state %d", data.State)
}

type CreateGameRequest []byte

type CreateGameRequestData struct {
	State    uint32
	MapID    uint32
	RoomName string
	Password string
}

func (r CreateGameRequest) Parse() (data CreateGameRequestData, err error) {
	rd := packet.NewReader(r)

	data.State, err = rd.ReadUint32()
	if err != nil {
		return data, fmt.Errorf("packet-28: malformed state %w", err)
	}
	data.MapID, err = rd.ReadUint32()
	if err != nil {
		return data, fmt.Errorf("packet-28: malformed map id %w", err)
	}
	if data.MapID > 5 {
		return data, fmt.Errorf("packet-28: incorrect map id %w", err)
	}
	data.RoomName, err = rd.ReadString()
	if err != nil {
		return data, fmt.Errorf("packet-28: malformed room name %w", err)
	}
	data.Password, err = rd.ReadString()
	if err != nil {
		return data, fmt.Errorf("packet-28: malformed password %w", err)
	}

	return data, rd.Close()
}
