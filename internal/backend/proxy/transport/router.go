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

	roomID        string
	currentHostID string
	pingTicker    *time.Ticker
	wg            sync.WaitGroup
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
		logger:    logger,
		selfID:    selfID,
		session:   session,
		manager:   manager,
		transport: transport,
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
	if r.pingTicker != nil {
		r.pingTicker.Stop()
	}

	r.disconnectLocked()

	r.manager.StopAll()
	r.roomID = ""
	r.currentHostID = ""
	r.mu.Unlock()

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

	r.mu.Lock()
	r.currentHostID = newHostID
	roomID := r.roomID
	selfID := r.selfID
	r.mu.Unlock()

	if newHostID == selfID {
		// I became a host!

		payload := packet.NewHostSwitch(false, net.IPv4(127, 0, 0, 1))
		if err := r.session.SendToGame(packet.HostMigration, payload); err != nil {
			r.logger.Error("failed to send host migration packet", logging.Error(err))
			return fmt.Errorf("failed to send host migration packet: %w", err)
		}

		// Shutdown the previous proxies and save {[peerID: IPv4]} parameters to
		// reuse them.
		rebindHosts := make(map[string]string)
		r.manager.ForEachPeerHost(func(peerID string, host *redirect.FakeHost) bool {
			rebindHosts[peerID] = host.AssignedIP
			r.manager.StopHost(host)
			return true
		})

		// Recreate the proxies to the new host
		for peerID, ip := range rebindHosts {
			onUDPMessage := func(p []byte) error {
				return r.SendPacket(RelayPacket{
					Type:    "udp",
					RoomID:  roomID,
					ToID:    peerID,
					Payload: p,
				})
			}
			onTCPMessage := func(p []byte) error {
				return r.SendPacket(RelayPacket{
					Type:    "tcp",
					RoomID:  roomID,
					ToID:    peerID,
					Payload: p,
				})
			}
			onHostDisconnected := func(host *redirect.FakeHost, forced bool) {
				slog.Warn("Host went offline", logging.PeerID(peerID), "ip", host.AssignedIP, "forced", forced)
				r.Stop(host)
				if forced {
					r.Disconnect()
					r.Reset()
				}
			}
			host, err := r.manager.StartGuest(ctx, peerID, ip, 6114, 6113, onTCPMessage, onUDPMessage, onHostDisconnected)
			if err != nil {
				r.logger.Warn("failed to start dial host", logging.Error(err), logging.PeerID(peerID))
				return nil
			}
			r.logger.Info("dial host started", logging.PeerID(peerID), "ip", host.AssignedIP)
		}

		// TODO: Send notice about the completion

		return nil
	}

	// Non-self host migration: defer the delayed work to avoid blocking the event loop
	go func() {
		select {
		case <-time.After(3 * time.Second):
			// Someone else became a host
			host, ok := r.manager.GetPeerHost(newHostID)
			if !ok {
				r.logger.Warn("peer not found, nothing to migrate", logging.PeerID(newHostID))
				return
			}
			r.manager.StopHost(host)

			onTCPMessage := func(p []byte) error {
				return r.SendPacket(RelayPacket{
					Type:    "tcp",
					RoomID:  roomID,
					ToID:    newHostID,
					Payload: p,
				})
			}
			onUDPMessage := func(p []byte) error {
				return r.SendPacket(RelayPacket{
					Type:    "udp",
					RoomID:  roomID,
					ToID:    newHostID,
					Payload: p,
				})
			}

			onHostDisconnected := func(host *redirect.FakeHost, forced bool) {
				slog.Warn("Host went offline", logging.PeerID(newHostID), "ip", host.AssignedIP, "forced", forced)
				r.Stop(host)
				if forced {
					r.Disconnect()
					r.Reset()
				}
			}
			host, err := r.manager.StartHost(context.Background(), newHostID, host.AssignedIP, 6114, 6113, onTCPMessage, onUDPMessage, onHostDisconnected)
			if err != nil {
				r.logger.Warn("failed to start host", logging.Error(err), logging.PeerID(newHostID))
				return
			}

			payload := packet.NewHostSwitch(true, net.ParseIP(host.AssignedIP))
			if err := r.session.SendToGame(packet.HostMigration, payload); err != nil {
				r.logger.Error("failed to send host migration packet", logging.Error(err))
			}
		case <-ctx.Done():
		}
	}()

	return nil
}

// Connect joins the relay infrastructure for the given room via the injected
// PeerTransport and starts the receive loop.
func (r *PacketRouter) Connect(ctx context.Context, roomID string) error {
	if err := r.transport.Join(ctx, roomID); err != nil {
		return fmt.Errorf("failed to join relay: %w", err)
	}

	r.wg.Add(1)
	go r.receiveLoop(ctx)
	return nil
}

// keepAliveHost periodically sends ping packets to the relay server to keep the connection alive.
func (r *PacketRouter) keepAliveHost(ctx context.Context) { //nolint:unused // may be used in future
	r.mu.Lock()
	if r.pingTicker != nil {
		r.pingTicker.Stop()
	}
	r.pingTicker = time.NewTicker(15 * time.Second)
	r.mu.Unlock()

	go func(ticker *time.Ticker) {
		defer func() {
			ticker.Stop()
			r.logger.Debug("keep alive ping stopped")
		}()

		for {
			select {
			case <-ctx.Done():
				return
			case _, ok := <-ticker.C:
				if !ok {
					return
				}

				// Send a packet to the relay server to keep it announced, when
				// playing alone
				if err := r.SendPacket(RelayPacket{Type: "ping"}); err != nil {
					r.logger.Error("failed to send ping packet", logging.Error(err))
					r.Reset()
					return
				}
			}
		}
	}(r.pingTicker)
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

	kind := KindTCP
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
