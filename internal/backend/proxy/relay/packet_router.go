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

type PacketRouter struct {
	mu        sync.Mutex
	logger    *slog.Logger
	manager   *HostManager
	session   *bsession.Session
	selfID    string
	relayAddr string

	roomID        string
	currentHostID string
	relayConn     quic.Connection
	stream        quic.Stream
	pingTicker    *time.Ticker
}

func (r *PacketRouter) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.pingTicker != nil {
		r.pingTicker.Stop()
	}

	if r.relayConn != nil {
		_ = r.stream.Close()
		_ = r.relayConn.CloseWithError(0, "done")
	}

	for ipAddress, host := range r.manager.hosts {
		r.manager.stopHost(host, ipAddress)
	}

	slog.Warn("Resetting PacketRouter", "ipPrefix", r.manager.ipPrefix)
	ipPrefix := r.manager.ipPrefix
	r.manager = NewManager(ipPrefix)
	r.roomID = ""
	r.currentHostID = ""
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
	return nil
}

func (r *PacketRouter) handleLeaveRoom(ctx context.Context, player wire.Player) error {
	slog.Error("LEAVE_ROOM - Delete only", "player", player.UserID)

	peerID := remoteID(player.UserID)
	if r.selfID == peerID {
		return nil
	}

	r.manager.RemoveByRemoteID(peerID)
	return nil
}

func (r *PacketRouter) handleHostMigration(ctx context.Context, player wire.Player) error {
	newHostID := strconv.Itoa(int(player.UserID))
	roomID := r.roomID

	if newHostID == r.selfID {
		// I became a host!

		payload := packet.NewHostSwitch(true, net.IPv4(127, 0, 0, 1))

		if err := r.session.SendToGame(packet.HostMigration, payload); err != nil {
			r.logger.Error("failed to send host migration packet", logging.Error(err))
			return nil
		}
		return nil
	}

	// Someone else became a host
	ipAddress, ok := r.manager.peerIPs[newHostID]
	if !ok {
		r.logger.Warn("ip address if peer not found, nothing to migrate", logging.PeerID(newHostID))
		return nil
	}
	host, ok := r.manager.hosts[ipAddress]
	if !ok {
		r.logger.Warn("peer not found, nothing to migrate", logging.PeerID(newHostID))
		return nil
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
	host, err = r.manager.StartListenerHost(ctx, newHostID, ipAddress, 6114, 6113, onTCPMessage, onUDPMessage, nil)
	if err != nil {
		r.logger.Warn("failed to start host", logging.Error(err), logging.PeerID(newHostID))
		return nil
	}

	payload := packet.NewHostSwitch(true, net.ParseIP(ipAddress))
	if err := r.session.SendToGame(packet.HostMigration, payload); err != nil {
		r.logger.Error("failed to send host migration packet", logging.Error(err))
		return nil
	}

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

func (r *PacketRouter) startHostProbe(ctx context.Context, addr string, onDisconnect func()) error {
	return redirect.StartProbeTCP(ctx, addr, onDisconnect)
}

func (r *PacketRouter) stop(host *FakeHost, peerID string, ipAddress string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	slog.Info("Stopping host", logging.PeerID(peerID), "ip", ipAddress, "lastSeen", host.LastSeen)
	r.manager.stopHost(host, ipAddress)
}

var hmacKey = []byte("shared-secret-key")

func sign(data []byte) []byte {
	// mac := hmac.New(sha256.New, hmacKey)
	// mac.Write(data)
	// return append(mac.Sum(nil), data...)
	return data
}

func verify(packet []byte) ([]byte, bool) {
	// if len(packet) < 32 {
	//	return nil, false
	// }
	// sig := packet[:32]
	// data := packet[32:]
	// mac := hmac.New(sha256.New, hmacKey)
	// mac.Write(data)
	// expected := mac.Sum(nil)
	// return data, hmac.Equal(sig, expected)
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
	// packet := sign(data)

	// r.logger.Debug("Sending packet", "fromID", pkt.FromID, "type", pkt.Type, "data", pkt.Payload, "datastr", string(pkt.Payload), "toId", pkt.ToID)

	data = append(data, '\n')

	_, err = r.stream.Write(data)
	if err != nil {
		return fmt.Errorf("write packet failed: %w", err)
	}
	return nil
}

func (r *PacketRouter) receiveLoop(ctx context.Context, stream quic.Stream) {
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

		// r.logger.Debug("Received packet", "data", data, "datastr", string(data))

		d := json.NewDecoder(bytes.NewReader(data))
		for {
			var pkt RelayPacket
			if err := d.Decode(&pkt); err != nil {
				if err == io.EOF {
					break
				}
				r.logger.Warn("failed to unmarshal packet", logging.Error(err))
				break
			}

			switch pkt.Type {
			case "join":
				r.dynamicJoin(ctx, pkt.RoomID, pkt.FromID, pkt)

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
			}
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
	slog.Debug("[TCP] Remote => GameClient", "data", pkt.Payload, logging.PeerID(peerID))

	// r.manager.mu.Lock()
	// defer r.manager.mu.Unlock()

	host, ok := r.manager.peerHosts[peerID]
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
	slog.Debug("[UDP] Remote => GameClient", "data", pkt.Payload, logging.PeerID(peerID))

	// r.manager.mu.Lock()
	// defer r.manager.mu.Unlock()

	host, ok := r.manager.peerHosts[peerID]
	if !ok {
		r.logger.Warn("peer not found, nothing to write", logging.PeerID(peerID))
		return
	}
	if _, err := host.ProxyUDP.Write(pkt.Payload); err != nil {
		r.logger.Warn("failed to write packet", logging.Error(err))
		return
	}
}

func (r *PacketRouter) dynamicJoin(ctx context.Context, roomID string, peerID string, pkt RelayPacket) {
	ip, err := r.manager.assignIP(peerID)
	if err != nil {
		r.logger.Warn("failed to assign IP for the peer ", logging.Error(err), logging.PeerID(peerID))
		return
	}
	var (
		tcpPort      int
		onTCPMessage func(p []byte) error = nil

		onUDPMessage = func(p []byte) error {
			return r.sendPacket(RelayPacket{
				Type:    "udp",
				RoomID:  roomID,
				ToID:    peerID,
				Payload: p,
			})
		}
	)
	if r.selfID == r.currentHostID {
		tcpPort, onTCPMessage = 6114, func(p []byte) error {
			return r.sendPacket(RelayPacket{
				Type:    "tcp",
				RoomID:  roomID,
				ToID:    peerID,
				Payload: p,
			})
		}
	}

	// TODO: It must be local addr
	host, err := r.manager.StartDialHost(peerID, ip, tcpPort, 6113, onTCPMessage, onUDPMessage)
	if err != nil {
		r.logger.Warn("failed to start dial host", logging.Error(err), logging.PeerID(peerID))
		// TODO: Unassign IP address
		return
	}
	r.manager.mu.Lock()
	r.manager.peerIPs[peerID] = ip
	r.manager.ipToPeerID[ip] = peerID
	r.manager.hosts[ip] = host
	r.manager.peerHosts[peerID] = host
	r.manager.mu.Unlock()

	// TODO: There is no probe for checking if it exist?
}

func (r *PacketRouter) leaveRoom(peerID string) {
	r.manager.RemoveByRemoteID(peerID)
}
