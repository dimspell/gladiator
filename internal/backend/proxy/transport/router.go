package transport

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/dimspell/gladiator/internal/app/logger/logging"
	"github.com/dimspell/gladiator/internal/backend/bsession"
	"github.com/dimspell/gladiator/internal/backend/packet"
	"github.com/dimspell/gladiator/internal/backend/redirect"
	"github.com/dimspell/gladiator/internal/wire"
)

// DefaultHostPingInterval is how often the host sends a keepalive ping.
const DefaultHostPingInterval = 15 * time.Second

// PacketRouter manages the routing of packets between the local game client and the
// remote peer network (relay or WebRTC). It depends only on the PeerTransport
// port, so the same dispatch logic serves every proxy mode.
type PacketRouter struct {
	mu        sync.Mutex
	logger    *slog.Logger
	manager   *redirect.HostManager
	session   *bsession.Session
	selfID    string
	transport PeerTransport

	roomID           string
	currentHostID    string
	pingTicker       *time.Ticker
	hostPingInterval time.Duration
	wg               sync.WaitGroup

	// loopCancel cancels the receive loop's context. The loop is owned by the
	// router (not the caller's context) so it survives request-scoped contexts.
	loopCancel context.CancelFunc

	// keepAliveCancel cancels the keep-alive ping goroutine. It is created when
	// Connect is called (all peers send pings) and cancelled in Reset.
	keepAliveCancel context.CancelFunc

	// hostPingCancel cancels the host ping goroutine's context. Created when
	// StartHostPing is called and cancelled in Reset.
	hostPingCancel context.CancelFunc
}

// NewPacketRouter constructs a PacketRouter. The manager and transport are
// injected so callers (relay, WebRTC) control lifecycle and test seams.
func NewPacketRouter(
	logger *slog.Logger,
	selfID string,
	session *bsession.Session,
	manager *redirect.HostManager,
	transport PeerTransport,
) *PacketRouter {
	return &PacketRouter{
		logger:           logger,
		selfID:           selfID,
		session:          session,
		manager:          manager,
		transport:        transport,
		hostPingInterval: DefaultHostPingInterval,
	}
}

// Manager returns the underlying HostManager.
func (r *PacketRouter) Manager() *redirect.HostManager { return r.manager }

// SetManager replaces the HostManager (used by tests to inject a capture factory).
func (r *PacketRouter) SetManager(m *redirect.HostManager) { r.manager = m }

// Logger returns the router's logger.
func (r *PacketRouter) Logger() *slog.Logger { return r.logger }

// Transport returns the configured PeerTransport.
func (r *PacketRouter) Transport() PeerTransport { return r.transport }

// SelfID returns the local peer ID.
func (r *PacketRouter) SelfID() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.selfID
}

// SetSelfID sets the local peer ID.
func (r *PacketRouter) SetSelfID(v string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.selfID = v
}

// CurrentHostID returns the current host peer ID.
func (r *PacketRouter) CurrentHostID() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.currentHostID
}

// SetCurrentHostID sets the current host peer ID.
func (r *PacketRouter) SetCurrentHostID(v string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.currentHostID = v
}

// RoomID returns the active room ID.
func (r *PacketRouter) RoomID() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.roomID
}

// SetRoomID sets the active room ID.
func (r *PacketRouter) SetRoomID(v string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.roomID = v
}

// SetRoomState atomically records the room, self, and host IDs. It replaces the
// manual mutex locking that callers previously did inline.
func (r *PacketRouter) SetRoomState(roomID, selfID, hostID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.roomID = roomID
	r.selfID = selfID
	r.currentHostID = hostID
}

// Reset cleans up all resources, closes connections, stops hosts, and resets the router state.
func (r *PacketRouter) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.pingTicker != nil {
		r.pingTicker.Stop()
	}
	r.stopKeepAliveLocked()
	if r.hostPingCancel != nil {
		r.hostPingCancel()
		r.hostPingCancel = nil
	}

	r.disconnectLocked()

	if r.manager != nil {
		r.manager.StopAll()
	}
	r.roomID = ""
	r.currentHostID = ""

	// Wait for receiveLoop to finish (stream is closed, so Read should return quickly)
	waitCh := make(chan struct{})
	go func() {
		r.wg.Wait()
		close(waitCh)
	}()
	select {
	case <-waitCh:
	case <-time.After(5 * time.Second):
		r.logger.Warn("timed out waiting for receiveLoop to exit")
	}
}

