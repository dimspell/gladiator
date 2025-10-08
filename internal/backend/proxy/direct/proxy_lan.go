package direct

import (
	"context"
	"fmt"
	"log/slog"
	"net"

	"connectrpc.com/connect"
	multiv1 "github.com/dimspell/gladiator/gen/multi/v1"
	"github.com/dimspell/gladiator/gen/multi/v1/multiv1connect"
	"github.com/dimspell/gladiator/internal/app/logger/logging"
	"github.com/dimspell/gladiator/internal/backend/bsession"
	"github.com/dimspell/gladiator/internal/backend/packet"
	"github.com/dimspell/gladiator/internal/backend/proxy"
	"github.com/dimspell/gladiator/internal/model"
	"github.com/dimspell/gladiator/internal/wire"
)

var _ proxy.ProxyClient = (*LAN)(nil)

type ProxyLAN struct {
	MyIPAddress string
}

func (p *ProxyLAN) Mode() model.RunMode { return model.RunModeLAN }

func (p *ProxyLAN) Create(session *bsession.Session, gameClient multiv1connect.GameServiceClient) proxy.ProxyClient {
	ipAddress := p.MyIPAddress

	if ipAddress == "" {
		ipAddress = "127.0.0.1"
	}

	if session == nil {
		panic("nil session")
	}

	return &LAN{
		Session:           session,
		MyIPAddress:       ipAddress,
		GameServiceClient: gameClient,
	}
}

type LAN struct {
	GameServiceClient multiv1connect.GameServiceClient
	MyIPAddress       string
	Session           *bsession.Session

	// GameRoom *GameRoom
}

func (p *LAN) CreateRoom(ctx context.Context, params proxy.CreateParams) error {
	p.Close()

	ip := net.ParseIP(p.MyIPAddress).To4()
	if ip == nil {
		return fmt.Errorf("incorrect host IP address: %s", p.MyIPAddress)
	}

	_, err := p.GameServiceClient.CreateGame(ctx, connect.NewRequest(&multiv1.CreateGameRequest{
		GameName:      params.GameID,
		Password:      params.Password,
		MapId:         multiv1.GameMap(params.MapId),
		HostUserId:    p.Session.UserID,
		HostIpAddress: ip.String(),
	}))
	if err != nil {
		return fmt.Errorf("could not create game room: %w", err)
	}

	// p.GameRoom = NewGameRoom(params.GameID, p.Session.ToPlayer(ip))

	return nil
}

func (p *LAN) SetRoomReady(ctx context.Context, params proxy.CreateParams) error {
	respGame, err := p.GameServiceClient.GetGame(ctx, connect.NewRequest(&multiv1.GetGameRequest{
		GameRoomId: params.GameID,
	}))
	if err != nil {
		slog.Info("Failed to get a game room", logging.Error(err))
		return err
	}

	if respGame.Msg.Game.MapId != multiv1.GameMap(params.MapId) {
		return fmt.Errorf("incorrect map id: %d", respGame.Msg.Game.MapId)
	}

	if err := p.Session.SendSetRoomReady(ctx, params.GameID); err != nil {
		return err
	}

	return nil
}

func (p *LAN) ListGames(ctx context.Context) ([]model.LobbyRoom, error) {
	resp, err := p.GameServiceClient.ListGames(ctx, connect.NewRequest(&multiv1.ListGamesRequest{}))
	if err != nil {
		return nil, fmt.Errorf("could not list games: %w", err)
	}

	var lobbyRooms []model.LobbyRoom
	for _, room := range resp.Msg.GetGames() {
		roomIP := net.ParseIP(room.HostIpAddress).To4()
		if roomIP == nil {
			continue
		}
		lobbyRooms = append(lobbyRooms, model.LobbyRoom{
			Name:          room.Name,
			Password:      room.Password,
			HostIPAddress: roomIP,
		})
	}
	return lobbyRooms, nil
}

func (p *LAN) GetGame(ctx context.Context, roomID string) (*model.LobbyRoom, []model.LobbyPlayer, error) {
	p.Close()

	respGame, err := p.GameServiceClient.GetGame(ctx, connect.NewRequest(&multiv1.GetGameRequest{
		GameRoomId: roomID,
	}))
	if err != nil {
		slog.Warn("No game found", logging.RoomID(roomID), logging.Error(err))
		return nil, nil, err
	}

	hostPlayer, err := proxy.FindPlayer(respGame.Msg.Players, respGame.Msg.Game.HostUserId)
	if err != nil {
		return nil, nil, err
	}

	// gameRoom := NewGameRoom(roomID, hostPlayer)
	// for _, player := range proxy.ToWirePlayers(respGame.Msg.GetPlayers()) {
	// 	gameRoom.SetPlayer(player)
	// }
	// p.GameRoom = gameRoom

	hostIP := net.ParseIP(hostPlayer.IPAddress).To4()
	if hostIP == nil {
		return nil, nil, fmt.Errorf("incorrect host IP address: %s", hostPlayer.IPAddress)
	}

	room := &model.LobbyRoom{
		HostIPAddress: hostIP,
		Name:          respGame.Msg.Game.Name,
		Password:      respGame.Msg.Game.Password,
		MapID:         respGame.Msg.Game.MapId,
	}
	players := p.mapPlayersToLobbyPlayers(respGame.Msg.GetPlayers())
	return room, players, nil
}

func (p *LAN) JoinGame(ctx context.Context, roomID string, password string) ([]model.LobbyPlayer, error) {
	ip := net.ParseIP(p.MyIPAddress)
	if ip == nil {
		return nil, fmt.Errorf("incorrect IP address: %s", p.MyIPAddress)
	}

	// if p.GameRoom == nil {
	// 	return nil, fmt.Errorf("could not find current session among the peers for user ID: %d", p.Session.UserID)
	// }
	// p.GameRoom.SetPlayer(p.Session.ToPlayer(ip))

	joinResp, err := p.GameServiceClient.JoinGame(ctx, connect.NewRequest(&multiv1.JoinGameRequest{
		UserId:     p.Session.UserID,
		GameRoomId: roomID,
		IpAddress:  ip.String(),
	}))
	if err != nil {
		return nil, err
	}

	players := p.mapPlayersToLobbyPlayers(joinResp.Msg.GetPlayers())
	return players, nil
}

func (p *LAN) mapPlayersToLobbyPlayers(resp []*multiv1.Player) []model.LobbyPlayer {
	var players []model.LobbyPlayer
	for _, player := range proxy.ToWirePlayers(resp) {
		ip := net.ParseIP(player.IPAddress).To4()
		if ip == nil {
			continue
		}
		if player.UserID == p.Session.UserID {
			continue
		}
		players = append(players, model.LobbyPlayer{
			Name:      player.Username,
			ClassType: multiv1.ClassType(player.ClassType),
			IPAddress: ip,
		})
	}
	return players
}

func (p *LAN) Close() {}

func (p *LAN) Handle(ctx context.Context, payload []byte) error {
	et := wire.ParseEventType(payload)

	switch et {
	case wire.JoinRoom:
		// Ignore

	case wire.LeaveRoom, wire.LeaveLobby:
		// Ignore

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

		response := packet.NewHostSwitch(true, ip)
		if err := p.Session.SendToGame(packet.HostMigration, response); err != nil {
			slog.Error("Failed to send host migration response", logging.Error(err))
			return nil
		}
	default:
		//	Ignore
	}

	return nil
}
