// Package p2p provides the implementation of a WebRTC-based peer-to-peer proxy for multiplayer networking.
package p2p

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"sync"

	"connectrpc.com/connect"
	multiv1 "github.com/dimspell/gladiator/gen/multi/v1"
	"github.com/dimspell/gladiator/gen/multi/v1/multiv1connect"
	"github.com/dimspell/gladiator/internal/app/logger/logging"
	"github.com/dimspell/gladiator/internal/backend/bsession"
	"github.com/dimspell/gladiator/internal/backend/proxy"
	"github.com/dimspell/gladiator/internal/backend/proxy/transport"
	"github.com/dimspell/gladiator/internal/backend/redirect"
	"github.com/dimspell/gladiator/internal/model"
	"github.com/dimspell/gladiator/internal/wire"
	"github.com/pion/webrtc/v4"
)

var _ proxy.ProxyClient = (*PeerToPeer)(nil)

// ProxyP2P is the factory for creating PeerToPeer proxy instances.
type ProxyP2P struct {
	ICEServers []webrtc.ICEServer
	IPPrefix   net.IP
}

func (p *ProxyP2P) Mode() model.RunMode { return model.RunModeWebRTC }

func (p *ProxyP2P) Create(session *bsession.Session, gameClient multiv1connect.GameServiceClient) proxy.ProxyClient {
	return NewPeerToPeer(p, gameClient, session)
}

// PeerToPeer implements the Proxy interface for WebRTC-based peer-to-peer connections.
// It manages game rooms, peer connections, and network addressing for multiplayer games.
type PeerToPeer struct {
	mu            sync.Mutex
	session       *bsession.Session
	logger        *slog.Logger
	webrtcConfig  webrtc.Configuration
	gameClient    multiv1connect.GameServiceClient
	router        *transport.PacketRouter
	p2pTransport  *webrtcTransport
	selfID        string
	roomID        string
	currentHostID string

	// peers holds active WebRTC peer connections indexed by user ID string
	peers map[string]*Peer
}

// NewPeerToPeer creates a new PeerToPeer proxy instance.
func NewPeerToPeer(config *ProxyP2P, client multiv1connect.GameServiceClient, session *bsession.Session) *PeerToPeer {
	ipPrefix := config.IPPrefix
	if ipPrefix == nil {
		ipPrefix = net.IPv4(127, 0, 0, 0)
	}

	webrtcConfig := webrtc.Configuration{}
	if config.ICEServers != nil {
		webrtcConfig.ICEServers = append(webrtcConfig.ICEServers, config.ICEServers...)
	}

	p := &PeerToPeer{
		session:      session,
		logger:       slog.With(slog.String("proxy", "p2p"), slog.String("sessionId", session.ID)),
		webrtcConfig: webrtcConfig,
		gameClient:   client,
		selfID:       peerID(session.UserID),
		peers:        make(map[string]*Peer),
	}

	// webrtcTransport multiplexes every WebRTC peer through the PacketRouter's
	// single Send/Recv surface. Its lookup reads p.peers under p.mu.
	p2pTransport := &webrtcTransport{
		logger: p.logger,
		lookup: func(peerID string) (*Peer, bool) {
			p.mu.Lock()
			peer, ok := p.peers[peerID]
			p.mu.Unlock()
			return peer, ok
		},
		recvCh: make(chan transport.TransportPacket, 256),
	}
	p.p2pTransport = p2pTransport
	p.router = transport.NewPacketRouter(
		p.logger,
		p.selfID,
		session,
		redirect.NewManager(redirect.WithIPPrefix(ipPrefix.To4())),
		p2pTransport,
	)

	return p
}

func peerID(userID int64) string { return fmt.Sprintf("%d", userID) }

// Reset cleans up all resources and resets the proxy state.
func (p *PeerToPeer) Reset() {
	p.mu.Lock()
	peers := make([]*Peer, 0, len(p.peers))
	for id, peer := range p.peers {
		peers = append(peers, peer)
		delete(p.peers, id)
	}

	p.router.Reset()
	p.roomID = ""
	p.currentHostID = ""
	p.mu.Unlock()

	// Close peer connections outside the lock: peer.Close() can synchronously
	// fire OnConnectionStateChange which re-acquires p.mu.
	for _, peer := range peers {
		peer.Close()
	}
}

