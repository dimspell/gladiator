// Package libp2p provides a proxy implementation backed by the libp2p networking
// library.  Each player runs a lightweight libp2p host; peers discover each
// other by exchanging their multiaddresses through the existing WebSocket
// signalling channel (the same one the WebRTC proxy uses for SDP and ICE).
// Once the addresses are known the player dials the remote host directly and
// multiplexes TCP and UDP game traffic over a single bidirectional stream.
package libp2p

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"time"

	"connectrpc.com/connect"
	libp2p "github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/multiformats/go-multiaddr"

	multiv1 "github.com/dimspell/gladiator/gen/multi/v1"
	"github.com/dimspell/gladiator/gen/multi/v1/multiv1connect"
	"github.com/dimspell/gladiator/internal/app/logger/logging"
	"github.com/dimspell/gladiator/internal/backend/bsession"
	"github.com/dimspell/gladiator/internal/backend/proxy"
	"github.com/dimspell/gladiator/internal/backend/redirect"
	"github.com/dimspell/gladiator/internal/model"
	"github.com/dimspell/gladiator/internal/wire"
)

const gameProtocol protocol.ID = "/gladiator/game/1.0.0"

// ProxyLibp2p is the factory / configuration object for the libp2p proxy.
type ProxyLibp2p struct {
	// ListenAddrs is the set of multiaddress strings the local libp2p host will
	// listen on.  Leave nil to use the library default (all interfaces, random
	// port).
	ListenAddrs []string

	// IPPrefix is the 127.x.x.0 subnet used for fake-host IP assignment.
	IPPrefix net.IP
}

func (p *ProxyLibp2p) Mode() model.RunMode { return model.RunModeLibp2p }

func (p *ProxyLibp2p) Create(session *bsession.Session, gameClient multiv1connect.GameServiceClient) proxy.ProxyClient {
	return newLibp2pProxy(p, gameClient, session)
}

// Libp2pProxy implements proxy.ProxyClient using libp2p streams.
type Libp2pProxy struct {
	mu            sync.Mutex
	session       *bsession.Session
	logger        *slog.Logger
	gameClient    multiv1connect.GameServiceClient
	manager       *redirect.HostManager
	selfID        string
	roomID        string
	currentHostID string

	// ipPrefix is the /24 block used by the redirect manager.
	ipPrefix net.IP

	// h is the local libp2p host for this session.
	h host.Host

	// peers maps peerID string → open stream to that peer.
	peers map[string]*peerStream

	// wg tracks receiveFromPeer goroutines so Close / reset can wait for them.
	wg sync.WaitGroup

	// readTimeout is the per-iteration read deadline on peer streams.  If a
	// remote peer silently drops the connection, the read will time out and
	// the receive goroutine will exit cleanly instead of leaking.
	readTimeout time.Duration

	// peerIDToUserID maps libp2p peer IDs → game user ID strings so that
	// handleIncomingStream can store inbound streams under the game user ID.
	peerIDToUserID map[string]string
}

// peerStream wraps a single libp2p stream that carries both TCP and UDP frames.
type peerStream struct {
	mu     sync.Mutex
	peerID string
	stream network.Stream
}

func (ps *peerStream) send(data []byte) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	if ps.stream == nil {
		return fmt.Errorf("stream to peer %s is nil", ps.peerID)
	}
	// Write a simple length-prefixed frame: [4-byte big-endian length][payload]
	buf := make([]byte, 4+len(data))
	l := uint32(len(data))
	buf[0] = byte(l >> 24)
	buf[1] = byte(l >> 16)
	buf[2] = byte(l >> 8)
	buf[3] = byte(l)
	copy(buf[4:], data)
	_, err := ps.stream.Write(buf)
	return err
}

func (ps *peerStream) close() {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	if ps.stream != nil {
		_ = ps.stream.Reset()
		ps.stream = nil
	}
}

var _ proxy.ProxyClient = (*Libp2pProxy)(nil)

