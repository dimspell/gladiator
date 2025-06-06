package relay

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/dimspell/gladiator/internal/app/logger/logging"
	"github.com/dimspell/gladiator/internal/backend/bsession"
	"github.com/dimspell/gladiator/internal/backend/packet"
	"github.com/dimspell/gladiator/internal/wire"
	"github.com/quic-go/quic-go"
)

type PacketRouter struct {
	mu        sync.Mutex
	logger    *slog.Logger
	manager   *HostManager
	session   *bsession.Session
	selfID    string
	relayAddr string

	isHost        bool
	currentHostID string
	relayConn     quic.Connection
	stream        quic.Stream
}

func (r *PacketRouter) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.relayConn != nil {
		_ = r.relayConn.CloseWithError(0, "done")
	}

	for ipAddress, host := range r.manager.hosts {
		r.manager.stopHost(host, ipAddress)
	}

	r.manager = NewManager()
}

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
	// Handled in QUIC stream
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
	// Handled in QUIC stream
	return nil
}

func (r *PacketRouter) connect(ctx context.Context, roomID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	tlsConf := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"game-relay"},
	}
	conn, err := quic.DialAddr(ctx, r.relayAddr, tlsConf, &quic.Config{
		MaxIdleTimeout:  300 * time.Second,
		KeepAlivePeriod: 250 * time.Second,
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
	r.sendPacket(RelayPacket{
		Type:   "join",
		RoomID: roomID,
	})

	// Start receiver
	go r.receiveLoop(stream)

	return nil
}

var hmacKey = []byte("shared-secret-key")

func sign(data []byte) []byte {
	//mac := hmac.New(sha256.New, hmacKey)
	//mac.Write(data)
	//return append(mac.Sum(nil), data...)
	return data
}

func verify(packet []byte) ([]byte, bool) {
	//if len(packet) < 32 {
	//	return nil, false
	//}
	//sig := packet[:32]
	//data := packet[32:]
	//mac := hmac.New(sha256.New, hmacKey)
	//mac.Write(data)
	//expected := mac.Sum(nil)
	//return data, hmac.Equal(sig, expected)
	return packet, true
}

type RelayPacket struct {
	Type    string `json:"type"` // "join", "leave", "data", "broadcast", "migrate", "tcp", "udp"
	RoomID  string `json:"room"`
	FromID  string `json:"from"`
	ToID    string `json:"to,omitempty"`
	Payload []byte `json:"payload"`
}

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
	packet := sign(data)
	_, err = r.stream.Write(packet)
	if err != nil {
		return fmt.Errorf("write packet failed: %w", err)
	}
	return nil
}

func (r *PacketRouter) receiveLoop(stream quic.Stream) {
	buf := make([]byte, 4096)
	for {
		n, err := stream.Read(buf)
		if err == io.EOF {
			return
		}
		if err != nil {
			r.logger.Error("received error while reading packet", logging.Error(err))
			return
		}
		data, ok := verify(buf[:n])
		if !ok {
			r.logger.Warn("received invalid packet - signature is incorrect")
			continue
		}
		var pkt RelayPacket
		if err := json.Unmarshal(data, &pkt); err != nil {
			r.logger.Warn("failed to unmarshal packet", logging.Error(err))
			continue
		}

		if pkt.ToID != r.selfID {
			r.logger.Warn("received packet from other peer does not match our own peer")
			continue
		}

		log.Printf("Received from %s: %s", pkt.FromID, string(pkt.Payload))

		switch pkt.Type {
		case "join":
			r.dynamicJoin(pkt.RoomID, pkt.FromID, pkt)

		case "data":
			r.readMessage(pkt.FromID, pkt)

		case "tcp":
			r.writeTCP(pkt.FromID, pkt)

		case "udp":
			r.writeUDP(pkt.FromID, pkt)

		case "broadcast":
			r.readBroadcast(pkt.FromID, pkt)

		case "leave":
			r.leaveRoom(pkt.FromID)

		case "migrate":
			r.hostMigration(pkt.RoomID, pkt)
		}
	}
}
func (r *PacketRouter) readBroadcast(fromID string, pkt RelayPacket) {
	r.logger.Info("broadcast packet received", slog.String("fromID", fromID), slog.String("payload", string(pkt.Payload)))
}

