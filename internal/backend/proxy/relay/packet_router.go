package relay

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
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
	"github.com/quic-go/quic-go"
)

// RelayStream abstracts a QUIC stream for reading and writing relay packets.
type RelayStream interface {
	io.Reader
	io.Writer
	CancelRead(code quic.StreamErrorCode)
	CancelWrite(code quic.StreamErrorCode)
	Close() error
}

// RelayConn abstracts a QUIC connection for accepting streams and closing with an error.
type RelayConn interface {
	AcceptStream(context.Context) (*quic.Stream, error)
	CloseWithError(code quic.ApplicationErrorCode, msg string) error
}

// PacketRouter manages the routing of packets between the local game client and the remote relay server.
// It handles connection management, host migration, and packet forwarding.
type PacketRouter struct {
	mu        sync.Mutex
	logger    *slog.Logger
	manager   *redirect.HostManager
	session   *bsession.Session
	selfID    string
	relayAddr string

	roomID        string
	currentHostID string
	relayConn     RelayConn
	stream        RelayStream
	pingTicker    *time.Ticker
}

// Reset cleans up all resources, closes connections, stops hosts, and resets the router state.
func (r *PacketRouter) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.pingTicker != nil {
		r.pingTicker.Stop()
	}

	r.disconnect()

	r.manager.StopAll()
	r.roomID = ""
	r.currentHostID = ""
}

// disconnect closes the current stream and relay connection, if any.
func (r *PacketRouter) disconnect() {
	if r.stream != nil {
		r.stream.CancelRead(0xDEAD)
		r.stream.CancelWrite(0xDEAD)
		_ = r.stream.Close()
	}
	if r.relayConn != nil {
		_ = r.relayConn.CloseWithError(0xDEAD, "done")
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
	if r.selfID == peerID {
		return nil
	}

	r.manager.RemoveByRemoteID(peerID)
	return nil
}

func (r *PacketRouter) handleHostMigration(ctx context.Context, player wire.Player) error {
	// oldHostID := r.currentHostID
	newHostID := strconv.Itoa(int(player.UserID))

	r.mu.Lock()
	r.currentHostID = newHostID
	r.mu.Unlock()

	roomID := r.roomID

	if newHostID == r.selfID {
		// I became a host!

		payload := packet.NewHostSwitch(false, net.IPv4(127, 0, 0, 1))
		if err := r.session.SendToGame(packet.HostMigration, payload); err != nil {
			r.logger.Error("failed to send host migration packet", logging.Error(err))
			return fmt.Errorf("failed to send host migration packet: %w", err)
		}

		// Shutdown the previous proxies and save {[peerID: IPv4]} parameters to
		// reuse them.
		rebindHosts := make(map[string]string)
		for peerID, host := range r.manager.PeerHosts {
			rebindHosts[peerID] = host.AssignedIP
			r.manager.StopHost(host)
		}

		// Recreate the proxies to the new host
		for peerID, ip := range rebindHosts {
			onUDPMessage := func(p []byte) error {
				return r.sendPacket(RelayPacket{
					Type:    "udp",
					RoomID:  roomID,
					ToID:    peerID,
					Payload: p,
				})
			}
			onTCPMessage := func(p []byte) error {
				return r.sendPacket(RelayPacket{
					Type:    "tcp",
					RoomID:  roomID,
					ToID:    peerID,
					Payload: p,
				})
			}
			onHostDisconnected := func(host *redirect.FakeHost, forced bool) {
				slog.Warn("Host went offline", logging.PeerID(peerID), "ip", host.AssignedIP, "forced", forced)
				r.stop(host)
				if forced {
					r.disconnect()
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

	// TODO: Wait for completion and register
	time.Sleep(3 * time.Second)

	// Someone else became a host
	host, ok := r.manager.PeerHosts[newHostID]
	if !ok {
		r.logger.Warn("peer not found, nothing to migrate", logging.PeerID(newHostID))
		return nil
	}
	r.manager.StopHost(host)

	onTCPMessage := func(p []byte) error {
		return r.sendPacket(RelayPacket{
			Type:    "tcp",
			RoomID:  roomID,
			ToID:    newHostID,
			Payload: p,
		})
	}
	onUDPMessage := func(p []byte) error {
		return r.sendPacket(RelayPacket{
			Type:    "udp",
			RoomID:  roomID,
			ToID:    newHostID,
			Payload: p,
		})
	}

	onHostDisconnected := func(host *redirect.FakeHost, forced bool) {
		slog.Warn("Host went offline", logging.PeerID(newHostID), "ip", host.AssignedIP, "forced", forced)
		r.stop(host)
		if forced {
			r.disconnect()
			r.Reset()
		}
	}
	var err error
	host, err = r.manager.StartHost(ctx, newHostID, host.AssignedIP, 6114, 6113, onTCPMessage, onUDPMessage, onHostDisconnected)
	if err != nil {
		r.logger.Warn("failed to start host", logging.Error(err), logging.PeerID(newHostID))
		return nil
	}

	payload := packet.NewHostSwitch(true, net.ParseIP(host.AssignedIP))
	if err := r.session.SendToGame(packet.HostMigration, payload); err != nil {
		r.logger.Error("failed to send host migration packet", logging.Error(err))
		return fmt.Errorf("failed to send host migration packet: %w", err)
	}

	return nil
}

// connect establishes a new QUIC connection and stream to the relay server for the given room.
func (r *PacketRouter) connect(ctx context.Context, roomID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	tlsConf := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"game-relay"},
	}
	conn, err := quic.DialAddr(ctx, r.relayAddr, tlsConf, &quic.Config{
		MaxIdleTimeout:  30 * time.Second,
		KeepAlivePeriod: 15 * time.Second,
	})
	if err != nil {
		return fmt.Errorf("quic dial failed: %w", err)
	}
	r.relayConn = conn

	stream, err := conn.OpenStreamSync(ctx)
	if err != nil {
		return fmt.Errorf("quic open stream failed: %w", err)
	}
	r.stream = stream

	// Send "join" packet
	if err := r.sendPacket(RelayPacket{
		Type:   "join",
		RoomID: roomID,
	}); err != nil {
		return fmt.Errorf("send join packet failed: %w", err)
	}

	// Make sure the QUIC send the only packet, not joined with others.
	time.Sleep(100 * time.Millisecond)

	// Start receiver
	go r.receiveLoop(ctx, stream)

	return nil
}

// keepAliveHost periodically sends ping packets to the relay server to keep the connection alive.
func (r *PacketRouter) keepAliveHost(ctx context.Context) {
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
				if err := r.sendPacket(RelayPacket{Type: "ping"}); err != nil {
					r.logger.Error("failed to send ping packet", logging.Error(err))
					r.Reset()
					return
				}
			}
		}
	}(r.pingTicker)
}

