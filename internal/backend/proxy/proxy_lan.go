package proxy

import (
	"context"
	"fmt"
	"log/slog"
	"net"

	"github.com/dimspell/gladiator/internal/backend/bsession"
	"github.com/dimspell/gladiator/internal/wire"
)

var _ Proxy = (*LAN)(nil)

type LAN struct {
	MyIPAddress string

	// FIXME: Not thread-safe
	BySession map[*bsession.Session]*GameRoom
}

func NewLAN(myIPAddress string) *LAN {
	if myIPAddress == "" {
		myIPAddress = "127.0.0.1"
	}

	return &LAN{
		MyIPAddress: myIPAddress,
		BySession:   make(map[*bsession.Session]*GameRoom),
	}
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

	p.BySession[session] = NewGameRoom(params.GameID, session.ToPlayer(ip))

	return ip, nil
}

func (p *LAN) HostRoom(ctx context.Context, params HostParams, session *bsession.Session) error {
	if err := session.SendSetRoomReady(ctx, params.GameID); err != nil {
		return err
	}

	return nil
}

func (p *LAN) SelectGame(params GameData, session *bsession.Session) error {
	p.Close(session)

	host, err := params.FindHostUser()
	if err != nil {
		return err
	}
	gameRoom := NewGameRoom(params.Game.GameId, host)
	for _, player := range params.ToWirePlayers() {
		gameRoom.SetPlayer(player)
	}

	p.BySession[session] = gameRoom

	return nil
}

func (p *LAN) Join(ctx context.Context, params JoinParams, session *bsession.Session) (net.IP, error) {
	ip := net.ParseIP(p.MyIPAddress)
	if ip == nil {
		return nil, fmt.Errorf("incorrect IP address: %s", p.MyIPAddress)
	}

	mapping, exist := p.BySession[session]
	if !exist {
		return nil, fmt.Errorf("could not find current session among the peers for user ID: %s", session.GetUserID())
	}
	mapping.SetPlayer(session.ToPlayer(ip))

	return ip, nil
}

func (p *LAN) GetPlayerAddr(params GetPlayerAddrParams, session *bsession.Session) (net.IP, error) {
	ip := net.ParseIP(params.IPAddress)
	if ip == nil {
		return net.IP{}, fmt.Errorf("incorrect exchange IP address: %s", params.IPAddress)
	}
	return ip, nil
}

func (p *LAN) ConnectToPlayer(ctx context.Context, params GetPlayerAddrParams, session *bsession.Session) (net.IP, error) {
	return p.GetPlayerAddr(params, session)
}

func (p *LAN) Close(session *bsession.Session) {
	delete(p.BySession, session)
}

func (p *LAN) NewWebSocketHandler(session *bsession.Session) MessageHandler {
	return &LanMessageHandler{session: session, BySession: p.BySession}
}

type LanMessageHandler struct {
	session   *bsession.Session
	BySession map[*bsession.Session]*GameRoom
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

		gameRoom, found := l.BySession[l.session]
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

		gameRoom, found := l.BySession[l.session]
		if !found {
			return nil
		}

		gameRoom.DeletePlayer(player.ID())
	default:
		//	Ignore
	}

	return nil
}
