package relay

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/dimspell/gladiator/internal/backend/proxy/transport"
)

// InMemoryHub routes TransportPackets between InMemoryTransport instances that
// share a room, without any network. It is the test double that lets the full
// relay dispatch logic run deterministically in-process (see integration_test.go).
type InMemoryHub struct {
	mu         sync.Mutex
	byRoomPeer map[string]map[string]*InMemoryTransport
}

func NewInMemoryHub() *InMemoryHub {
	return &InMemoryHub{byRoomPeer: make(map[string]map[string]*InMemoryTransport)}
}

func (h *InMemoryHub) register(t *InMemoryTransport) {
	h.mu.Lock()
	defer h.mu.Unlock()
	room, ok := h.byRoomPeer[t.roomID]
	if !ok {
		room = make(map[string]*InMemoryTransport)
		h.byRoomPeer[t.roomID] = room
	}
	room[t.selfID] = t
}

func (h *InMemoryHub) unregister(t *InMemoryTransport) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if room, ok := h.byRoomPeer[t.roomID]; ok {
		delete(room, t.selfID)
		if len(room) == 0 {
			delete(h.byRoomPeer, t.roomID)
		}
	}
}

func (h *InMemoryHub) deliver(pkt transport.TransportPacket) error {
	h.mu.Lock()
	t, ok := h.byRoomPeer[pkt.RoomID][pkt.ToID]
	h.mu.Unlock()
	if !ok {
		return fmt.Errorf("in-memory transport: no peer %q in room %q", pkt.ToID, pkt.RoomID)
	}
	// Buffered, non-blocking delivery: the target's receiveLoop drains recvCh.
	t.recvCh <- pkt
	return nil
}

// InMemoryTransport is a PeerTransport backed by an InMemoryHub. Send delivers
// synchronously to the target peer's receive channel.
type InMemoryTransport struct {
	hub    *InMemoryHub
	selfID string
	roomID string

	mu     sync.Mutex
	recvCh chan transport.TransportPacket
	done   chan struct{}
	once   sync.Once
}

func NewInMemoryTransport(hub *InMemoryHub, selfID string) *InMemoryTransport {
	return &InMemoryTransport{
		hub:    hub,
		selfID: selfID,
		recvCh: make(chan transport.TransportPacket, 64),
		done:   make(chan struct{}),
	}
}

func (t *InMemoryTransport) Join(ctx context.Context, roomID string) error {
	t.mu.Lock()
	t.recvCh = make(chan transport.TransportPacket, 64)
	t.done = make(chan struct{})
	t.roomID = roomID
	t.mu.Unlock()
	t.hub.register(t)
	return nil
}

func (t *InMemoryTransport) Send(ctx context.Context, pkt transport.TransportPacket) error {
	return t.hub.deliver(pkt)
}

func (t *InMemoryTransport) Recv(ctx context.Context) (transport.TransportPacket, error) {
	t.mu.Lock()
	done := t.done
	recvCh := t.recvCh
	t.mu.Unlock()
	select {
	case <-ctx.Done():
		return transport.TransportPacket{}, ctx.Err()
	case <-done:
		return transport.TransportPacket{}, io.EOF
	case pkt, ok := <-recvCh:
		if !ok {
			return transport.TransportPacket{}, io.EOF
		}
		return pkt, nil
	}
}

func (t *InMemoryTransport) Leave(ctx context.Context) error {
	t.hub.unregister(t)
	return nil
}

func (t *InMemoryTransport) Close() error {
	t.once.Do(func() {
		t.mu.Lock()
		close(t.done)
		t.mu.Unlock()
		t.hub.unregister(t)
	})
	return nil
}
