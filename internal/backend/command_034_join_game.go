package backend

import (
	"bytes"
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

// HandleJoinGame handles 0x22ff (255-34) command
func (b *Backend) HandleJoinGame(ctx context.Context, session *bsession.Session, req JoinGameRequest) error {
	if session.UserID == 0 {
		return fmt.Errorf("packet-34: user is not logged in")
	}

	data, err := req.Parse()
	if err != nil {
		slog.Warn("Invalid packet", "error", err)
		return nil
	}

	respGame, err := b.gameClient.GetGame(ctx, connect.NewRequest(&multiv1.GetGameRequest{
		GameRoomId: data.RoomName,
	}))
	if err != nil {
		return err
	}

	myIpAddr, err := b.Proxy.Join(ctx, proxy.JoinParams{
		HostUserID: respGame.Msg.GetGame().HostUserId,
		HostUserIP: respGame.Msg.GetGame().HostIpAddress,
		GameID:     respGame.Msg.GetGame().GetName(),
	}, session)
	if err != nil {
		return err
	}

	respJoin, err := b.gameClient.JoinGame(ctx, connect.NewRequest(&multiv1.JoinGameRequest{
		UserId:     session.UserID,
		GameRoomId: respGame.Msg.Game.GetGameId(),
		IpAddress:  myIpAddr.To4().String(),
	}))
	if err != nil {
		slog.Error("Could not join game room", "error", err)
		return nil
	}

	response := []byte{model.GameStateStarted, 0}
	for _, player := range respJoin.Msg.GetPlayers() {
		if player.UserId == session.UserID {
			continue
		}

		ps := proxy.GetPlayerAddrParams{
			GameID:     respGame.Msg.GetGame().GetName(),
			UserID:     player.UserId,
			IPAddress:  player.IpAddress,
			HostUserID: fmt.Sprintf("%d", respGame.Msg.GetGame().HostUserId),
		}
		proxyIP, err := b.Proxy.ConnectToPlayer(ctx, ps, session)
		if err != nil {
			return err
		}
		if bytes.Equal(proxyIP, []byte{0, 0, 0, 0}) {
			return fmt.Errorf("packet-34: incorrect proxy for %v", player.IpAddress)
		}

		// TODO: make sure the host is the first one
		// lobbyPlayer := model.LobbyPlayer{
		//	ClassType: model.ClassType(player.ClassType),
		//	Name:      player.Username,
		//	IPAddress: proxyIP.To4(),
		// }
		// gameRoom.Players = append(gameRoom.Players, lobbyPlayer)

		response = append(response, byte(player.ClassType), 0, 0, 0) // Class type (4 bytes)
		response = append(response, proxyIP.To4()[:]...)             // IP Address (4 bytes)
		response = append(response, player.Username...)              // Player name (null terminated string)
		response = append(response, byte(0))                         // Null byte
	}

	return session.SendToGame(packet.JoinGame, response)
}

type JoinGameRequest []byte

type JoinGameRequestData struct {
	RoomName string
	Password string
}

func (r JoinGameRequest) Parse() (data JoinGameRequestData, err error) {
	rd := packet.NewReader(r)

	data.RoomName, err = rd.ReadString()
	if err != nil {
		return data, fmt.Errorf("packet-34: could not read room name: %w", err)
	}

	// TODO: Read password if given

	// TODO: 216 byte at the end of the packet

	return data, rd.Close()
}

type JoinGameResponse struct {
	Lobby     model.LobbyRoom
	GameState uint16
	Players   []model.LobbyPlayer
}

func (r *JoinGameResponse) Details() []byte {
	buf := []byte{}
	buf = binary.LittleEndian.AppendUint16(buf, r.GameState)
	for _, player := range r.Players {
		buf = append(buf, player.ToBytes()...)
	}
	return buf
}
