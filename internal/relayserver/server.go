package relayserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/dimspell/gladiator/internal/app/logger/logging"
	"github.com/dimspell/gladiator/internal/backend/proxy/relay/types"
	"github.com/dimspell/gladiator/internal/metrics"
	"github.com/quic-go/quic-go"
)

// Default timeout constants for peer liveness detection.
const (
	DefaultPingTimeout        = 45 * time.Second // max time since last-seen before declaring dead
	DefaultLivenessInterval   = 30 * time.Second // how often livenessChecker scans rooms
)

type RelayStream interface {
	io.Reader
	io.Writer
	CancelRead(code quic.StreamErrorCode)
	CancelWrite(code quic.StreamErrorCode)
}

type RelayConn interface {
	AcceptStream(context.Context) (RelayStream, error)
	CloseWithError(code quic.ApplicationErrorCode, msg string) error
	RemoteAddr() net.Addr
}

type RelayPacket = types.RelayPacket

type SessionProvider interface {
	SessionExists(id int64) bool
}

func AllowAllSessions() SessionProvider {
	return sessionAllowAll{}
}

type sessionAllowAll struct{}

func (sessionAllowAll) SessionExists(int64) bool { return true }

type PeerConn struct {
	ID       string
	RoomID   string
	Stream   RelayStream
	Conn     RelayConn
	LastSeen time.Time
	writeMu  sync.Mutex
}

type Room struct {
	ID        string
	Peers     map[string]*PeerConn
	CreatedAt time.Time
}

type RelayMetrics interface {
	IncConnectedPeers()
	DecConnectedPeers()
	IncPacketIn()
	IncPacketOut()
	SetPeersInRoom(roomID string, n int)
	IncActiveRooms()
	DecActiveRooms()
	DeletePeersInRoom(roomID string)
}

type defaultRelayMetrics struct{}

func (defaultRelayMetrics) IncConnectedPeers() { metrics.ConnectedPeers.Inc() }
func (defaultRelayMetrics) DecConnectedPeers() { metrics.ConnectedPeers.Dec() }
func (defaultRelayMetrics) IncPacketIn()       { metrics.PacketIn.Inc() }
func (defaultRelayMetrics) IncPacketOut()      { metrics.PacketOut.Inc() }
func (defaultRelayMetrics) SetPeersInRoom(roomID string, n int) {
	metrics.PeersInRoom.WithLabelValues(roomID).Set(float64(n))
}
func (defaultRelayMetrics) IncActiveRooms() { metrics.ActiveRooms.Inc() }
func (defaultRelayMetrics) DecActiveRooms() { metrics.ActiveRooms.Dec() }
func (defaultRelayMetrics) DeletePeersInRoom(roomID string) {
	metrics.PeersInRoom.DeleteLabelValues(roomID)
}

type RelayEventHook func(eventType, peerID, roomID string)

type RelayEvent struct {
	Type   string
	PeerID string
	RoomID string
}

type RelayServerOption func(*RelayServer)

func WithLogger(l *slog.Logger) RelayServerOption {
	return func(rs *RelayServer) { rs.logger = l }
}

func defaultVerify(data []byte) ([]byte, bool) {
	return types.VerifyHMAC(data, types.HMACKey())
}

func WithVerifyFunc(f func([]byte) ([]byte, bool)) RelayServerOption {
	return func(rs *RelayServer) { rs.verifyFunc = f }
}

func WithPingTimeout(d time.Duration) RelayServerOption {
	return func(rs *RelayServer) { rs.PingTimeout = d }
}

func WithLivenessInterval(d time.Duration) RelayServerOption {
	return func(rs *RelayServer) { rs.livenessInterval = d }
}

func WithEventHooks(join, leave, del RelayEventHook) RelayServerOption {
	return func(rs *RelayServer) {
		rs.OnJoin = join
		rs.OnLeave = leave
		rs.OnDelete = del
	}
}

