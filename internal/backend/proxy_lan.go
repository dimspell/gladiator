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

	player := wire.Player{
		UserID:      session.UserID,
		Username:    session.Username,
		CharacterID: session.CharacterID,
		ClassType:   byte(session.ClassType),
		IPAddress:   p.MyIPAddress,
	}

	gameRoom := NewGameRoom()
	gameRoom.ID = params.GameID
	gameRoom.Name = params.GameID
	gameRoom.SetHost(player)
	gameRoom.SetPlayer(player)

	session.State.SetGameRoom(gameRoom)

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

func (p *LAN) Join(params JoinParams, session *Session) (net.IP, error) {
	ip := net.ParseIP(p.MyIPAddress)
	if ip == nil {
		return nil, fmt.Errorf("incorrect IP address: %s", p.MyIPAddress)
	}

	return ip, nil
}

func (p *LAN) GetPlayerAddr(params GetPlayerAddrParams, session *Session) (net.IP, error) {
	ip := net.ParseIP(params.IPAddress)
	if ip == nil {
		return net.IP{}, fmt.Errorf("incorrect exchange IP address: %s", params.IPAddress)
	}
	return ip, nil
}

func (p *LAN) Close(session *Session) {
	if gameRoom := session.State.GameRoom(); gameRoom != nil {
		session.SendLeaveRoom(context.TODO(), gameRoom)
		session.State.SetGameRoom(nil)
	}

	// p.RoomPlayers.Clear()
}

func (p *LAN) ExtendWire(session *Session) MessageHandler {
	return &LanMessageHandler{session: session}
}

type LanMessageHandler struct {
	session *Session
}

func (l *LanMessageHandler) Handle(ctx context.Context, payload []byte) error {
	et := wire.ParseEventType(payload)

	switch et {
	case wire.JoinRoom:
		_, msg, err := wire.DecodeTyped[wire.Player](payload)
		if err != nil {
			return nil
		}

		player := msg.Content
		slog.Info("Other player is joining", "playerId", player.ID())

		if l.session.State.GameRoom() == nil {
			return nil
		}

		l.session.State.GameRoom().SetPlayer(player)
	case wire.LeaveRoom, wire.LeaveLobby:
		_, msg, err := wire.DecodeTyped[wire.Player](payload)
		if err != nil {
			return nil
		}

		player := msg.Content
		slog.Info("Other player is leaving", "playerId", player.ID())

		if l.session.State.GameRoom() == nil {
			return nil
		}

		l.session.State.GameRoom().DeletePlayer(player)
	default:
		//	Ignore
	}

	return nil
}