// Disconnect closes the current transport without acquiring the lock.
func (r *PacketRouter) Disconnect() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.disconnectLocked()
}

// disconnectLocked closes the current transport without acquiring the lock.
// Caller must hold r.mu.
func (r *PacketRouter) disconnectLocked() {
	// Cancel the receive loop first so it unblocks even if the transport's Recv
	// does not return promptly on Close.
	if r.loopCancel != nil {
		r.loopCancel()
		r.loopCancel = nil
	}
	if r.transport != nil {
		_ = r.transport.Close()
	}
}

// Handle processes an incoming payload from the relay and dispatches it to the appropriate handler.
func (r *PacketRouter) Handle(ctx context.Context, payload []byte) error {
	eventType := wire.ParseEventType(payload)

	switch eventType {
	case wire.JoinRoom:
		return decodeAndHandle(ctx, r.logger, payload, eventType, r.handleJoinRoom)
	case wire.LeaveRoom, wire.LeaveLobby:
		return decodeAndHandle(ctx, r.logger, payload, eventType, r.handleLeaveRoom)
	case wire.HostMigration:
		return decodeAndHandle(ctx, r.logger, payload, eventType, r.handleHostMigration)
	default:
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

// handleJoinRoom handles the event when new dynamic joiner has arrived (player who connected mid-game)
func (r *PacketRouter) handleJoinRoom(ctx context.Context, player wire.Player) error {
	return nil
}

func (r *PacketRouter) handleLeaveRoom(ctx context.Context, player wire.Player) error {
	peerID := remoteID(player.UserID)

	r.mu.Lock()
	selfID := r.selfID
	r.mu.Unlock()

	if selfID == peerID {
		return nil
	}

	r.manager.RemoveByRemoteID(peerID)
	return nil
}

func (r *PacketRouter) handleHostMigration(ctx context.Context, player wire.Player) error {
	newHostID := strconv.Itoa(int(player.UserID))
	newHostIP := player.IPAddress

	r.mu.Lock()
	prevHostID := r.currentHostID
	r.currentHostID = newHostID
	roomID := r.roomID
	selfID := r.selfID
	r.mu.Unlock()

	// Null IP means refresh only, not a migration.
	if newHostIP == "" || newHostIP == "0.0.0.0" {
		r.logger.Info("host migration: refresh", "host", newHostID)
		if host, ok := r.manager.GetPeerHost(newHostID); ok {
			r.manager.StopHost(host)
		}
		return nil
	}

	if newHostID == selfID {
		if prevHostID == selfID {
			r.logger.Info("already host, refresh ping", "room", roomID)
			r.StartHostPing()
			return nil
		}
		r.logger.Info("became host", "room", roomID)
		r.StartHostPing()

		payload := packet.NewHostSwitch(false, net.IPv4(127, 0, 0, 1))
		if err := r.session.SendToGame(packet.HostMigration, payload); err != nil {
			r.logger.Error("failed to send host migration", logging.Error(err))
			return fmt.Errorf("failed to send host migration: %w", err)
		}

		// Tear down the old listener and rebind surviving peer guests:
		// stop old FakeHosts and recreate as UDP-only guests (TCP relay via
		// relay path).
		rebindHosts := make(map[string]string)
		r.manager.ForEachPeerHost(func(peerID string, host *redirect.FakeHost) bool {
			rebindHosts[peerID] = host.AssignedIP
			r.manager.StopHost(host)
			return true
		})
		r.logger.Info("host migration: rebind surviving peers as guests", "count", len(rebindHosts), "peers", rebindHosts)

		for peerID, ip := range rebindHosts {
			onUDPMessage := func(p []byte) error {
				return r.SendPacket(RelayPacket{Type: "udp", RoomID: roomID, ToID: peerID, Payload: p})
			}
			onTCPMessage := func(p []byte) error {
				return r.SendPacket(RelayPacket{Type: "tcp", RoomID: roomID, ToID: peerID, Payload: p})
			}
			onHostDisconnected := func(host *redirect.FakeHost, forced bool) {
				slog.Warn("Host went offline", logging.PeerID(peerID), "ip", host.AssignedIP, "forced", forced)
				r.Stop(host)
				if forced {
					r.Disconnect()
					r.Reset()
				}
			}
			// TCP port 0: no TCP listener for non-host peers after migration;
			// the new host's game client is not yet listening.
			host, err := r.manager.StartGuest(ctx, peerID, ip, 0, 6113, onTCPMessage, onUDPMessage, onHostDisconnected)
			if err != nil {
				r.logger.Warn("failed to start dial host for rebind", logging.Error(err), logging.PeerID(peerID))
				continue
			}
			r.logger.Info("dial host re-bound", logging.PeerID(peerID), "ip", host.AssignedIP)
		}

		return nil
	}

	// Peer host migration (deferred to avoid blocking).
	go func() {
		select {
		case <-time.After(3 * time.Second):
			r.mu.Lock()
			current := r.currentHostID
			r.mu.Unlock()
			if current != newHostID {
				r.logger.Info("host migration superseded", "expected", newHostID, "current", current)
				return
			}

			host, ok := r.manager.GetPeerHost(newHostID)
			if !ok {
				r.logger.Warn("unknown host, waiting", logging.PeerID(newHostID))
				_ = r.session.SendToGame(packet.ReceiveMessage, packet.NewAdminNotice("system", "Host election: unknown host, please wait"))
				return
			}

			r.manager.StopHost(host)

			onTCPMessage := func(p []byte) error {
				return r.SendPacket(RelayPacket{Type: "tcp", RoomID: roomID, ToID: newHostID, Payload: p})
			}
			onUDPMessage := func(p []byte) error {
				return r.SendPacket(RelayPacket{Type: "udp", RoomID: roomID, ToID: newHostID, Payload: p})
			}
			onHostDisconnected := func(host *redirect.FakeHost, forced bool) {
				slog.Warn("Host went offline", logging.PeerID(newHostID), "ip", host.AssignedIP, "forced", forced)
				r.Stop(host)
				if forced {
					r.Disconnect()
					r.Reset()
				}
			}
			newHost, err := r.manager.StartHost(context.Background(), newHostID, host.AssignedIP, 6114, 6113, onTCPMessage, onUDPMessage, onHostDisconnected)
			if err != nil {
				r.logger.Warn("failed to start host for peer", logging.Error(err), logging.PeerID(newHostID))
				return
			}

			// Send HostMigration to the local game client so its 0x47FF handler
			// takes the peer branch (connect to host:6114).
			payload := packet.NewHostSwitch(true, net.ParseIP(newHost.AssignedIP))
			if err := r.session.SendToGame(packet.HostMigration, payload); err != nil {
				r.logger.Error("failed to send host migration packet (peer)", logging.Error(err))
			} else {
				r.logger.Info("host migration: peer host established", "newHostID", newHostID, "ip", newHost.AssignedIP)
			}
		case <-ctx.Done():
		}
	}()

	return nil
}

// Connect joins the relay infrastructure for the given room via the injected
// PeerTransport, starts the receive loop, and begins periodic keep-alive pings
// so the relay server does not time out this peer.
func (r *PacketRouter) Connect(ctx context.Context, roomID string) error {
	r.mu.Lock()
	// Cancel any previously running receive loop before (re)joining, so a
	// reconnect cannot leave a stale loop reading from the old transport.
	if r.loopCancel != nil {
		r.loopCancel()
		r.loopCancel = nil
	}
	if err := r.transport.Join(ctx, roomID); err != nil {
		r.mu.Unlock()
		return fmt.Errorf("failed to join relay: %w", err)
	}
	// The receive loop must outlive the caller's context (e.g. an HTTP request
	// scope). It is owned by the router and torn down via loopCancel on
	// disconnect/reset.
	loopCtx, cancel := context.WithCancel(context.Background())
	r.loopCancel = cancel
	r.mu.Unlock()

	r.wg.Add(1)
	go r.receiveLoop(loopCtx)
	r.startKeepAlive()
	return nil
}

// startKeepAlive sends periodic ping packets to the relay server so it does
// not disconnect this peer due to liveness timeout. Every peer in the room
// needs this, not just the host. The goroutine is stopped in Reset.
func (r *PacketRouter) startKeepAlive() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.stopKeepAliveLocked()

	ctx, cancel := context.WithCancel(context.Background())
	r.keepAliveCancel = cancel
	interval := r.hostPingInterval
	if interval <= 0 {
		interval = DefaultHostPingInterval
	}
	ticker := time.NewTicker(interval)

	r.wg.Add(1)
	go func() {
		defer ticker.Stop()
		defer r.wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := r.SendPacket(RelayPacket{Type: "ping"}); err != nil {
					r.logger.Error("keep-alive ping failed", logging.Error(err))
					return
				}
			}
		}
	}()
}