func (p *PeerToPeer) CreateRoom(ctx context.Context, params proxy.CreateParams) error {
	p.Reset()

	roomID := params.GameID
	p.mu.Lock()
	p.roomID = roomID
	p.selfID = peerID(p.session.UserID)
	p.currentHostID = p.selfID
	p.mu.Unlock()

	p.router.SetRoomState(roomID, p.selfID, p.selfID)
	if err := p.router.Connect(ctx, roomID); err != nil {
		return fmt.Errorf("failed to connect p2p transport: %w", err)
	}

	_, err := p.gameClient.CreateGame(ctx, connect.NewRequest(&multiv1.CreateGameRequest{
		GameName:      params.GameID,
		Password:      params.Password,
		MapId:         multiv1.GameMap(params.MapId),
		HostUserId:    p.session.UserID,
		HostIpAddress: "",
	}))
	if err != nil {
		return fmt.Errorf("could not create game room: %w", err)
	}

	return nil
}

func (p *PeerToPeer) SetRoomReady(ctx context.Context, params proxy.CreateParams) error {
	respGame, err := p.gameClient.GetGame(ctx, connect.NewRequest(&multiv1.GetGameRequest{
		GameRoomId: params.GameID,
	}))
	if err != nil {
		p.logger.Info("Failed to get a game room", logging.Error(err))
		return err
	}

	if respGame.Msg.Game.MapId != multiv1.GameMap(params.MapId) {
		return fmt.Errorf("incorrect map id: %d", respGame.Msg.Game.MapId)
	}

	if err := p.session.SendSetRoomReady(ctx, params.GameID); err != nil {
		return fmt.Errorf("could not send set room ready: %w", err)
	}

	return nil
}

func (p *PeerToPeer) ListGames(ctx context.Context) ([]model.LobbyRoom, error) {
	resp, err := p.gameClient.ListGames(ctx, connect.NewRequest(&multiv1.ListGamesRequest{}))
	if err != nil {
		return nil, fmt.Errorf("could not list games: %w", err)
	}

	var lobbyRooms []model.LobbyRoom
	for _, room := range resp.Msg.GetGames() {
		lobbyRooms = append(lobbyRooms, model.LobbyRoom{
			Name:          room.Name,
			Password:      room.Password,
			HostIPAddress: net.IPv4(127, 0, 0, 2).To4(),
		})
	}
	return lobbyRooms, nil
}

func (p *PeerToPeer) GetGame(ctx context.Context, roomID string) (*model.LobbyRoom, []model.LobbyPlayer, error) {
	p.Reset()

	respGame, err := p.gameClient.GetGame(ctx, connect.NewRequest(&multiv1.GetGameRequest{GameRoomId: roomID}))
	if err != nil {
		return nil, nil, fmt.Errorf("could not get game room: %w", err)
	}

	hostPlayer, err := proxy.FindPlayer(respGame.Msg.Players, respGame.Msg.Game.HostUserId)
	if err != nil {
		return nil, nil, fmt.Errorf("could not find the host player: %w", err)
	}

	p.mu.Lock()
	selfID := p.selfID
	p.mu.Unlock()

	var lobbyPlayers []model.LobbyPlayer
	for _, player := range respGame.Msg.Players {
		pid := peerID(player.UserId)
		if pid == selfID {
			continue
		}

		ip, err := p.router.Manager().AssignIP(pid)
		if err != nil {
			return nil, nil, fmt.Errorf("could not assign ip: %w", err)
		}

		lobbyPlayers = append(lobbyPlayers, model.LobbyPlayer{
			ClassType: player.ClassType,
			IPAddress: net.ParseIP(ip).To4(),
			Name:      player.Username,
		})
	}

	p.mu.Lock()
	p.selfID = peerID(p.session.UserID)
	p.roomID = roomID
	p.currentHostID = peerID(hostPlayer.UserID)
	p.mu.Unlock()

	p.router.SetRoomState(roomID, p.selfID, p.currentHostID)

	lobbyRoom := &model.LobbyRoom{
		Name:          respGame.Msg.Game.Name,
		Password:      respGame.Msg.Game.Password,
		HostIPAddress: net.IPv4(127, 0, 0, 2),
		MapID:         multiv1.GameMap(respGame.Msg.Game.MapId),
	}

	return lobbyRoom, lobbyPlayers, nil
}

