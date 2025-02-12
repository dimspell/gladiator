package direct

import (
	"context"
	"fmt"
	"net"

	"github.com/dimspell/gladiator/internal/backend/bsession"
	"github.com/dimspell/gladiator/internal/backend/proxy"
)

var _ proxy.Proxy = (*LAN)(nil)

type LAN struct {
	MyIPAddress string

	// FIXME: Not thread-safe
	BySession map[*bsession.Session]*proxy.GameRoom
}

func NewLAN(myIPAddress string) *LAN {
	if myIPAddress == "" {
		myIPAddress = "127.0.0.1"
	}

	return &LAN{
		MyIPAddress: myIPAddress,
		BySession:   make(map[*bsession.Session]*proxy.GameRoom),
	}
}

func (p *LAN) GetHostIP(hostIpAddress net.IP, _ *bsession.Session) net.IP {
	return hostIpAddress
}

func (p *LAN) CreateRoom(params proxy.CreateParams, session *bsession.Session) (net.IP, error) {
	p.Close(session)

	ip := net.ParseIP(p.MyIPAddress)
	if ip == nil {
		return net.IP{}, fmt.Errorf("incorrect host IP address: %s", p.MyIPAddress)
	}

	p.BySession[session] = proxy.NewGameRoom(params.GameID, session.ToPlayer(ip))

	return ip, nil
}

func (p *LAN) HostRoom(ctx context.Context, params proxy.HostParams, session *bsession.Session) error {
	if err := session.SendSetRoomReady(ctx, params.GameID); err != nil {
		return err
	}

	return nil
}

func (p *LAN) SelectGame(params proxy.GameData, session *bsession.Session) error {
	p.Close(session)

	host, err := params.FindHostUser()
	if err != nil {
		return err
	}
	gameRoom := proxy.NewGameRoom(params.Game.GameId, host)
	for _, player := range params.ToWirePlayers() {
		gameRoom.SetPlayer(player)
	}

	p.BySession[session] = gameRoom

	return nil
}

func (p *LAN) Join(ctx context.Context, params proxy.JoinParams, session *bsession.Session) (net.IP, error) {
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

func (p *LAN) GetPlayerAddr(params proxy.GetPlayerAddrParams, session *bsession.Session) (net.IP, error) {
	ip := net.ParseIP(params.IPAddress)
	if ip == nil {
		return net.IP{}, fmt.Errorf("incorrect exchange IP address: %s", params.IPAddress)
	}
	return ip, nil
}

func (p *LAN) ConnectToPlayer(ctx context.Context, params proxy.GetPlayerAddrParams, session *bsession.Session) (net.IP, error) {
	return p.GetPlayerAddr(params, session)
}

func (p *LAN) Close(session *bsession.Session) {
	delete(p.BySession, session)
}

func (p *LAN) NewWebSocketHandler(session *bsession.Session) proxy.MessageHandler {
	return &LanMessageHandler{session: session, BySession: p.BySession}
}