// stop stops and cleans up the given fake host.
func (r *PacketRouter) stop(host *redirect.FakeHost) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.manager.StopHost(host)
}

type RelayPacket struct {
	Type    string `json:"type"` // "join", "leave", "tcp", "udp"
	RoomID  string `json:"room"`
	FromID  string `json:"from"`
	ToID    string `json:"to,omitempty"`
	Payload []byte `json:"payload"`
}

// sendPacket marshals and sends a RelayPacket over the current stream.
func (r *PacketRouter) sendPacket(pkt RelayPacket) error {
	if r.stream == nil {
		return fmt.Errorf("stream is nil")
	}

	// Always associate who sending the packet
	pkt.FromID = r.selfID

	data, err := json.Marshal(pkt)
	if err != nil {
		return fmt.Errorf("marshal packet failed: %w", err)
	}

	data = append(data, '\n')

	_, err = r.stream.Write(data)
	if err != nil {
		return fmt.Errorf("write packet failed: %w", err)
	}
	return nil
}

// receiveLoop continuously reads packets from the relay stream and dispatches them for handling.
func (r *PacketRouter) receiveLoop(ctx context.Context, stream *quic.Stream) {
	buf := make([]byte, 4096)
	for {
		select {
		case <-ctx.Done():
			return
		default:
			n, err := stream.Read(buf)
			if err != nil {
				r.logger.Error("received error while reading packet", logging.Error(err), logging.RoomID(r.roomID))
				return
			}
			data := buf[:n]

			d := json.NewDecoder(bytes.NewReader(data))
			for {
				var pkt RelayPacket
				if err := d.Decode(&pkt); err != nil {
					if err == io.EOF {
						break
					}
					r.logger.Warn("failed to unmarshal packet", logging.Error(err))
					r.logger.Debug("invalid packet", slog.Any("data", data))
					continue
				}

				switch pkt.Type {
				case "join":
					r.dynamicJoin(ctx, pkt.RoomID, pkt.FromID)

				case "tcp":
					r.writeTCP(pkt.FromID, pkt)

				case "udp":
					r.writeUDP(pkt.FromID, pkt)

				case "leave":
					r.leaveRoom(pkt.FromID)

				default:
					r.logger.Debug("Unhandled relay packet", slog.Any("packet", pkt))
				}
			}
		}
	}
}

