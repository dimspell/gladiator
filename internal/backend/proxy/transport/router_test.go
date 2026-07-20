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

// TestPacketRouter_DynamicJoin_NonHost_NoGuest proves the "StartGuest only when
// host" rule end-to-end: when this peer is NOT the current host, a dynamic join
// must not create a guest dialer (no fake host is registered). This defends
// against the §8.1 pain point 5 noise/IP-conflict class of bug.
func TestPacketRouter_DynamicJoin_NonHost_NoGuest(t *testing.T) {
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

	// Act as a NON-host: selfID != currentHostID.
	pr.mu.Lock()
	pr.selfID = "100"
	pr.currentHostID = "999"
	pr.mu.Unlock()

	pr.DynamicJoin(context.Background(), "test-room", "200")

	if _, ok := pr.manager.GetPeerHost("200"); ok {
		t.Fatal("non-host peer must not create a guest dialer for a dynamic join")
	}
}

// TestPacketRouter_StartHostPing_SendsPings proves the host ping goroutine
// sends KindPing packets when the router is the room host.
func TestPacketRouter_StartHostPing_SendsPings(t *testing.T) {
	mt := newMockTransport()
	pr := &PacketRouter{
		logger:           slog.Default(),
		selfID:           "100",
		transport:        mt,
		hostPingInterval: 10 * time.Millisecond,
	}

	pr.StartHostPing()
	defer pr.Reset()

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		mt.mu.Lock()
		found := false
		for _, pkt := range mt.sent {
			if pkt.Kind == KindPing {
				found = true
			}
		}
		mt.mu.Unlock()
		if found {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("expected at least one ping packet within 1s (interval=10ms)")
}

func TestPacketRouter_StartHostPing_StopsOnReset(t *testing.T) {
	mt := newMockTransport()
	pr := &PacketRouter{
		logger:           slog.Default(),
		selfID:           "100",
		transport:        mt,
		hostPingInterval: 10 * time.Millisecond,
	}

	pr.StartHostPing()

	// Wait for a few pings to be sent
	time.Sleep(50 * time.Millisecond)

	pr.Reset()

	// Give time for a stale goroutine to fire (it shouldn't)
	time.Sleep(100 * time.Millisecond)

	mt.mu.Lock()
	// Count pings sent after the reset (we can't easily know when the last pre-reset
	// ping fired, but we can check that the total is reasonable and not runaway)
	sentAfter := 0
	for _, pkt := range mt.sent {
		if pkt.Kind == KindPing {
			sentAfter++
		}
	}
	mt.mu.Unlock()

	if sentAfter > 5 {
		t.Fatalf("expected no more than 5 pings (pre-reset), got %d (goroutine may have kept running)", sentAfter)
	}
}

// TestPacketRouter_Connect_LoopSurvivesCallerCtxCancel proves the receive loop
// is owned by the router and keeps running after the caller's context is
// cancelled (e.g. an HTTP request scope), and that Reset cancels it promptly.
func TestPacketRouter_Connect_LoopSurvivesCallerCtxCancel(t *testing.T) {
	callerCtx, callerCancel := context.WithCancel(context.Background())

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

	pr.DynamicJoin(context.Background(), "test-room", "200")
	if _, ok := pr.manager.GetPeerHost("200"); !ok {
		t.Fatal("peer 200 not registered after dynamicJoin")
	}

	// Connect with the caller's context, then cancel it. The loop must keep
	// running because the router derives its own loopCtx from Background.
	if err := pr.Connect(callerCtx, "test-room"); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	callerCancel()

	// Give the loop a moment; it should still be alive despite callerCtx cancel.
	time.Sleep(20 * time.Millisecond)

	// Push a TCP packet into the mock transport; the loop must still dispatch it.
	pr.transport.(*mockTransport).recvCh <- TransportPacket{
		FromID: "200",
		RoomID: "test-room",
		Kind:   KindTCP,
		Data:   []byte("survived"),
	}

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if string(cap.Bytes()) == "survived" {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if got := string(cap.Bytes()); got != "survived" {
		t.Fatalf("loop died after caller ctx cancel; got %q", got)
	}

	// Reset must cancel the loop promptly (not block waiting on transport close).
	done := make(chan struct{})
	go func() {
		pr.Reset()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Reset did not exit loop promptly after caller ctx cancel")
	}
}
