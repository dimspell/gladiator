package backend

import (
	"context"
	"encoding/binary"
	"fmt"
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

	response := make([]byte, 4)

	switch data.State {
	case uint32(model.GameStateNone):
		hostIPAddress, err := session.Proxy.CreateRoom(proxy.CreateParams{GameID: data.RoomName})
		if err != nil {
			return fmt.Errorf("packet-28: incorrect host address %w", err)
		}

		respGame, err := b.gameClient.CreateGame(ctx, connect.NewRequest(&multiv1.CreateGameRequest{
			GameName:      data.RoomName,
			Password:      data.Password,
			MapId:         multiv1.GameMap(data.MapID),
			HostUserId:    session.UserID,
			HostIpAddress: hostIPAddress.String(),
		}))
		if err != nil {
			return err
		}
		slog.Info("packet-28: created game room", "id", respGame.Msg.Game.GameId, "name", respGame.Msg.Game.Name)

		binary.LittleEndian.PutUint32(response[0:4], uint32(model.GameStateCreating))
		break
	case uint32(model.GameStateCreating):
		respGame, err := b.gameClient.GetGame(ctx, connect.NewRequest(&multiv1.GetGameRequest{
			GameRoomId: data.RoomName,
		}))
		if err != nil {
			return err
		}

		if err := session.Proxy.HostRoom(ctx, proxy.HostParams{GameID: respGame.Msg.GetGame().Name}); err != nil {
			return err
		}
		binary.LittleEndian.PutUint32(response[0:4], uint32(model.GameStateStarted))
		break
	}

	return session.SendToGame(packet.CreateGame, response)
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
