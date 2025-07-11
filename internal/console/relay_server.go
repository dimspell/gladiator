package console

import (
	"bytes"
	"context"
	"crypto/tls"
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
	"github.com/dimspell/gladiator/internal/metrics"
	"github.com/quic-go/quic-go"
)

type RelayStream interface {
	io.Reader
	io.Writer
	CancelRead(code quic.StreamErrorCode)
	CancelWrite(code quic.StreamErrorCode)
}

type RelayConn interface {
	AcceptStream(context.Context) (*quic.Stream, error)
	CloseWithError(code quic.ApplicationErrorCode, msg string) error
	RemoteAddr() net.Addr
}

type RelayPacket struct {
	Type    string `json:"type"` // "join", "leave", ...
	RoomID  string `json:"room"`
	FromID  string `json:"from"`
	ToID    string `json:"to,omitempty"`
	Payload []byte `json:"payload"`
}

type PeerConn struct {
	// ID is a peer identifier.
	ID string

	// RoomID is a game room identifier.
	RoomID string

	// Stream holds a reference to the QUIC R/W streams of the relay server.
	Stream RelayStream

	// Conn holds a reference to the QUIC connection to the relay server.
	Conn RelayConn

	// LastSeen is a timestamp, when the user has sent the packet for the last
	// time.
	LastSeen time.Time

	Session *UserSession
}

type Room struct {
	ID    string
	Peers map[string]*PeerConn
}

type RelayServer struct {
	listener      *quic.Listener
	mu            sync.Mutex
	rooms         map[string]*Room  // keyed by roomID
	peerToRoomIDs map[string]string // key: peerID, value: roomID
	logger        *slog.Logger

	Multiplayer *Multiplayer

	Events chan RelayEvent
}

type RelayEvent struct {
	Type   string
	PeerID string
	RoomID string
}

func NewQUICRelay(addr string, multiplayer *Multiplayer) (*RelayServer, error) {
	tlsConf := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"game-relay"},
		Certificates:       []tls.Certificate{generateSelfSigned()},
	}

	listener, err := quic.ListenAddr(addr, tlsConf, &quic.Config{
		MaxIdleTimeout:  30 * time.Second,
		KeepAlivePeriod: 15 * time.Second,
	})
	if err != nil {
		return nil, err
	}

	return &RelayServer{
		listener:      listener,
		rooms:         make(map[string]*Room),
		peerToRoomIDs: make(map[string]string),
		logger:        slog.With(slog.String("component", "relay")),
		Multiplayer:   multiplayer,
		Events:        make(chan RelayEvent),
	}, nil
}