type RelayServer struct {
	listener      RelayListener
	mu            sync.Mutex
	rooms         map[string]*Room
	peerToRoomIDs map[string]string
	logger        *slog.Logger

	Multiplayer SessionProvider

	verifyFunc func([]byte) ([]byte, bool)

	PingTimeout       time.Duration
	livenessInterval  time.Duration

	OnJoin   RelayEventHook
	OnLeave  RelayEventHook
	OnDelete RelayEventHook
}

func NewRelayServer(addr string, multiplayer SessionProvider, opts ...RelayServerOption) (*RelayServer, error) {
	rs := &RelayServer{
		rooms:         make(map[string]*Room),
		peerToRoomIDs: make(map[string]string),
		logger:        slog.With(slog.String("component", "relay")),
		Multiplayer:   multiplayer,
		verifyFunc:       defaultVerify,
		PingTimeout:      DefaultPingTimeout,
		livenessInterval: DefaultLivenessInterval,
	}
	for _, opt := range opts {
		opt(rs)
	}

	if rs.listener == nil {
		listener, err := newQUICListener(addr)
		if err != nil {
			return nil, err
		}
		rs.listener = listener
	}

	return rs, nil
}

func (rs *RelayServer) Addr() net.Addr {
	if rs.listener == nil {
		return nil
	}
	return rs.listener.Addr()
}

func (rs *RelayServer) Start(ctx context.Context) {
	rs.logger.Info("QUIC Relay Server listening", "addr", rs.listener.Addr())

	go rs.livenessChecker(ctx)

	for {
		conn, err := rs.listener.Accept(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}
			rs.logger.Warn("Relay server failed to accept", logging.Error(err))
			continue
		}
		go rs.handleConn(ctx, conn)
	}
}

func (rs *RelayServer) handleConn(ctx context.Context, conn RelayConn) {
	stream, err := conn.AcceptStream(ctx)
	if err != nil {
		rs.logger.Warn("Relay stream accept error", logging.Error(err))
		_ = conn.CloseWithError(0x0, "done")
		return
	}

	peerID, roomID, err := rs.handshake(stream)
	if err != nil {
		rs.logger.Warn("Relay handshake error", logging.Error(err))
		rs.closeStream(conn, stream)
		return
	}

	metrics.PacketIn.Inc()

	peer := rs.joinRoom(roomID, peerID, conn, stream)

	go rs.relayLoop(roomID, peerID, peer)
}

func (rs *RelayServer) closeStream(conn RelayConn, stream RelayStream) {
	var errorCode quic.StreamErrorCode = 0xdead

	stream.CancelWrite(errorCode)
	stream.CancelRead(errorCode)
	_ = conn.CloseWithError(0xdead, "done")

	rs.logger.Info("Closed relay connection", "addr", conn.RemoteAddr())
}

func (rs *RelayServer) handshake(stream RelayStream) (string, string, error) {
	data, err := types.ReadFramed(stream)
	if err != nil {
		return "", "", fmt.Errorf("error reading stream: %w", err)
	}

	payload, ok := rs.verifyFunc(data)
	if !ok {
		return "", "", fmt.Errorf("signature failed from client")
	}

	var pkt RelayPacket
	if err := json.Unmarshal(payload, &pkt); err != nil {
		return "", "", fmt.Errorf("error unmarshaling packet: %w", err)
	}
	if pkt.Type != "join" {
		return "", "", fmt.Errorf("invalid join packet")
	}

	userID, _ := strconv.ParseInt(pkt.FromID, 10, 64)
	if rs.Multiplayer != nil {
		// Short polling retry: the session may not be visible yet if the WS
		// connection just registered it and the goroutine scheduler hasn't
		// yielded. This happens reliably in Docker integration tests where
		// QUIC dial latency (bridge network) races the session map write.
		found := false
		waited := false
		for i := 0; i < 8 && !found; i++ {
			waited = true
			if rs.Multiplayer.SessionExists(userID) {
				found = true
				break
			}
			time.Sleep(15 * time.Millisecond)
		}
		if waited {
			rs.logger.Debug("session check retry", "userID", userID, "found", found)
		}
		if !found {
			return "", "", fmt.Errorf("failed to get user session")
		}
	}

	rs.logger.Debug("handshake success", "peerID", pkt.FromID, "roomID", pkt.RoomID)
	return pkt.FromID, pkt.RoomID, nil
}