func (p *PeerToPeer) JoinGame(ctx context.Context, roomID string, password string) ([]model.LobbyPlayer, error) {
	respGame, err := p.gameClient.GetGame(ctx, connect.NewRequest(&multiv1.GetGameRequest{GameRoomId: roomID}))
	if err != nil {
		return nil, fmt.Errorf("could not get game room: %w", err)
	}

	respJoin, err := p.gameClient.JoinGame(ctx, connect.NewRequest(&multiv1.JoinGameRequest{
		UserId:     p.session.UserID,
		GameRoomId: roomID,
		IpAddress:  "",
	}))
	if err != nil {
		return nil, fmt.Errorf("could not join game room: %w", err)
	}

	hostPlayer, err := proxy.FindPlayer(respGame.Msg.GetPlayers(), respGame.Msg.GetGame().GetHostUserId())
	if err != nil {
		return nil, fmt.Errorf("could not find the host player: %w", err)
	}
	hostID := peerID(hostPlayer.UserID)

	p.mu.Lock()
	currentHostID := p.currentHostID
	p.mu.Unlock()

	if err := p.router.Connect(ctx, roomID); err != nil {
		return nil, fmt.Errorf("failed to connect p2p transport: %w", err)
	}

	var lobbyPlayers []model.LobbyPlayer
	for _, player := range respJoin.Msg.GetPlayers() {
		if player.UserId == p.session.UserID {
			continue
		}

		pid := peerID(player.UserId)
		ipAddress, ok := p.router.Manager().GetPeerIP(pid)
		if !ok {
			return nil, fmt.Errorf("not found the IP for a peer with ID %s", pid)
		}
		ipv4 := net.ParseIP(ipAddress).To4()
		if ipv4 == nil {
			return nil, fmt.Errorf("invalid IP %s", ipAddress)
		}

		p.logger.Debug("Starting fake host for", logging.PeerID(pid), "host", pid == hostID)

		var tcpPort int
		if pid == currentHostID {
			tcpPort = 6114
		}

		onTCPMessage := p.router.OnTCPMessage(roomID, pid)
		onUDPMessage := p.router.OnUDPMessage(roomID, pid)
		onHostDisconnected := func(host *redirect.FakeHost, forced bool) {
			p.logger.Warn("Host went offline", logging.PeerID(pid), "ip", host.AssignedIP, "forced", forced)
			if forced {
				p.Reset()
			} else {
				p.router.Manager().StopHost(host)
			}
		}

		_, err := p.router.Manager().StartHost(ctx, pid, ipAddress, tcpPort, 6113, onTCPMessage, onUDPMessage, onHostDisconnected)
		if err != nil {
			return nil, err
		}

		lobbyPlayers = append(lobbyPlayers, model.LobbyPlayer{
			ClassType: player.ClassType,
			IPAddress: net.ParseIP(ipAddress).To4(),
			Name:      player.Username,
		})
	}

	return lobbyPlayers, nil
}

// Close closes the connection for a session.
func (p *PeerToPeer) Close() {
	p.Reset()
}

// Handle processes incoming WebSocket messages for WebRTC signaling.
func (p *PeerToPeer) Handle(ctx context.Context, payload []byte) error {
	eventType := wire.ParseEventType(payload)

	switch eventType {
	case wire.LobbyUsers, wire.JoinLobby, wire.CreateRoom:
		return nil
	case wire.JoinRoom:
		return decodeAndHandle(ctx, p.logger, payload, eventType, p.handleJoinRoom)
	case wire.LeaveRoom, wire.LeaveLobby:
		return decodeAndHandle(ctx, p.logger, payload, eventType, p.handleLeaveRoom)
	case wire.HostMigration:
		return decodeAndHandle(ctx, p.logger, payload, eventType, p.handleHostMigration)
	case wire.RTCOffer:
		return p.handleRTCOffer(ctx, payload)
	case wire.RTCAnswer:
		return p.handleRTCAnswer(ctx, payload)
	case wire.RTCICECandidate:
		return p.handleRTCCandidate(ctx, payload)
	default:
		p.logger.Debug("unknown wire message", "type", eventType.String())
		return nil
	}
}

// Generic handler for simple event messages
func decodeAndHandle[T any](
	ctx context.Context,
	logger *slog.Logger,
	payload []byte,
	eventType wire.EventType,
	handler func(context.Context, T) error,
) error {
	_, msg, err := wire.DecodeTyped[T](payload)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to decode payload for event: %s", eventType.String()), logging.Error(err), "payload", string(payload))
		return err
	}
	return handler(ctx, msg.Content)
}

func (p *PeerToPeer) handleJoinRoom(ctx context.Context, player wire.Player) error {
	pid := peerID(player.UserID)

	p.mu.Lock()
	selfID := p.selfID
	currentHostID := p.currentHostID
	p.mu.Unlock()

	if pid == selfID {
		return nil
	}

	p.logger.Info("New player joining", logging.PeerID(pid))

	// Mirror relay host behavior: if we are the current host, dial into the local game server
	// and forward packets to this joining peer. DynamicJoin reuses the PacketRouter's
	// dispatch engine (StartGuest + OnTCPMessage/OnUDPMessage) instead of a bespoke path.
	if currentHostID == selfID {
		p.router.DynamicJoin(ctx, p.roomID, pid)
	}

	// Create WebRTC peer connection for the new player
	if err := p.createPeerConnection(ctx, pid, true); err != nil {
		return fmt.Errorf("failed to create peer connection: %w", err)
	}

	return nil
}