func (rs *RelayServer) Start(ctx context.Context) error {
	slog.Info("QUIC Relay Server listening", "addr", rs.listener.Addr())

	for {
		conn, err := rs.listener.Accept(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			slog.Warn("Relay server failed to accept", logging.Error(err))
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

// closeStream closes the stream and connection abruptly
func (rs *RelayServer) closeStream(conn RelayConn, stream RelayStream) {
	// TODO: Name and handle various error codes
	var errorCode quic.StreamErrorCode = 0xdead

	stream.CancelWrite(errorCode)
	stream.CancelRead(errorCode)
	_ = conn.CloseWithError(0xdead, "done")

	slog.Info("Closed relay connection", "addr", conn.RemoteAddr())
}

func (rs *RelayServer) handshake(stream RelayStream) (string, string, error) {
	// Initial handshake: receive signed join a packet
	buf := make([]byte, 128)
	n, err := stream.Read(buf)
	if err != nil {
		return "", "", fmt.Errorf("error reading stream: %w", err)
	}

	data, ok := verify(buf[:n])
	if !ok {
		return "", "", fmt.Errorf("signature failed from client")
	}

	var pkt RelayPacket
	if err := json.Unmarshal(data, &pkt); err != nil {
		return "", "", fmt.Errorf("error unmarshaling packet: %w", err)
	}
	if pkt.Type != "join" {
		return "", "", fmt.Errorf("invalid join packet")
	}

	userID, _ := strconv.ParseInt(pkt.FromID, 10, 64)
	if _, ok := rs.Multiplayer.GetUserSession(userID); !ok {
		return "", "", fmt.Errorf("failed to get user session")
	}

	// TODO: Authenticate & authorize

	return pkt.FromID, pkt.RoomID, nil
}

func (rs *RelayServer) joinRoom(roomID, peerID string, conn RelayConn, stream RelayStream) *PeerConn {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	room, ok := rs.rooms[roomID]
	if !ok {
		room = &Room{ID: roomID, Peers: make(map[string]*PeerConn)}
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

	// Notify about the new dynamic joiner
	for _, peer := range room.Peers {
		if peer.ID == peerID {
			continue
		}

		rs.sendSigned(peer.Stream, RelayPacket{
			Type:    "join",
			RoomID:  roomID,
			FromID:  peerID,
			ToID:    peer.ID,
			Payload: nil,
		})
	}

	rs.Events <- RelayEvent{
		Type:   "join",
		PeerID: peerID,
		RoomID: roomID,
	}
	metrics.PeersInRoom.WithLabelValues(roomID).Set(float64(len(room.Peers)))

	return pc
}

func (rs *RelayServer) relayLoop(roomID, peerID string, peer *PeerConn) {
	metrics.ConnectedPeers.Inc()
	defer metrics.ConnectedPeers.Dec()

	buf := make([]byte, 4096)

	for {
		n, err := peer.Stream.Read(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			var se *quic.StreamError
			if ok := errors.As(err, &se); ok && se.ErrorCode == 0xdead {
				break
			}
			rs.logger.Warn("stream error when reading", logging.Error(err), logging.PeerID(peerID))
			break
		}

		data, ok := verify(buf[:n])
		if !ok {
			rs.logger.Warn("signature check failed when reading", logging.PeerID(peerID))
			continue
		}

		peer.LastSeen = time.Now()

		d := json.NewDecoder(bytes.NewReader(data))
		for {
			var pkt RelayPacket
			if err := d.Decode(&pkt); err != nil {
				if err == io.EOF {
					// TODO: Maybe clear(buf) is needed?
					break
				}
				rs.logger.Warn("relay packet unmarshal error", logging.Error(err), logging.PeerID(peerID))
				break
			}
			metrics.PacketIn.Inc()

			// if pkt.Type != "ping" {
			rs.logger.Debug("[RELAY]", "payload", pkt.Payload, "from", pkt.FromID, "to", pkt.ToID, "type", pkt.Type)
			// }

			switch pkt.Type {
			case "udp", "tcp":
				rs.sendTo(pkt.RoomID, pkt.ToID, pkt)

			case "broadcast":
				rs.broadcastFrom(pkt.RoomID, pkt.FromID, pkt)

			case "leave":
				if pkt.FromID != peerID && pkt.RoomID != roomID {
					continue
				}

				slog.Info("leave room", logging.PeerID(peerID))
				rs.leaveRoom(peerID, roomID)
				return
			}
		}
	}

	rs.logger.Info("disconnected from relay", logging.PeerID(peerID))
	rs.leaveRoom(peerID, roomID)
}

func (rs *RelayServer) leaveRoom(peerID, roomID string) {
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

	leaver, _ := room.Peers[peerID]
	if leaver == nil {
		return
	}

	rs.closeStream(leaver.Conn, leaver.Stream)
	delete(room.Peers, peerID)

	rs.Events <- RelayEvent{
		Type:   "leave",
		PeerID: peerID,
		RoomID: roomID,
	}
	rs.logger.Info("peer left room", logging.RoomID(roomID), logging.PeerID(peerID))

	if len(room.Peers) == 0 {
		delete(rs.rooms, roomID)
		rs.logger.Info("room deleted (empty)", logging.RoomID(roomID))
		rs.Events <- RelayEvent{
			Type:   "delete",
			PeerID: peerID,
			RoomID: roomID,
		}

		metrics.ActiveRooms.Dec()
		metrics.PeersInRoom.DeleteLabelValues(roomID)
		return
	}

	metrics.PeersInRoom.WithLabelValues(roomID).Set(float64(len(room.Peers)))
}

func (rs *RelayServer) cleanupPeers() {
	ticker := time.NewTicker(30 * time.Second)

	for now := range ticker.C {
		timeout := now.Add(-5 * time.Minute)
		var toLeave []*PeerConn
		rs.mu.Lock()
		for roomID, room := range rs.rooms {
			for peerID, peer := range room.Peers {
				if timeout.After(peer.LastSeen) {
					rs.logger.Info("peer timed out", logging.PeerID(peerID), logging.RoomID(roomID))
					toLeave = append(toLeave, peer)
				}
			}
		}
		rs.mu.Unlock()

		for _, peer := range toLeave {
			slog.Info("cleaning up users", logging.PeerID(peer.ID), logging.RoomID(peer.RoomID))
			rs.leaveRoom(peer.ID, peer.RoomID)
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

	rs.sendSigned(peer.Stream, pkt)
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
		rs.sendSigned(peer.Stream, pkt)
	}
}

func (rs *RelayServer) sendSigned(stream RelayStream, pkt RelayPacket) {
	data, err := json.Marshal(pkt)
	if err != nil {
		slog.Error("json marshal failed", logging.Error(err))
	}
	// packet := sign(data)
	data = append(data, '\n')
	if _, err := stream.Write(data); err != nil {
		slog.Error("could not write the msg", logging.Error(err))
		return
	}
	metrics.PacketOut.Inc()
}
