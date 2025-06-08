package direct

import (
	"context"
	"fmt"
	"log/slog"
	"net"

	"github.com/dimspell/gladiator/internal/app/logger/logging"
	"github.com/dimspell/gladiator/internal/backend/bsession"
	"github.com/dimspell/gladiator/internal/backend/packet"
	"github.com/dimspell/gladiator/internal/backend/proxy"
	"github.com/dimspell/gladiator/internal/wire"
)

var _ proxy.ProxyClient = (*LAN)(nil)

type ProxyLAN struct {
	MyIPAddress string
}

func (p *ProxyLAN) Create(session *bsession.Session) proxy.ProxyClient {
	ipAddress := p.MyIPAddress

	if ipAddress == "" {
		ipAddress = "127.0.0.1"
	}

	if session == nil {
		panic("nil session")
	}

	return &LAN{
		Session:     session,
		MyIPAddress: ipAddress,
	}
}

type LAN struct {
	MyIPAddress string
	Session     *bsession.Session
	GameRoom    *GameRoom
}

func (p *LAN) GetHostIP(hostIpAddress net.IP) net.IP {
	return hostIpAddress
}

func (p *LAN) CreateRoom(params proxy.CreateParams) (net.IP, error) {
	p.Close()

	ip := net.ParseIP(p.MyIPAddress)
	if ip == nil {
		return net.IP{}, fmt.Errorf("incorrect host IP address: %s", p.MyIPAddress)
	}

	p.GameRoom = NewGameRoom(params.GameID, p.Session.ToPlayer(ip))

	return ip, nil
}

func (p *LAN) HostRoom(ctx context.Context, params proxy.HostParams) error {
	if err := p.Session.SendSetRoomReady(ctx, params.GameID); err != nil {
		return err
	}

	return nil
}

func (p *LAN) SelectGame(params proxy.GameData) error {
	p.Close()

	host, err := params.FindHostUser()
	if err != nil {
		return err
	}
	gameRoom := NewGameRoom(params.Game.GameId, host)
	for _, player := range params.ToWirePlayers() {
		gameRoom.SetPlayer(player)
	}

	p.GameRoom = gameRoom

	return nil
}

func (p *LAN) Join(ctx context.Context, params proxy.JoinParams) (net.IP, error) {
	ip := net.ParseIP(p.MyIPAddress)
	if ip == nil {
		return nil, fmt.Errorf("incorrect IP address: %s", p.MyIPAddress)
	}

	if p.GameRoom == nil {
		return nil, fmt.Errorf("could not find current session among the peers for user ID: %d", p.Session.UserID)
	}
	p.GameRoom.SetPlayer(p.Session.ToPlayer(ip))

	return ip, nil
}

func (p *LAN) GetPlayerAddr(params proxy.GetPlayerAddrParams) (net.IP, error) {
	ip := net.ParseIP(params.IPAddress)
	if ip == nil {
		return net.IP{}, fmt.Errorf("incorrect exchange IP address: %s", params.IPAddress)
	}
	return ip, nil
}

func (p *LAN) ConnectToPlayer(ctx context.Context, params proxy.GetPlayerAddrParams) (net.IP, error) {
	return p.GetPlayerAddr(params)
}

func (p *LAN) Close() {}

func (p *LAN) Handle(ctx context.Context, payload []byte) error {
	et := wire.ParseEventType(payload)

	switch et {
	case wire.JoinRoom:
		_, msg, err := wire.DecodeTyped[wire.Player](payload)
		if err != nil {
			return nil
		}

		player := msg.Content
		slog.Info("Other player is joining", "playerId", player.ID())

		gameRoom, found := p.GameRoom, p.GameRoom != nil
		if !found {
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

		gameRoom, found := p.GameRoom, p.GameRoom != nil
		if !found {
			return nil
		}

		gameRoom.DeletePlayer(player.UserID)
	case wire.HostMigration:
		_, msg, err := wire.DecodeTyped[wire.Player](payload)
		if err != nil {
			return nil
		}

		ip := net.ParseIP(msg.Content.IPAddress)
		if ip == nil {
			slog.Error("Failed to parse IP address", "ip", msg.Content.IPAddress)
			return nil
		}

		response := make([]byte, 8)
		copy(response[0:4], []byte{1, 0, 0, 0})
		copy(response[4:], ip.To4())

		if err := p.Session.SendToGame(packet.HostMigration, response); err != nil {
			slog.Error("Failed to send host migration response", logging.Error(err))
			return nil
		}
	default:
		//	Ignore
	}

	return nil
}
