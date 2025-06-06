package console

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"log"
	"log/slog"
	"sync"
	"time"

	"github.com/dimspell/gladiator/internal/app/logger/logging"
	quic "github.com/quic-go/quic-go"
)

type RelayPacket struct {
	Type    string `json:"type"` // "join", "leave", "data", "broadcast", "migrate", "tcp", "udp"
	RoomID  string `json:"room"` // new!
	FromID  string `json:"from"`
	ToID    string `json:"to,omitempty"`
	Payload []byte `json:"payload"`
}

type PeerConn struct {
	ID     string
	RoomID string
	Role   string // "host" or "guest"
	Stream quic.Stream

	// ConnectedAt time.Time
	LastSeen time.Time
}

type Room struct {
	ID    string
	Peers map[string]*PeerConn
}

type RelayServer struct {
	listener    *quic.Listener
	mu          sync.Mutex
	rooms       map[string]*Room  // keyed by roomID
	peerToRooms map[string]string // key: peerID, value: roomID
	logger      *slog.Logger
}

func NewQUICRelay(addr string) (*RelayServer, error) {
	tlsConf := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"game-relay"},
		Certificates:       []tls.Certificate{generateSelfSigned()},
	}

	listener, err := quic.ListenAddr(addr, tlsConf, &quic.Config{
		MaxIdleTimeout:  300 * time.Second,
		KeepAlivePeriod: 250 * time.Second,
	})
	if err != nil {
		return nil, err
	}

	return &RelayServer{
		listener:    listener,
		rooms:       make(map[string]*Room),
		peerToRooms: make(map[string]string),
		logger:      slog.With(slog.String("component", "relay")),
	}, nil
}

func (rs *RelayServer) Start(ctx context.Context) error {
	slog.Info("QUIC Relay Server listening", "addr", rs.listener.Addr())

	for {
		conn, err := rs.listener.Accept(ctx)
		if err != nil {
			slog.Warn("Relay server failed to accept", logging.Error(err))
			continue
		}
		go rs.handleConn(ctx, conn)
	}
}

func (rs *RelayServer) handleConn(ctx context.Context, conn quic.Connection) {
	stream, err := conn.AcceptStream(ctx)
	if err != nil {
		rs.logger.Warn("Relay stream error", logging.Error(err))
		return
	}

	// Initial handshake: receive signed join a packet
	buf := make([]byte, 4096)
	n, err := stream.Read(buf)
	if err != nil {
		rs.logger.Warn("initial read error", logging.Error(err))
		return
	}
	data, ok := verify(buf[:n])
	if !ok {
		rs.logger.Warn("signature failed from client", slog.String("remoteAddr", conn.RemoteAddr().String()))
		return
	}

	rs.logger.Debug("First join packet", "data", data, "datastr", string(data))

	var pkt RelayPacket
	if err := json.Unmarshal(data, &pkt); err != nil {
		rs.logger.Warn("first packet unmarshal error", logging.Error(err))
		return
	}
	if pkt.Type != "join" {
		rs.logger.Warn("invalid join packet", logging.Error(err))
		return
	}

	rs.joinRoom(pkt.RoomID, pkt.FromID, stream)

	go rs.relayLoop(pkt.FromID, stream)
}

func (rs *RelayServer) relayLoop(peerID string, stream quic.Stream) {
	buf := make([]byte, 4096)

	for {
		n, err := stream.Read(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			rs.logger.Warn("stream error when reading", logging.Error(err), logging.PeerID(peerID))
			break
		}

		data, ok := verify(buf[:n])
		if !ok {
			rs.logger.Warn("signature check failed when reading", logging.PeerID(peerID))
			continue
		}

		rs.logger.Debug("Following relay packets", "data", data, "datastr", string(data))

		var pkt RelayPacket
		if err := json.Unmarshal(data, &pkt); err != nil {
			rs.logger.Warn("relay packet unmarshal error", logging.Error(err), logging.PeerID(peerID))
			continue
		}

		switch pkt.Type {
		// case "join":
		// 	rs.joinRoom(pkt.RoomID, pkt.FromID, stream)

		case "data", "udp", "tcp":
			rs.sendTo(pkt.RoomID, pkt.ToID, pkt)

		case "broadcast":
			rs.broadcastFrom(pkt.RoomID, pkt.FromID, pkt)

		case "leave":
			rs.leaveRoom(pkt.FromID, pkt.RoomID)
		}
	}

	rs.disconnected(peerID)
}

