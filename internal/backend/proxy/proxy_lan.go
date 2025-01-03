package proxy

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/dimspell/gladiator/internal/backend/bsession"
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

func (p *LAN) GetHostIP(hostIpAddress net.IP, _ *bsession.Session) net.IP {
	return hostIpAddress
}

func (p *LAN) CreateRoom(params CreateParams, session *bsession.Session) (net.IP, error) {
	p.Close(session)

	ip := net.ParseIP(p.MyIPAddress)
	if ip == nil {
		return net.IP{}, fmt.Errorf("incorrect host IP address: %s", p.MyIPAddress)
	}

	player := session.ToPlayer(ip)

	gameRoom := bsession.NewGameRoom(params.GameID, player)
	session.State.SetGameRoom(gameRoom)

	return ip, nil
}

func (p *LAN) HostRoom(params HostParams, session *bsession.Session) error {
	ctx, cancel := context.WithTimeout(context.TODO(), 3*time.Second)
	defer cancel()

	if err := session.SendSetRoomReady(ctx, params.GameID); err != nil {
		return err
	}

	return nil
}

func (p *LAN) Join(params JoinParams, session *bsession.Session) (net.IP, error) {
	ip := net.ParseIP(p.MyIPAddress)
	if ip == nil {
		return nil, fmt.Errorf("incorrect IP address: %s", p.MyIPAddress)
	}

	return ip, nil
}

func (p *LAN) GetPlayerAddr(params GetPlayerAddrParams, session *bsession.Session) (net.IP, error) {
	ip := net.ParseIP(params.IPAddress)
	if ip == nil {
		return net.IP{}, fmt.Errorf("incorrect exchange IP address: %s", params.IPAddress)
	}
	return ip, nil
}

func (p *LAN) Close(session *bsession.Session) {
	if gameRoom, err := session.State.GameRoom(); err != nil {
		session.SendLeaveRoom(context.TODO(), gameRoom)
		session.State.SetGameRoom(nil)
	}

	// p.RoomPlayers.Clear()
}

func (p *LAN) ExtendWire(session *bsession.Session) MessageHandler {
	return &LanMessageHandler{session: session}
}

type LanMessageHandler struct {
	session *bsession.Session
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

		gameRoom, err := l.session.State.GameRoom()
		if err != nil {
			return nil
		}

		gameRoom.SetPlayer(player)
	case wire.LeaveRoom, wire.LeaveLobby:
		_, msg, err := wire.DecodeTyped[wire.Player](payload)
		if err != nil {
			return nil
		}

		player := msg.Content
		slog.Info("Other player is leaving", "playerId", player.ID())

		gameRoom, err := l.session.State.GameRoom()
		if err != nil {
			return nil
		}

		gameRoom.DeletePlayer(player)
	default:
		//	Ignore
	}

	return nil
}