// dynamicJoin handles a new peer dynamically joining the room and sets up the necessary hosts.
func (r *PacketRouter) dynamicJoin(ctx context.Context, roomID string, peerID string) {
	// TODO: There is no probe for checking if it exist?

	ip, err := r.manager.AssignIP(peerID)
	if err != nil {
		r.logger.Warn("failed to assign IP for the peer", logging.Error(err), logging.PeerID(peerID))
		return
	}
	var (
		tcpPort      int
		onTCPMessage func(p []byte) error = nil
		onUDPMessage                      = r.onUDPMessage(roomID, peerID)
	)
	if r.selfID == r.currentHostID {
		tcpPort, onTCPMessage = 6114, r.onTCPMessage(roomID, peerID)
	}

	host, err := r.manager.StartGuest(ctx, peerID, ip, tcpPort, 6113, onTCPMessage, onUDPMessage, r.onFakeHostDisconnect(peerID, ip))
	if err != nil {
		r.logger.Warn("failed to start dial host", logging.Error(err), logging.PeerID(peerID))
		return
	}
	r.manager.SetHost(ip, peerID, host)
}

// leaveRoom removes a peer from the room and cleans up its resources.
func (r *PacketRouter) leaveRoom(peerID string) {
	r.manager.RemoveByRemoteID(peerID)
}

// onFakeHostDisconnect returns a handler for when a fake host disconnects.
func (r *PacketRouter) onFakeHostDisconnect(peerID string, ip string) func(host *redirect.FakeHost, forced bool) {
	return func(host *redirect.FakeHost, forced bool) {
		slog.Warn("Host went offline", logging.PeerID(peerID), "ip", ip, "forced", forced)
		r.stop(host)
	}
}

// onTCPMessage returns a handler for sending TCP packets to a peer via the relay.
func (r *PacketRouter) onTCPMessage(roomID string, peerID string) func(p []byte) error {
	return func(p []byte) error {
		return r.sendPacket(RelayPacket{Type: "tcp", RoomID: roomID, ToID: peerID, Payload: p})
	}
}

// onUDPMessage returns a handler for sending UDP packets to a peer via the relay.
func (r *PacketRouter) onUDPMessage(roomID string, peerID string) func(p []byte) error {
	return func(p []byte) error {
		return r.sendPacket(RelayPacket{Type: "udp", RoomID: roomID, ToID: peerID, Payload: p})
	}
}

// writeTCP writes a TCP packet to the local game client for the given peer.
func (r *PacketRouter) writeTCP(peerID string, pkt RelayPacket) {
	slog.Debug("[TCP] Remote => GameClient", "data", pkt.Payload, logging.PeerID(peerID))

	host, ok := r.manager.PeerHosts[peerID]
	if !ok {
		r.logger.Warn("peer not found, nothing to write", logging.PeerID(peerID))
		return
	}
	if _, err := host.ProxyTCP.Write(pkt.Payload); err != nil {
		r.logger.Warn("failed to write packet", logging.Error(err))
		return
	}
}

// writeUDP writes a UDP packet to the local game client for the given peer.
func (r *PacketRouter) writeUDP(peerID string, pkt RelayPacket) {
	slog.Debug("[UDP] Remote => GameClient", "data", pkt.Payload, logging.PeerID(peerID))

	host, ok := r.manager.PeerHosts[peerID]
	if !ok {
		r.logger.Warn("peer not found, nothing to write", logging.PeerID(peerID))
		return
	}
	if _, err := host.ProxyUDP.Write(pkt.Payload); err != nil {
		r.logger.Warn("failed to write packet", logging.Error(err))
		return
	}
}