// stopKeepAliveLocked stops the keep-alive goroutine. Must be called with r.mu held.
func (r *PacketRouter) stopKeepAliveLocked() {
	if r.keepAliveCancel != nil {
		r.keepAliveCancel()
		r.keepAliveCancel = nil
	}
}

// StartHostPing begins sending periodic ping packets to the relay server while
// this peer is the room host. It cancels any previous ping goroutine and creates
// a new one. The goroutine is stopped when Reset is called.
func (r *PacketRouter) StartHostPing() {
	r.mu.Lock()
	if r.hostPingCancel != nil {
		r.hostPingCancel()
	}
	if r.pingTicker != nil {
		r.pingTicker.Stop()
	}
	ctx, cancel := context.WithCancel(context.Background())
	r.hostPingCancel = cancel
	r.pingTicker = time.NewTicker(r.hostPingInterval)
	ticker := r.pingTicker
	r.mu.Unlock()

	go func() {
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				r.logger.Debug("host ping stopped")
				return
			case <-ticker.C:
				if err := r.SendPacket(RelayPacket{Type: "ping"}); err != nil {
					r.logger.Error("failed to send ping packet", logging.Error(err))
					return
				}
			}
		}
	}()
}

// Stop stops and cleans up the given fake host.
func (r *PacketRouter) Stop(host *redirect.FakeHost) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.manager.StopHost(host)
}