func (rs *RelayServer) joinRoom(roomID, peerID string, stream quic.Stream) {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	room, ok := rs.rooms[roomID]
	if !ok {
		room = &Room{ID: roomID, Peers: make(map[string]*PeerConn)}
		rs.rooms[roomID] = room
		rs.logger.Info("new room created", logging.RoomID(roomID), logging.PeerID(peerID))
	}

	role := "guest"
	if len(room.Peers) == 0 {
		role = "host"
		rs.logger.Info("user will become a host", logging.RoomID(roomID), logging.PeerID(peerID))
	}

	room.Peers[peerID] = &PeerConn{
		ID:     peerID,
		RoomID: roomID,
		Role:   role,
		Stream: stream,
	}
	rs.peerToRooms[peerID] = roomID
	rs.logger.Info("joined room", logging.RoomID(roomID), logging.PeerID(peerID))

	// Notify about the new dynamic joiner
	for _, peer := range room.Peers {
		if peer.ID == peerID {
			continue
		}

		pkt := RelayPacket{
			Type:    "join",
			RoomID:  roomID,
			FromID:  "system",
			ToID:    peer.ID,
			Payload: nil,
		}
		rs.sendSigned(peer.Stream, pkt)
	}
}

func (rs *RelayServer) leaveRoom(peerID, roomID string) {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	room, ok := rs.rooms[roomID]
	if !ok {
		rs.logger.Warn("peer leaving the room which does not exist", logging.RoomID(roomID), logging.PeerID(peerID))
		return
	}

	leaver, wasHost := room.Peers[peerID], false
	if leaver != nil && leaver.Role == "host" {
		wasHost = true
	}
	defer leaver.Stream.Close()

	delete(room.Peers, peerID)
	delete(rs.peerToRooms, peerID)
	rs.logger.Info("peer left room", logging.RoomID(roomID), logging.PeerID(peerID))

	if len(room.Peers) == 0 {
		delete(rs.rooms, roomID)
		rs.logger.Info("room deleted (empty)", logging.RoomID(roomID))
		return
	}

	if wasHost {
		// Elect the new host
		newHost := rs.electNewHost(room)
		if newHost == nil {
			panic("should not happen")
		}

		// Migrate host
		newHost.Role = "host"
		rs.logger.Info("peer promoted to host", slog.String("newHostID", newHost.ID), logging.RoomID(roomID))

		// Notify about the new host
		for _, peer := range room.Peers {
			pkt := RelayPacket{
				Type:    "migrate",
				RoomID:  roomID,
				FromID:  "system",
				ToID:    peer.ID,
				Payload: []byte(newHost.ID),
			}
			rs.sendSigned(peer.Stream, pkt)
		}
	}
}

func (rs *RelayServer) electNewHost(room *Room) *PeerConn {
	// TODO: Find the oldest by connection time
	for _, peer := range room.Peers {
		return peer
	}
	return nil
}

func (rs *RelayServer) disconnected(peerID string) {
	rs.logger.Info("disconnected from relay", logging.PeerID(peerID))

	rs.mu.Lock()
	roomID, ok := rs.peerToRooms[peerID]
	if !ok {
		rs.mu.Unlock()
		return
	}
	rs.mu.Unlock()

	rs.leaveRoom(peerID, roomID)
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

func (rs *RelayServer) sendSigned(stream quic.Stream, pkt RelayPacket) {
	data, _ := json.Marshal(pkt)
	packet := sign(data)
	stream.Write(packet)
}