func (rs *RelayServer) joinRoom(roomID, peerID string, conn RelayConn, stream RelayStream) *PeerConn {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	room, ok := rs.rooms[roomID]
	if !ok {
		room = &Room{ID: roomID, Peers: make(map[string]*PeerConn), CreatedAt: time.Now().In(time.UTC)}
		rs.rooms[roomID] = room
		rs.logger.Info("new room created", logging.RoomID(roomID), logging.PeerID(peerID))
		metrics.ActiveRooms.Inc()
	}

	pc := &PeerConn{
		ID:       peerID,
		RoomID:   roomID,
		Stream:   stream,
		Conn:     conn,
		LastSeen: time.Now(),
	}
	room.Peers[peerID] = pc
	rs.peerToRoomIDs[peerID] = roomID
	rs.logger.Info("joined room", logging.RoomID(roomID), logging.PeerID(peerID))

	for _, peer := range room.Peers {
		if peer.ID == peerID {
			continue
		}
		rs.sendSigned(peer, RelayPacket{
			Type:    "join",
			RoomID:  roomID,
			FromID:  peerID,
			ToID:    peer.ID,
			Payload: nil,
		})
	}

	metrics.PeersInRoom.WithLabelValues(roomID).Set(float64(len(room.Peers)))

	if rs.OnJoin != nil {
		rs.OnJoin("join", peerID, roomID)
	}

	return pc
}

func (rs *RelayServer) relayLoop(roomID, peerID string, peer *PeerConn) {
	metrics.ConnectedPeers.Inc()
	defer metrics.ConnectedPeers.Dec()

	for {
		raw, err := types.ReadFramed(peer.Stream)
		if err == io.EOF {
			break
		}
		if err != nil {
			var se *quic.StreamError
			if ok := errors.As(err, &se); ok && se.ErrorCode == 0xdead {
				break
			}
			rs.logger.Warn("stream error when reading", logging.Error(err), logging.PeerID(peerID))
			metrics.RelayErrors.WithLabelValues("stream_read").Inc()
			break
		}

		metrics.BytesReceived.Add(float64(len(raw) + 4))

		data, ok := rs.verifyFunc(raw)
		if !ok {
			rs.logger.Warn("signature check failed when reading", logging.PeerID(peerID))
			metrics.PacketsDropped.Inc()
			continue
		}

		rs.mu.Lock()
		peer.LastSeen = time.Now()
		rs.mu.Unlock()

		var pkt RelayPacket
		if err := json.Unmarshal(data, &pkt); err != nil {
			rs.logger.Warn("relay packet unmarshal error", logging.Error(err), logging.PeerID(peerID))
			metrics.RelayErrors.WithLabelValues("unmarshal").Inc()
			continue
		}
		metrics.PacketIn.Inc()
		rs.logger.Debug("[RELAY]", "payload", pkt.Payload, "from", pkt.FromID, "to", pkt.ToID, "type", pkt.Type)
		rs.handlePacket(pkt, peer)
	}

	rs.logger.Info("disconnected from relay", logging.PeerID(peerID))
	rs.LeaveRoom(peerID, roomID)
	metrics.PeerDisconnects.WithLabelValues("relay_loop_exit").Inc()
}

func (rs *RelayServer) handlePacket(pkt RelayPacket, peer *PeerConn) {
	switch pkt.Type {
	case "udp", "tcp":
		rs.sendTo(pkt.RoomID, pkt.ToID, pkt)
	case "ping":
		rs.logger.Debug("ping from peer", logging.PeerID(peer.ID))
	case "leave":
		if pkt.FromID != peer.ID && pkt.RoomID != peer.RoomID {
			return
		}
		rs.logger.Info("leave room", logging.PeerID(peer.ID))
		rs.LeaveRoom(peer.ID, peer.RoomID)
	}
}