func newLibp2pProxy(config *ProxyLibp2p, gameClient multiv1connect.GameServiceClient, session *bsession.Session) *Libp2pProxy {
	ipPrefix := config.IPPrefix
	if ipPrefix == nil {
		ipPrefix = net.IPv4(127, 0, 0, 0)
	}

	return &Libp2pProxy{
		session:        session,
		logger:         slog.With(slog.String("proxy", "libp2p"), slog.String("sessionId", session.ID)),
		gameClient:     gameClient,
		manager:        redirect.NewManager(redirect.WithIPPrefix(ipPrefix.To4())),
		selfID:         peerIDStr(session.UserID),
		ipPrefix:       ipPrefix,
		peers:          make(map[string]*peerStream),
		readTimeout:    30 * time.Second,
		peerIDToUserID: make(map[string]string),
	}
}

func peerIDStr(i int64) string { return fmt.Sprintf("%d", i) }

// startHost creates (or recreates) the local libp2p host and announces its
// multiaddresses through the WebSocket signalling channel.
func (p *Libp2pProxy) startHost(ctx context.Context, listenAddrs []string) error {
	opts := []libp2p.Option{
		libp2p.NATPortMap(),
	}
	if len(listenAddrs) > 0 {
		mas := make([]multiaddr.Multiaddr, 0, len(listenAddrs))
		for _, a := range listenAddrs {
			ma, err := multiaddr.NewMultiaddr(a)
			if err != nil {
				return fmt.Errorf("invalid listen addr %q: %w", a, err)
			}
			mas = append(mas, ma)
		}
		opts = append(opts, libp2p.ListenAddrs(mas...))
	}

	h, err := libp2p.New(opts...)
	if err != nil {
		return fmt.Errorf("create libp2p host: %w", err)
	}
	p.h = h

	// Register stream handler for incoming connections from peers.
	h.SetStreamHandler(gameProtocol, p.handleIncomingStream)

	p.logger.Info("libp2p host started", "peerID", h.ID().String(), "addrs", h.Addrs())

	// Build the full /p2p/ multiaddresses and broadcast them.
	fullAddrs := make([]string, 0, len(h.Addrs()))
	for _, a := range h.Addrs() {
		full := fmt.Sprintf("%s/p2p/%s", a.String(), h.ID().String())
		fullAddrs = append(fullAddrs, full)
	}

	if err := p.session.SendLibp2pAddresses(ctx, fullAddrs); err != nil {
		_ = h.Close()
		return fmt.Errorf("broadcast libp2p addresses: %w", err)
	}
	return nil
}

// reset tears down the libp2p host, all peer streams and the redirect manager.
func (p *Libp2pProxy) reset() {
	// Close all peer streams under the lock so that blocked reads error out.
	p.mu.Lock()
	for id, ps := range p.peers {
		ps.close()
		delete(p.peers, id)
	}
	p.mu.Unlock()

	// Wait for receiveFromPeer goroutines to finish (with a timeout guard).
	// The stream resets above should cause their reads to error out, letting
	// them return and call wg.Done().  We must NOT hold p.mu here because the
	// goroutines' defers need to acquire it to clean up the peers map.
	doneCh := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(doneCh)
	}()
	select {
	case <-doneCh:
	case <-time.After(10 * time.Second):
		p.logger.Warn("Timed out waiting for receiveFromPeer goroutines")
	}

	p.mu.Lock()
	if p.h != nil {
		_ = p.h.Close()
		p.h = nil
	}
	p.manager.StopAll()
	p.roomID = ""
	p.currentHostID = ""
	p.mu.Unlock()
}

// ─── ProxyClient interface ────────────────────────────────────────────────────