func (p *PeerToPeer) handleLeaveRoom(ctx context.Context, player wire.Player) error {
	pid := peerID(player.UserID)

	p.mu.Lock()
	selfID := p.selfID
	p.mu.Unlock()

	if selfID == pid {
		return nil
	}

	p.mu.Lock()
	if peer, ok := p.peers[pid]; ok {
		peer.Close()
		delete(p.peers, pid)
	}
	p.mu.Unlock()

	p.router.Manager().RemoveByRemoteID(pid)
	return nil
}

func (p *PeerToPeer) handleHostMigration(ctx context.Context, newHost wire.Player) error {
	newHostID := peerID(newHost.UserID)

	p.mu.Lock()
	p.currentHostID = newHostID
	p.mu.Unlock()

	p.router.SetCurrentHostID(newHostID)

	p.logger.Info("Host migration", "newHost", newHostID)
	return nil
}

func (p *PeerToPeer) handleRTCOffer(ctx context.Context, payload []byte) error {
	_, msg, err := wire.DecodeTyped[wire.Offer](payload)
	if err != nil {
		return fmt.Errorf("failed to decode RTC offer: %w", err)
	}

	// Check if this offer is for us
	p.mu.Lock()
	selfID := p.selfID
	p.mu.Unlock()

	if msg.To != selfID {
		return nil
	}

	fromID := peerID(msg.Content.CreatorID)
	p.logger.Debug("Received RTC offer", "from", fromID)

	// Create peer connection if it doesn't exist
	p.mu.Lock()
	peer, exists := p.peers[fromID]
	p.mu.Unlock()

	if !exists {
		if err := p.createPeerConnection(ctx, fromID, false); err != nil {
			return fmt.Errorf("failed to create peer connection: %w", err)
		}
		p.mu.Lock()
		peer = p.peers[fromID]
		p.mu.Unlock()
	}

	if peer == nil || peer.connection == nil {
		return fmt.Errorf("peer connection not found for %s", fromID)
	}

	// Set remote description
	if err := peer.connection.SetRemoteDescription(msg.Content.Offer); err != nil {
		return fmt.Errorf("failed to set remote description: %w", err)
	}

	// Create and send answer
	answer, err := peer.connection.CreateAnswer(nil)
	if err != nil {
		return fmt.Errorf("failed to create answer: %w", err)
	}

	if err := peer.connection.SetLocalDescription(answer); err != nil {
		return fmt.Errorf("failed to set local description: %w", err)
	}

	if err := p.session.SendRTCAnswer(ctx, answer, msg.Content.CreatorID); err != nil {
		return fmt.Errorf("failed to send answer: %w", err)
	}

	return nil
}

func (p *PeerToPeer) handleRTCAnswer(ctx context.Context, payload []byte) error {
	_, msg, err := wire.DecodeTyped[wire.Offer](payload)
	if err != nil {
		return fmt.Errorf("failed to decode RTC answer: %w", err)
	}

	// Check if this answer is for us
	p.mu.Lock()
	selfID := p.selfID
	p.mu.Unlock()

	if msg.To != selfID {
		return nil
	}

	fromID := peerID(msg.Content.CreatorID)
	p.logger.Debug("Received RTC answer", "from", fromID)

	p.mu.Lock()
	peer, ok := p.peers[fromID]
	p.mu.Unlock()

	if !ok || peer.connection == nil {
		return fmt.Errorf("peer connection not found for %s", fromID)
	}

	answer := webrtc.SessionDescription{
		Type: webrtc.SDPTypeAnswer,
		SDP:  msg.Content.Offer.SDP,
	}

	if err := peer.connection.SetRemoteDescription(answer); err != nil {
		return fmt.Errorf("failed to set remote description: %w", err)
	}

	return nil
}