// SendPacket marshals and sends a RelayPacket over the transport.
func (r *PacketRouter) SendPacket(pkt RelayPacket) error {
	// Always associate who is sending the packet.
	r.mu.Lock()
	pkt.FromID = r.selfID
	r.mu.Unlock()

	var kind PacketKind
	switch pkt.Type {
	case "udp":
		kind = KindUDP
	case "tcp":
		kind = KindTCP
	case "ping":
		kind = KindPing
	default:
		return fmt.Errorf("unsupported relay packet type %q", pkt.Type)
	}

	return r.transport.Send(context.Background(), TransportPacket{
		ToID:   pkt.ToID,
		RoomID: pkt.RoomID,
		Kind:   kind,
		Data:   pkt.Payload,
	})
}

// receiveLoop continuously reads packets from the transport and dispatches
// them. It runs until the transport closes or ctx is cancelled.
func (r *PacketRouter) receiveLoop(ctx context.Context) {
	defer r.wg.Done()

	for {
		pkt, err := r.transport.Recv(ctx)
		if err != nil {
			if err != io.EOF && err != context.Canceled {
				r.logger.Error("transport recv error", logging.Error(err))
			}
			return
		}
		r.onTransportPacket(pkt)
	}
}

// onTransportPacket dispatches a transport-agnostic packet to the appropriate
// handler. This is the single dispatch path shared by every PeerTransport
// (relay/QUIC, WebRTC, or in-memory).
func (r *PacketRouter) onTransportPacket(pkt TransportPacket) {
	switch pkt.Kind {
	case KindJoin:
		r.DynamicJoin(context.Background(), pkt.RoomID, pkt.FromID)
	case KindTCP:
		r.writeTCP(pkt.FromID, pkt.Data)
	case KindUDP:
		r.writeUDP(pkt.FromID, pkt.Data)
	case KindLeave:
		r.leaveRoom(pkt.FromID)
	case KindPing:
		// keep-alive; nothing to forward to the game client
	}
}

