package relay

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net"
	"sync"

	"github.com/dimspell/gladiator/internal/backend/bsession"
	"github.com/dimspell/gladiator/internal/wire"
)

type PacketRouter struct {
	mu      sync.Mutex
	logger  *slog.Logger
	manager *HostManager

	session *bsession.Session

	isHost        bool
	selfID        string
	currentHostID string
}

func NewPacketRouter(selfID string, relayAddr *net.UDPAddr, relayConn *net.UDPConn, manager *HostManager) *PacketRouter {
	return &PacketRouter{
		manager: manager,
		// relayAddr:  relayAddr,
		// relayConn:  relayConn,
		selfID: selfID,
	}
}

func (r *PacketRouter) Handle(ctx context.Context, payload []byte) error {
	eventType := wire.ParseEventType(payload)

	// May the game data flow on the side channel (16 opened ports)

	switch eventType {
	case wire.LobbyUsers:
		return nil
	case wire.JoinLobby:
		return nil
	case wire.CreateRoom:
		return nil
	case wire.JoinRoom:
		return decodeAndHandle(ctx, payload, wire.JoinRoom, r.handleJoinRoom)
	case wire.LeaveRoom, wire.LeaveLobby:
		return decodeAndHandle(ctx, payload, wire.LeaveRoom, r.handleLeaveRoom)
	case wire.HostMigration:
		return decodeAndHandle(ctx, payload, wire.HostMigration, r.handleHostMigration)
	default:
		r.logger.Debug("unknown wire message", "type", eventType.String())
		return nil
	}
}

// Generic handler for simple event messages
func decodeAndHandle[T any](ctx context.Context, payload []byte, eventType wire.EventType, handler func(context.Context, T) error) error {
	_, msg, err := wire.DecodeTyped[T](payload)
	if err != nil {
		slog.Error(fmt.Sprintf("failed to decode payload for event: %s", eventType.String()), "error", err, "payload", string(payload))
		return err
	}
	return handler(ctx, msg.Content)
}

// handleJoinRoom handles the event when new dynamic joiner has arrived (player who connected mid-game)
func (r *PacketRouter) handleJoinRoom(ctx context.Context, player wire.Player) error {
	// create guest host

	return nil
}

func (r *PacketRouter) handleLeaveRoom(ctx context.Context, player wire.Player) error {
	// r.OnPeerLeave(player.ID())

	return nil
}

// LEAVE:PlayerC
func (r *PacketRouter) OnPeerLeave(peerID string) {
	if r.selfID == peerID {
		// r.Reset()
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.manager.RemoveByRemoteID(peerID)

	fakeIP, ok := r.manager.peerIPs[peerID]
	if !ok {
		log.Printf("Peer %s not found, nothing to remove", peerID)
		return
	}

	// Cleanup guest instance
	r.manager.RemoveByIP(fakeIP)
}

func (r *PacketRouter) handleHostMigration(ctx context.Context, player wire.Player) error {
	return nil
}
