package transport

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/dimspell/gladiator/internal/backend/redirect"
)

// mockTransport is an in-test PeerTransport that buffers sent packets and lets
// the test push packets into the receive channel.
type mockTransport struct {
	mu       sync.Mutex
	recvCh   chan TransportPacket
	closed   bool
	joinRoom string
	joinErr  error
	sendErr  error
	sent     []TransportPacket
}

func newMockTransport() *mockTransport {
	return &mockTransport{recvCh: make(chan TransportPacket, 16)}
}

func (m *mockTransport) Join(ctx context.Context, roomID string) error {
	m.mu.Lock()
	m.joinRoom = roomID
	m.mu.Unlock()
	return m.joinErr
}

func (m *mockTransport) Send(ctx context.Context, pkt TransportPacket) error {
	m.mu.Lock()
	m.sent = append(m.sent, pkt)
	m.mu.Unlock()
	return m.sendErr
}

func (m *mockTransport) Recv(ctx context.Context) (TransportPacket, error) {
	select {
	case <-ctx.Done():
		return TransportPacket{}, ctx.Err()
	case pkt, ok := <-m.recvCh:
		if !ok {
			return TransportPacket{}, io.EOF
		}
		return pkt, nil
	}
}

func (m *mockTransport) Leave(ctx context.Context) error { return nil }

func (m *mockTransport) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return nil
	}
	m.closed = true
	close(m.recvCh)
	return nil
}

func TestPacketRouter_ReceiveLoop_DispatchesTCP(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cap := &CaptureRedirect{}
	factory := &CaptureFactory{Shared: cap}

	pr := &PacketRouter{
		logger: slog.Default(),
		roomID: "test-room",
		manager: redirect.NewManager(
			redirect.WithProxyFactory(factory),
			redirect.WithDisabledLogger(),
		),
		transport: newMockTransport(),
	}

	// Act as the host so dynamicJoin provisions a guest host for peer 200.
	pr.mu.Lock()
	pr.selfID = "100"
	pr.currentHostID = "100"
	pr.mu.Unlock()

	pr.DynamicJoin(ctx, "test-room", "200")

	host, ok := pr.manager.GetPeerHost("200")
	if !ok || host.ProxyTCP == nil {
		t.Fatalf("peer 200 not registered after dynamicJoin (ok=%v, proxyTCP=%v)", ok, host)
	}

	// Drive the dispatch path synchronously (receiveLoop just calls this).
	pr.onTransportPacket(TransportPacket{
		FromID: "200",
		RoomID: "test-room",
		Kind:   KindTCP,
		Data:   []byte("hello"),
	})

	if got := string(cap.Bytes()); got != "hello" {
		t.Fatalf("TCP payload not delivered to peer, got %q", got)
	}
}

func TestPacketRouter_ReceiveLoop_ExitsOnContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	pr := &PacketRouter{
		logger:    slog.Default(),
		roomID:    "test-room",
		transport: newMockTransport(),
	}

	pr.wg.Add(1)
	done := make(chan struct{})
	go func() {
		pr.receiveLoop(ctx)
		close(done)
	}()

	// Let the receiveLoop settle into the blocking Recv
	time.Sleep(10 * time.Millisecond)

	// Cancel the context while Recv is blocking
	cancel()

	select {
	case <-done:
		// receiveLoop exited due to context cancel
	case <-time.After(time.Second):
		t.Fatal("receiveLoop did not exit within 1s after context cancel")
	}
}

func TestPacketRouter_SendPacket_DataRace(t *testing.T) {
	mt := newMockTransport()
	pr := &PacketRouter{
		logger:    slog.Default(),
		selfID:    "test-self",
		transport: mt,
	}

	var wg sync.WaitGroup
	wg.Add(3)

	// Concurrent sendPacket from FakeHost-like goroutine
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			_ = pr.SendPacket(RelayPacket{Type: "tcp", RoomID: "room"})
		}
	}()

	// Concurrent selfID writes (simulates relay.go CreateRoom/GetGame)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			pr.mu.Lock()
			pr.selfID = fmt.Sprintf("id-%d", i)
			pr.mu.Unlock()
		}
	}()

	// Concurrent disconnect/reset (disconnect handles its own locking)
	go func() {
		defer wg.Done()
		for i := 0; i < 20; i++ {
			time.Sleep(time.Microsecond)
			pr.Disconnect()
		}
	}()

	wg.Wait()

	mt.mu.Lock()
	sent := len(mt.sent)
	mt.mu.Unlock()
	if sent != 100 {
		t.Errorf("expected 100 sent packets, got %d", sent)
	}
}