// DynamicJoin handles a new peer dynamically joining the room and sets up the
// necessary dial host (StartGuest) so the local game client can exchange traffic
// with that peer. It is exported so proxy modes that learn about new peers through
// signaling (WebRTC) rather than an inbound transport packet can trigger it.
func (r *PacketRouter) DynamicJoin(ctx context.Context, roomID string, peerID string) {
	r.mu.Lock()
	selfID := r.selfID
	currentHostID := r.currentHostID
	r.mu.Unlock()

	// Only the host needs to create StartGuest dialers to forward
	// game-client data to the new peer. Non-host peers already have
	// a receive-side FakeHost from the initial JoinGame/StartHost path.
	if selfID == currentHostID {
		ip, err := r.manager.AssignIP(peerID)
		if err != nil {
			r.logger.Warn("failed to assign IP for the peer", logging.Error(err), logging.PeerID(peerID))
			return
		}

		host, err := r.manager.StartGuest(ctx, peerID, ip, 6114, 6113,
			r.OnTCPMessage(roomID, peerID), r.OnUDPMessage(roomID, peerID),
			r.onFakeHostDisconnect(peerID, ip))
		if err != nil {
			r.logger.Warn("failed to start dial host", logging.Error(err), logging.PeerID(peerID))
			return
		}
		r.manager.SetHost(ip, peerID, host)
	}
}

// leaveRoom removes a peer from the room and cleans up its resources.
func (r *PacketRouter) leaveRoom(peerID string) {
	r.manager.RemoveByRemoteID(peerID)
}

// onFakeHostDisconnect returns a handler for when a fake host disconnects.
func (r *PacketRouter) onFakeHostDisconnect(peerID string, ip string) func(host *redirect.FakeHost, forced bool) {
	return func(host *redirect.FakeHost, forced bool) {
		slog.Warn("Host went offline", logging.PeerID(peerID), "ip", ip, "forced", forced)
		r.Stop(host)
	}
}

// OnTCPMessage returns a handler for sending TCP packets to a peer via the transport.
func (r *PacketRouter) OnTCPMessage(roomID string, peerID string) func(p []byte) error {
	return func(p []byte) error {
		return r.SendPacket(RelayPacket{Type: "tcp", RoomID: roomID, ToID: peerID, Payload: p})
	}
}

// OnUDPMessage returns a handler for sending UDP packets to a peer via the transport.
func (r *PacketRouter) OnUDPMessage(roomID string, peerID string) func(p []byte) error {
	return func(p []byte) error {
		return r.SendPacket(RelayPacket{Type: "udp", RoomID: roomID, ToID: peerID, Payload: p})
	}
}

// writeTCP writes a TCP payload to the local game client for the given peer.
func (r *PacketRouter) writeTCP(peerID string, payload []byte) {
	slog.Debug("[TCP] Remote => GameClient", "data", payload, logging.PeerID(peerID))

	host, ok := r.manager.GetPeerHost(peerID)
	if !ok {
		r.logger.Warn("peer not found, nothing to write", logging.PeerID(peerID))
		return
	}
	if host.ProxyTCP == nil {
		r.logger.Warn("tcp proxy not available for peer", logging.PeerID(peerID))
		return
	}
	if _, err := host.ProxyTCP.Write(payload); err != nil {
		r.logger.Warn("failed to write packet", logging.Error(err))
		return
	}
}

// writeUDP writes a UDP payload to the local game client for the given peer.
func (r *PacketRouter) writeUDP(peerID string, payload []byte) {
	slog.Debug("[UDP] Remote => GameClient", "data", payload, logging.PeerID(peerID))

	host, ok := r.manager.GetPeerHost(peerID)
	if !ok {
		r.logger.Warn("peer not found, nothing to write", logging.PeerID(peerID))
		return
	}
	if _, err := host.ProxyUDP.Write(payload); err != nil {
		r.logger.Warn("failed to write packet", logging.Error(err))
		return
	}
}

// remoteID converts a user/session ID into the peer ID string used on the wire.
func remoteID(i int64) string { return fmt.Sprintf("%d", i) }