func (rs *RelayServer) LeaveRoom(peerID, roomID string) {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	if _, ok := rs.peerToRoomIDs[peerID]; !ok {
		return
	}
	delete(rs.peerToRoomIDs, peerID)

	room, ok := rs.rooms[roomID]
	if !ok {
		return
	}

	leaver := room.Peers[peerID]
	if leaver == nil {
		return
	}

	rs.closeStream(leaver.Conn, leaver.Stream)
	delete(room.Peers, peerID)

	rs.logger.Info("peer left room", logging.RoomID(roomID), logging.PeerID(peerID))
	metrics.PeerDisconnects.WithLabelValues("leave_room").Inc()

	if rs.OnLeave != nil {
		rs.OnLeave("leave", peerID, roomID)
	}

	if len(room.Peers) == 0 {
		metrics.RelayRoomLifetime.Observe(time.Since(room.CreatedAt).Seconds())
		delete(rs.rooms, roomID)
		rs.logger.Info("room deleted (empty)", logging.RoomID(roomID))
		if rs.OnDelete != nil {
			rs.OnDelete("delete", peerID, roomID)
		}

		metrics.ActiveRooms.Dec()
		metrics.PeersInRoom.DeleteLabelValues(roomID)
		return
	}

	metrics.PeersInRoom.WithLabelValues(roomID).Set(float64(len(room.Peers)))
}

func (rs *RelayServer) livenessChecker(ctx context.Context) {
	ticker := time.NewTicker(rs.livenessInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			var toLeave []*PeerConn
			rs.mu.Lock()
			now := time.Now()
			for roomID, room := range rs.rooms {
				for peerID, peer := range room.Peers {
					if now.Sub(peer.LastSeen) > rs.PingTimeout {
						rs.logger.Info("peer timed out (liveness)",
							logging.PeerID(peerID), logging.RoomID(roomID),
							"lastSeen", peer.LastSeen)
						toLeave = append(toLeave, peer)
					}
				}
			}
			rs.mu.Unlock()

			for _, peer := range toLeave {
				rs.LeaveRoom(peer.ID, peer.RoomID)
			}
		}
	}
}

func (rs *RelayServer) sendTo(roomID, peerID string, pkt RelayPacket) {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	room, ok := rs.rooms[roomID]
	if !ok {
		log.Printf("Room %s not found", roomID)
		return
	}

	peer, ok := room.Peers[peerID]
	if !ok {
		log.Printf("Peer %s not in room %s", peerID, roomID)
		return
	}

	rs.sendSigned(peer, pkt)
}

func (rs *RelayServer) broadcastFrom(roomID, fromID string, pkt RelayPacket) {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	room, ok := rs.rooms[roomID]
	if !ok {
		return
	}

	for id, peer := range room.Peers {
		if id == fromID {
			continue
		}
		rs.sendSigned(peer, pkt)
	}
}

func (rs *RelayServer) sendSigned(peer *PeerConn, pkt RelayPacket) {
	peer.writeMu.Lock()
	defer peer.writeMu.Unlock()

	data, err := json.Marshal(pkt)
	if err != nil {
		rs.logger.Error("json marshal failed", logging.Error(err))
		metrics.RelayErrors.WithLabelValues("marshal").Inc()
		return
	}
	if err := types.WriteFramed(peer.Stream, data); err != nil {
		rs.logger.Error("could not write the msg", logging.Error(err))
		metrics.RelayErrors.WithLabelValues("write").Inc()
		return
	}
	metrics.PacketOut.Inc()
	metrics.BytesSent.Add(float64(len(data) + 4))
}

// PeersInRoom returns the peer IDs currently in a room. Only used in tests.
func (rs *RelayServer) PeersInRoom(roomID string) []string {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	room, ok := rs.rooms[roomID]
	if !ok {
		return nil
	}
	peers := make([]string, 0, len(room.Peers))
	for id := range room.Peers {
		peers = append(peers, id)
	}
	return peers
}

// HasPeer reports whether a peer ID is tracked. Only used in tests.
func (rs *RelayServer) HasPeer(peerID string) bool {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	_, ok := rs.peerToRoomIDs[peerID]
	return ok
}