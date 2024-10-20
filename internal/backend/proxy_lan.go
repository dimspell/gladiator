package backend

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/dimspell/gladiator/internal/wire"
)

var _ Proxy = (*LAN)(nil)

type LAN struct {
	MyIPAddress string

	Rooms map[string]*GameRoom
}

func NewLAN(myIPAddress string) *LAN {
	if myIPAddress == "" {
		myIPAddress = "127.0.0.1"
	}

	return &LAN{MyIPAddress: myIPAddress}
}

// TODO: This function should actually load the map and save it to the struct

func (p *LAN) GetHostIP(hostIpAddress string, _ *Session) net.IP {
	ip := net.ParseIP(hostIpAddress)
	if ip == nil {
		return net.IP{}
	}
	return ip
}

func (p *LAN) CreateRoom(params CreateParams, session *Session) (net.IP, error) {
	ip := net.ParseIP(p.MyIPAddress)
	if ip == nil {
		return net.IP{}, fmt.Errorf("incorrect host IP address: %s", p.MyIPAddress)
	}

	gameRoom := NewGameRoom()
	gameRoom.ID = params.GameID
	gameRoom.Name = params.GameID
	gameRoom.Ready = false
	gameRoom.SetHost(session.ToPlayer())
	gameRoom.SetPlayer(session.ToPlayer())

	session.SetGameRoom(gameRoom)

	return ip, nil
}

func (p *LAN) HostRoom(params HostParams, session *Session) error {
	ctx, cancel := context.WithTimeout(context.TODO(), 3*time.Second)
	defer cancel()

	if err := session.SendSetRoomReady(ctx, params.GameID); err != nil {
		return err
	}

	return nil
}

func (p *LAN) Join(_ JoinParams, _ *Session) error { return nil }

func (p *LAN) GetPlayerAddr(params GetPlayerAddrParams, session *Session) (net.IP, error) {
	ip := net.ParseIP(params.IPAddress)
	if ip == nil {
		return net.IP{}, fmt.Errorf("incorrect exchange IP address: %s", params.IPAddress)
	}
	return ip, nil
}

func (p *LAN) Close(session *Session) {
	session.gameRoom = nil
	// p.RoomPlayers.Clear()
}

func (p *LAN) ExtendWire(ctx context.Context, session *Session, et wire.EventType, payload []byte) {
	// if err != nil {
	// 	slog.Debug("failed to decode wire", slog.String("type", et.String()), slog.String("payload", string(payload)))
	// }

	switch et {
	case wire.JoinRoom:
		_, msg, err := wire.DecodeTyped[wire.Player](payload)
		if err != nil {
			return
		}

		player := msg.Content
		slog.Info("Other player is joining", "playerId", player.ID())

		// p.RoomPlayers.Store(player.UserID, player)
	case wire.LeaveRoom:
		_, msg, err := wire.DecodeTyped[wire.Player](payload)
		if err != nil {
			return
		}

		player := msg.Content
		slog.Info("Other player is leaving", "playerId", player.ID())
		// p.RoomPlayers.Delete(player.UserID)
	default:
		//	Ignore
	}
}