func (p *PeerToPeer) handleRTCCandidate(ctx context.Context, payload []byte) error {
	_, msg, err := wire.DecodeTyped[webrtc.ICECandidateInit](payload)
	if err != nil {
		return fmt.Errorf("failed to decode RTC candidate: %w", err)
	}

	// Check if this candidate is for us
	p.mu.Lock()
	selfID := p.selfID
	p.mu.Unlock()

	if msg.To != selfID {
		return nil
	}

	fromID := msg.From
	p.logger.Debug("Received ICE candidate", "from", fromID)

	p.mu.Lock()
	peer, ok := p.peers[fromID]
	p.mu.Unlock()

	if !ok || peer.connection == nil {
		p.logger.Warn("Peer connection not found for ICE candidate", logging.PeerID(fromID))
		return nil
	}

	if err := peer.connection.AddICECandidate(msg.Content); err != nil {
		return fmt.Errorf("failed to add ICE candidate: %w", err)
	}

	return nil
}

// createPeerConnection creates a new WebRTC peer connection for a remote peer.
func (p *PeerToPeer) createPeerConnection(ctx context.Context, remotePeerID string, createOffer bool) error {
	p.mu.Lock()
	if _, exists := p.peers[remotePeerID]; exists {
		p.mu.Unlock()
		return nil // Already exists
	}
	p.mu.Unlock()

	pc, err := webrtc.NewPeerConnection(p.webrtcConfig)
	if err != nil {
		return fmt.Errorf("failed to create peer connection: %w", err)
	}

	peer := &Peer{
		peerID:     remotePeerID,
		connection: pc,
		logger:     p.logger.With(logging.PeerID(remotePeerID)),
	}

	// Handle ICE candidates
	pc.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate == nil {
			return
		}
		remoteUserID, _ := parseUserID(remotePeerID)
		if err := p.session.SendRTCICECandidate(ctx, candidate.ToJSON(), remoteUserID); err != nil {
			peer.logger.Error("Failed to send ICE candidate", logging.Error(err))
		}
	})

	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		peer.logger.Debug("Connection state changed", "state", state.String())
		if state == webrtc.PeerConnectionStateConnected {
			peer.connected = true
		} else if state == webrtc.PeerConnectionStateDisconnected || state == webrtc.PeerConnectionStateFailed {
			peer.Close()
			p.mu.Lock()
			delete(p.peers, remotePeerID)
			p.mu.Unlock()
		}
	})

	// Handle incoming data channels (for the answerer)
	pc.OnDataChannel(func(dc *webrtc.DataChannel) {
		peer.logger.Debug("Received data channel", "label", dc.Label())
		peer.setDataChannel(dc)
		p.setupDataChannel(peer, dc)
	})

	p.mu.Lock()
	p.peers[remotePeerID] = peer
	p.mu.Unlock()

	if createOffer {
		// Create data channel (for the offerer)
		dc, err := pc.CreateDataChannel("game", nil)
		if err != nil {
			return fmt.Errorf("failed to create data channel: %w", err)
		}
		peer.setDataChannel(dc)
		p.setupDataChannel(peer, dc)

		// Create and send offer
		offer, err := pc.CreateOffer(nil)
		if err != nil {
			return fmt.Errorf("failed to create offer: %w", err)
		}

		if err := pc.SetLocalDescription(offer); err != nil {
			return fmt.Errorf("failed to set local description: %w", err)
		}

		remoteUserID, _ := parseUserID(remotePeerID)
		if err := p.session.SendRTCOffer(ctx, offer, remoteUserID); err != nil {
			return fmt.Errorf("failed to send offer: %w", err)
		}
	}

	return nil
}

// setupDataChannel configures data channel callbacks for receiving packets.
// Inbound messages are tagged with the sender's peer ID and pushed into the
// webrtcTransport, where the PacketRouter dispatch loop routes them to the
// matching FakeHost's ProxyTCP/ProxyUDP (mirroring relay's onTransportPacket).
func (p *PeerToPeer) setupDataChannel(peer *Peer, dc *webrtc.DataChannel) {
	dc.OnOpen(func() {
		peer.logger.Debug("Data channel opened")
	})

	dc.OnClose(func() {
		peer.logger.Debug("Data channel closed")
	})

	dc.OnError(func(err error) {
		peer.logger.Warn("Data channel error", logging.Error(err))
	})

	dc.OnMessage(func(msg webrtc.DataChannelMessage) {
		if len(msg.Data) < 2 {
			return
		}

		var kind transport.PacketKind
		switch msg.Data[0] {
		case 'T':
			kind = transport.KindTCP
		case 'U':
			kind = transport.KindUDP
		default:
			return
		}

		p.p2pTransport.deliver(peer.peerID, kind, msg.Data[1:])
	})
}

func parseUserID(s string) (int64, error) {
	var id int64
	_, err := fmt.Sscanf(s, "%d", &id)
	return id, err
}