func (p *Libp2pProxy) CreateRoom(ctx context.Context, params proxy.CreateParams) error {
	p.reset()

	p.mu.Lock()
	p.roomID = params.GameID
	p.selfID = peerIDStr(p.session.UserID)
	p.currentHostID = p.selfID
	p.mu.Unlock()

	if err := p.startHost(ctx, nil); err != nil {
		return fmt.Errorf("start libp2p host: %w", err)
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

func (p *Libp2pProxy) SetRoomReady(ctx context.Context, params proxy.CreateParams) error {
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

func (p *Libp2pProxy) ListGames(ctx context.Context) ([]model.LobbyRoom, error) {
	resp, err := p.gameClient.ListGames(ctx, connect.NewRequest(&multiv1.ListGamesRequest{}))
	if err != nil {
		return nil, fmt.Errorf("could not list games: %w", err)
	}

	var rooms []model.LobbyRoom
	for _, room := range resp.Msg.GetGames() {
		rooms = append(rooms, model.LobbyRoom{
			Name:          room.Name,
			Password:      room.Password,
			HostIPAddress: net.IPv4(127, 0, 0, 2).To4(),
		})
	}
	return rooms, nil
}

func (p *Libp2pProxy) GetGame(ctx context.Context, roomID string) (*model.LobbyRoom, []model.LobbyPlayer, error) {
	p.reset()

	respGame, err := p.gameClient.GetGame(ctx, connect.NewRequest(&multiv1.GetGameRequest{GameRoomId: roomID}))
	if err != nil {
		return nil, nil, fmt.Errorf("could not get game room: %w", err)
	}

	hostPlayer, err := proxy.FindPlayer(respGame.Msg.Players, respGame.Msg.Game.HostUserId)
	if err != nil {
		return nil, nil, fmt.Errorf("could not find the host player: %w", err)
	}

	var lobbyPlayers []model.LobbyPlayer
	for _, player := range respGame.Msg.Players {
		pid := peerIDStr(player.UserId)
		p.mu.Lock()
		selfID := p.selfID
		p.mu.Unlock()
		if pid == selfID {
			continue
		}

		ip, err := p.manager.AssignIP(pid)
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
	p.selfID = peerIDStr(p.session.UserID)
	p.roomID = roomID
	p.currentHostID = peerIDStr(hostPlayer.UserID)
	p.mu.Unlock()

	lobbyRoom := &model.LobbyRoom{
		Name:          respGame.Msg.Game.Name,
		Password:      respGame.Msg.Game.Password,
		HostIPAddress: net.IPv4(127, 0, 0, 2),
		MapID:         multiv1.GameMap(respGame.Msg.Game.MapId),
	}
	return lobbyRoom, lobbyPlayers, nil
}

func (p *Libp2pProxy) JoinGame(ctx context.Context, roomID string, password string) ([]model.LobbyPlayer, error) {
	respGame, err := p.gameClient.GetGame(ctx, connect.NewRequest(&multiv1.GetGameRequest{GameRoomId: roomID}))
	if err != nil {
		return nil, fmt.Errorf("could not get game room: %w", err)
	}

	// Start our own libp2p host first so we can advertise our address.
	if err := p.startHost(ctx, nil); err != nil {
		return nil, fmt.Errorf("start libp2p host: %w", err)
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
	hostID := peerIDStr(hostPlayer.UserID)

	var lobbyPlayers []model.LobbyPlayer
	for _, player := range respJoin.Msg.GetPlayers() {
		if player.UserId == p.session.UserID {
			continue
		}

		pid := peerIDStr(player.UserId)
		ipAddress, ok := p.manager.GetPeerIP(pid)
		if !ok {
			return nil, fmt.Errorf("not found the IP for a peer with ID %s", pid)
		}
		ipv4 := net.ParseIP(ipAddress).To4()
		if ipv4 == nil {
			return nil, fmt.Errorf("invalid IP %s", ipAddress)
		}

		p.logger.Debug("Starting fake host for", logging.PeerID(pid), "host", pid == hostID)

		var tcpPort int
		p.mu.Lock()
		currentHostID := p.currentHostID
		p.mu.Unlock()
		if pid == currentHostID {
			tcpPort = 6114
		}

		onTCPMessage := p.onTCPMessage(pid)
		onUDPMessage := p.onUDPMessage(pid)
		onHostDisconnected := func(host *redirect.FakeHost, forced bool) {
			p.logger.Warn("Host went offline", logging.PeerID(pid), "ip", host.AssignedIP, "forced", forced)
			if forced {
				p.reset()
			} else {
				p.manager.StopHost(host)
			}
		}

		if _, err := p.manager.StartHost(ctx, pid, ipAddress, tcpPort, 6113, onTCPMessage, onUDPMessage, onHostDisconnected); err != nil {
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

func (p *Libp2pProxy) Close() { p.reset() }

// ─── Signalling message handler ───────────────────────────────────────────────

// Handle processes incoming WebSocket signalling messages from the server-side
// broadcast channel.
func (p *Libp2pProxy) Handle(ctx context.Context, payload []byte) error {
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
	case wire.Libp2pAddresses:
		return decodeAndHandle(ctx, p.logger, payload, eventType, p.handleLibp2pAddresses)
	default:
		p.logger.Debug("unknown wire message", "type", eventType.String())
		return nil
	}
}

// ─── Wire event handlers ──────────────────────────────────────────────────────

func (p *Libp2pProxy) handleJoinRoom(ctx context.Context, player wire.Player) error {
	pid := peerIDStr(player.UserID)
	p.mu.Lock()
	selfID := p.selfID
	currentHostID := p.currentHostID
	p.mu.Unlock()
	if pid == selfID {
		return nil
	}
	p.logger.Info("New player joining", logging.PeerID(pid))

	// If we are the host, ensure we dial into the game server for this peer.
	if currentHostID == selfID {
		if err := p.ensureDialHostForPeer(ctx, pid); err != nil {
			return err
		}
	}
	return nil
}

func (p *Libp2pProxy) handleLeaveRoom(_ context.Context, player wire.Player) error {
	pid := peerIDStr(player.UserID)
	p.mu.Lock()
	selfID := p.selfID
	p.mu.Unlock()
	if selfID == pid {
		return nil
	}

	p.mu.Lock()
	if ps, ok := p.peers[pid]; ok {
		ps.close()
		delete(p.peers, pid)
	}
	p.mu.Unlock()

	p.manager.RemoveByRemoteID(pid)
	return nil
}

func (p *Libp2pProxy) handleHostMigration(_ context.Context, newHost wire.Player) error {
	newHostID := peerIDStr(newHost.UserID)
	p.mu.Lock()
	p.currentHostID = newHostID
	p.mu.Unlock()
	p.logger.Info("Host migration", "newHost", newHostID)
	return nil
}

// handleLibp2pAddresses is called when a remote peer broadcasts its address
// list.  We connect to it and open a game stream.
func (p *Libp2pProxy) handleLibp2pAddresses(ctx context.Context, info wire.Libp2pPeerInfo) error {
	fromID := peerIDStr(info.CreatorID)
	p.mu.Lock()
	selfID := p.selfID
	p.mu.Unlock()
	if fromID == selfID {
		return nil // ignore our own broadcast
	}
	if p.h == nil {
		return nil // host not yet started; will be dialled once we join
	}

	p.logger.Debug("Received libp2p addresses", "from", fromID, "addrs", info.Addresses)

	var addrInfo *peer.AddrInfo
	for _, a := range info.Addresses {
		ma, err := multiaddr.NewMultiaddr(a)
		if err != nil {
			p.logger.Warn("Invalid multiaddr from peer", "addr", a, logging.Error(err))
			continue
		}
		ai, err := peer.AddrInfoFromP2pAddr(ma)
		if err != nil {
			p.logger.Warn("Could not parse peer addr info", "addr", a, logging.Error(err))
			continue
		}
		addrInfo = ai
		break
	}
	if addrInfo == nil {
		return fmt.Errorf("no usable multiaddr from peer %s", fromID)
	}

	// Connect and open a stream if not already connected.
	p.mu.Lock()
	_, alreadyConnected := p.peers[fromID]
	p.mu.Unlock()

	if alreadyConnected {
		return nil
	}

	if err := p.h.Connect(ctx, *addrInfo); err != nil {
		return fmt.Errorf("libp2p connect to %s: %w", fromID, err)
	}

	stream, err := p.h.NewStream(ctx, addrInfo.ID, gameProtocol)
	if err != nil {
		return fmt.Errorf("libp2p open stream to %s: %w", fromID, err)
	}

	ps := &peerStream{peerID: fromID, stream: stream}

	p.mu.Lock()
	// Re-check under the lock: another goroutine may have connected and
	// inserted an entry for the same peer while we were dialing.
	if _, exists := p.peers[fromID]; exists {
		p.mu.Unlock()
		ps.close() // close our redundant stream; the other one is live
		p.logger.Debug("TOCTOU avoided: peer already connected", logging.PeerID(fromID))
		return nil
	}
	p.peers[fromID] = ps
	// Record the libp2p peer ID → game user ID mapping so that
	// handleIncomingStream can store inbound streams under the game user ID.
	p.peerIDToUserID[addrInfo.ID.String()] = fromID
	p.mu.Unlock()

	p.wg.Add(1)
	go p.receiveFromPeer(ps)
	p.logger.Info("libp2p stream opened (outbound)", logging.PeerID(fromID))
	return nil
}

// ─── Incoming stream handler (server side) ────────────────────────────────────

// handleIncomingStream is registered on the libp2p host and called whenever
// a remote peer opens a new stream.
func (p *Libp2pProxy) handleIncomingStream(stream network.Stream) {
	remotePeer := stream.Conn().RemotePeer()
	libp2pID := remotePeer.String()

	// Resolve the libp2p peer ID to a game user ID via the mapping that was
	// populated by handleLibp2pAddresses.  If the mapping is absent we fall
	// back to the libp2p ID string so the connection is not completely lost.
	p.mu.Lock()
	userID, ok := p.peerIDToUserID[libp2pID]
	if !ok {
		userID = libp2pID
		p.logger.Warn("Inbound stream from unknown peer – no game user ID mapping",
			"libp2pID", libp2pID)
	}
	pid := userID

	ps := &peerStream{peerID: pid, stream: stream}
	p.peers[pid] = ps
	p.mu.Unlock()

	p.logger.Info("libp2p stream opened (inbound)", "remotePeer", libp2pID, "userID", pid)
	p.wg.Add(1)
	go p.receiveFromPeer(ps)
}

// ─── Frame-level read loop ────────────────────────────────────────────────────

// receiveFromPeer reads length-prefixed frames from the stream and routes them
// to the appropriate fake host (TCP or UDP).
//
// Frame layout (same convention as the p2p/WebRTC proxy):
//
//	byte 0  : 'T' (TCP) or 'U' (UDP)
//	bytes 1…: raw game payload
func (p *Libp2pProxy) receiveFromPeer(ps *peerStream) {
	defer func() {
		ps.close()
		p.mu.Lock()
		delete(p.peers, ps.peerID)
		p.mu.Unlock()
		p.wg.Done()
	}()

	lenBuf := make([]byte, 4)
	for {
		// Set a read deadline so a silent remote peer does not orphan this
		// goroutine forever.
		if err := ps.stream.SetReadDeadline(time.Now().Add(p.readTimeout)); err != nil {
			p.logger.Debug("stream SetReadDeadline error", logging.PeerID(ps.peerID), logging.Error(err))
		}

		if _, err := readFull(ps.stream, lenBuf); err != nil {
			p.logger.Debug("stream read error (length)", logging.PeerID(ps.peerID), logging.Error(err))
			return
		}
		l := int(uint32(lenBuf[0])<<24 | uint32(lenBuf[1])<<16 | uint32(lenBuf[2])<<8 | uint32(lenBuf[3]))
		if l == 0 || l > 1<<20 {
			p.logger.Warn("implausible frame length", "len", l, logging.PeerID(ps.peerID))
			return
		}

		data := make([]byte, l)
		if _, err := readFull(ps.stream, data); err != nil {
			p.logger.Debug("stream read error (payload)", logging.PeerID(ps.peerID), logging.Error(err))
			return
		}
		if len(data) < 2 {
			continue
		}

		host, ok := p.manager.GetPeerHost(ps.peerID)

		if !ok {
			p.logger.Warn("No fake host for peer", logging.PeerID(ps.peerID))
			continue
		}

		switch data[0] {
		case 'T':
			if host.ProxyTCP != nil {
				if _, err := host.ProxyTCP.Write(data[1:]); err != nil {
					p.logger.Warn("Failed to write TCP data", logging.Error(err))
				}
			}
		case 'U':
			if host.ProxyUDP != nil {
				if _, err := host.ProxyUDP.Write(data[1:]); err != nil {
					p.logger.Warn("Failed to write UDP data", logging.Error(err))
				}
			}
		}
	}
}

// readFull reads exactly len(buf) bytes, retrying on short reads.
func readFull(r network.Stream, buf []byte) (int, error) {
	total := 0
	for total < len(buf) {
		n, err := r.Read(buf[total:])
		total += n
		if err != nil {
			return total, err
		}
	}
	return total, nil
}

// ─── Outbound message helpers ─────────────────────────────────────────────────

// onTCPMessage returns a handler that forwards TCP game data to a peer over the
// libp2p stream.
func (p *Libp2pProxy) onTCPMessage(pid string) func(data []byte) error {
	return func(data []byte) error {
		p.mu.Lock()
		ps, ok := p.peers[pid]
		p.mu.Unlock()
		if !ok {
			p.logger.Debug("No peer for outbound TCP packet", logging.PeerID(pid))
			return nil
		}
		payload := make([]byte, 1+len(data))
		payload[0] = 'T'
		copy(payload[1:], data)
		return ps.send(payload)
	}
}

// onUDPMessage returns a handler that forwards UDP game data to a peer over the
// libp2p stream.
func (p *Libp2pProxy) onUDPMessage(pid string) func(data []byte) error {
	return func(data []byte) error {
		p.mu.Lock()
		ps, ok := p.peers[pid]
		p.mu.Unlock()
		if !ok {
			p.logger.Debug("No peer for outbound UDP packet", logging.PeerID(pid))
			return nil
		}
		payload := make([]byte, 1+len(data))
		payload[0] = 'U'
		copy(payload[1:], data)
		return ps.send(payload)
	}
}

// ensureDialHostForPeer starts forwarding from the local game server (127.0.0.1:6114/6113)
// to the remote peer when we are the current host.
func (p *Libp2pProxy) ensureDialHostForPeer(ctx context.Context, pid string) error {
	ip, err := p.manager.AssignIP(pid)
	if err != nil {
		return fmt.Errorf("assign ip for peer %s: %w", pid, err)
	}
	if _, ok := p.manager.GetPeerHost(pid); ok {
		return nil
	}

	onTCP := p.onTCPMessage(pid)
	onUDP := p.onUDPMessage(pid)
	onDisconnect := func(host *redirect.FakeHost, forced bool) {
		p.logger.Warn("Dial host disconnected", logging.PeerID(pid), "ip", host.AssignedIP, "forced", forced)
		p.manager.StopHost(host)
	}

	host, err := p.manager.StartGuest(ctx, pid, ip, 6114, 6113, onTCP, onUDP, onDisconnect)
	if err != nil {
		return fmt.Errorf("start dial host for %s: %w", pid, err)
	}
	p.logger.Info("Started dial host for peer", logging.PeerID(pid), "ip", host.AssignedIP)
	return nil
}

// ─── Decode helper (mirrors the one in p2p) ───────────────────────────────────

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