func (r *PacketRouter) readMessage(fromID string, pkt RelayPacket) {
	r.logger.Info("data packet received", slog.String("fromID", fromID), slog.String("payload", string(pkt.Payload)))
}

func (r *PacketRouter) writeTCP(peerID string, pkt RelayPacket) {
	host, ok := r.manager.hosts[peerID]
	if !ok {
		r.logger.Warn("peer not found, nothing to write", logging.PeerID(peerID))
		return
	}
	if _, err := host.ProxyTCP.Write(pkt.Payload); err != nil {
		r.logger.Warn("failed to write packet", logging.Error(err))
		return
	}
}

func (r *PacketRouter) writeUDP(peerID string, pkt RelayPacket) {
	host, ok := r.manager.hosts[peerID]
	if !ok {
		r.logger.Warn("peer not found, nothing to write", logging.PeerID(peerID))
		return
	}
	if _, err := host.ProxyUDP.Write(pkt.Payload); err != nil {
		r.logger.Warn("failed to write packet", logging.Error(err))
		return
	}
}

func (r *PacketRouter) dynamicJoin(roomID string, peerID string, pkt RelayPacket) {
	ip, err := r.manager.assignIP(peerID)
	if err != nil {
		r.logger.Warn("failed to assign IP for the peer ", logging.Error(err), logging.PeerID(peerID))
		return
	}

	onTCPMessage := func(p []byte) error {
		return r.sendPacket(RelayPacket{
			Type:    "tcp",
			RoomID:  roomID,
			ToID:    peerID,
			Payload: p,
		})
	}
	onUDPMessage := func(p []byte) error {
		return r.sendPacket(RelayPacket{
			Type:    "udp",
			RoomID:  roomID,
			ToID:    peerID,
			Payload: p,
		})
	}

	if _, err := r.manager.StartGuestHost(ip, 0, 6113, onTCPMessage, onUDPMessage); err != nil {
		r.logger.Warn("failed to start guest host", logging.Error(err), logging.PeerID(peerID))
		// TODO: Unassign IP address
		return
	}
}

func (r *PacketRouter) leaveRoom(peerID string) {
	r.manager.RemoveByRemoteID(peerID)
}

func (r *PacketRouter) hostMigration(roomID string, pkt RelayPacket) {
	newHostID := string(pkt.Payload)

	if newHostID == r.selfID {
		// I became a host!

		payload := make([]byte, 8)
		copy(payload[0:4], []byte{1, 0, 0, 0})
		copy(payload[4:], net.IPv4(127, 0, 0, 1).To4())

		if err := r.session.SendToGame(packet.HostMigration, payload); err != nil {
			r.logger.Error("failed to send host migration packet", logging.Error(err))
			return
		}
		return
	}

	// Someone else became a host

	host, ok := r.manager.hosts[newHostID]
	if !ok {
		r.logger.Warn("peer not found, nothing to migrate", logging.PeerID(newHostID))
		return
	}
	ipAddress, ok := r.manager.peerIPs[newHostID]
	if !ok {
		r.logger.Warn("ip address if peer not found, nothing to migrate", logging.PeerID(newHostID))
		return
	}
	r.manager.stopHost(host, ipAddress)

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

	var err error
	host, err = r.manager.StartHost(ipAddress, 6114, 6113, onTCPMessage, onUDPMessage)
	if err != nil {
		r.logger.Warn("failed to start host", logging.Error(err), logging.PeerID(newHostID))
		return
	}

	payload := make([]byte, 8)
	copy(payload[0:4], []byte{1, 0, 0, 0})
	copy(payload[4:], ipAddress)

	if err := r.session.SendToGame(packet.HostMigration, payload); err != nil {
		r.logger.Error("failed to send host migration packet", logging.Error(err))
		return
	}
}
