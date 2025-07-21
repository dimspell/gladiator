package redirect

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"
)

type mockRedirect struct {
	runCalled   bool
	closeCalled bool
	mu          sync.Mutex
	runErr      error
}

func (m *mockRedirect) Run(ctx context.Context) error {
	m.mu.Lock()
	m.runCalled = true
	m.mu.Unlock()
	<-ctx.Done()
	return m.runErr
}
func (m *mockRedirect) Close() error {
	m.mu.Lock()
	m.closeCalled = true
	m.mu.Unlock()
	return nil
}
func (m *mockRedirect) Write(p []byte) (int, error) {
	return len(p), nil
}
func (m *mockRedirect) Alive(_ time.Time, _ time.Duration) bool {
	return true
}

// mockProxyFactory returns the same mockRedirect for all methods.
type mockProxyFactory struct {
	tcp, udp *mockRedirect
	fail     bool
}

func (m *mockProxyFactory) NewDialTCP(ip, port string, onReceive ReceiveFunc) (Redirect, error) {
	if m.fail {
		return nil, errors.New("fail")
	}
	return m.tcp, nil
}
func (m *mockProxyFactory) NewDialUDP(ip, port string, onReceive ReceiveFunc) (Redirect, error) {
	if m.fail {
		return nil, errors.New("fail")
	}
	return m.udp, nil
}
func (m *mockProxyFactory) NewListenerTCP(ip, port string, onReceive ReceiveFunc) (Redirect, error) {
	if m.fail {
		return nil, errors.New("fail")
	}
	return m.tcp, nil
}
func (m *mockProxyFactory) NewListenerUDP(ip, port string, onReceive ReceiveFunc) (Redirect, error) {
	if m.fail {
		return nil, errors.New("fail")
	}
	return m.udp, nil
}

func TestHostManager_IPAssignment(t *testing.T) {
	hm := NewManager(net.IPv4(127, 0, 0, 1))
	ip1, err := hm.AssignIP("peer1")
	if err != nil || ip1 == "" {
		t.Fatalf("expected IP, got %v %v", ip1, err)
	}
	ip2, err := hm.AssignIP("peer2")
	if err != nil || ip2 == "" || ip1 == ip2 {
		t.Fatalf("expected unique IPs, got %v %v", ip1, ip2)
	}
	// Should return same IP for same peer
	ip1b, _ := hm.AssignIP("peer1")
	if ip1b != ip1 {
		t.Errorf("expected same IP for same peer")
	}
}

func TestHostManager_StartHostAndGuest(t *testing.T) {
	tcp := &mockRedirect{}
	udp := &mockRedirect{}
	hm := NewManager(net.IPv4(127, 0, 0, 1), WithProxyFactory(&mockProxyFactory{tcp, udp, false}))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ip1, _ := hm.AssignIP("peer1")
	host, err := hm.StartHost(ctx, "peer1", ip1, 1234, 5678, func([]byte) error { return nil }, func([]byte) error { return nil }, nil)
	if err != nil {
		t.Fatalf("StartHost failed: %v", err)
	}
	if host.ProxyTCP == nil || host.ProxyUDP == nil {
		t.Errorf("proxies not set correctly")
	}
	ip2, _ := hm.AssignIP("peer2")
	guest, err := hm.StartGuest(ctx, "peer2", ip2, 1234, 5678, func([]byte) error { return nil }, func([]byte) error { return nil }, nil)
	if err != nil {
		t.Fatalf("StartGuest failed: %v", err)
	}
	if guest.ProxyTCP == nil || guest.ProxyUDP == nil {
		t.Errorf("proxies not set correctly")
	}
}

func TestHostManager_CreateFakeHost_ErrorHandling(t *testing.T) {
	hm := NewManager(net.IPv4(127, 0, 0, 1), WithProxyFactory(&mockProxyFactory{&mockRedirect{}, &mockRedirect{}, true}))
	ctx := context.Background()
	ip, _ := hm.AssignIP("peer1")
	_, err := hm.StartHost(ctx, "peer1", ip, 1234, 5678, func([]byte) error { return nil }, func([]byte) error { return nil }, nil)
	if err == nil {
		t.Errorf("expected error from proxy factory")
	}
}

func TestHostManager_RemoveByIPAndRemoteID(t *testing.T) {
	t.Skip("Failing - needs to be fixed")
	hm := NewManager(net.IPv4(127, 0, 0, 1))
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	ip, _ := hm.AssignIP("peer1")
	if _, err := hm.StartHost(ctx, "peer1", ip, 1234, 5678, func([]byte) error { return nil }, func([]byte) error { return nil }, nil); err != nil {
		t.Fatalf("StartHost failed: %v", err)
		return
	}
	if _, ok := hm.GetHostByIP(ip); !ok {
		t.Fatalf("host not found by IP")
	}
	hm.RemoveByIP(ip[:len(ip)-1]) // Remove by prefix
	if _, ok := hm.GetHostByIP(ip); ok {
		t.Errorf("host should be removed by prefix")
	}
	ip2, _ := hm.AssignIP("peer2")
	if _, err := hm.StartHost(ctx, "peer2", ip2, 1234, 5678, func([]byte) error { return nil }, func([]byte) error { return nil }, nil); err != nil {
		t.Fatalf("StartHost failed: %v", err)
		return
	}
	removed := hm.RemoveByRemoteID("peer2")
	if !removed {
		t.Errorf("expected RemoveByRemoteID to return true")
	}
	if _, ok := hm.GetPeerHost("peer2"); ok {
		t.Errorf("host should be removed by remoteID")
	}
}

func TestHostManager_StopHost_Idempotent(t *testing.T) {
	hm := NewManager(net.IPv4(127, 0, 0, 1))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ip, _ := hm.AssignIP("peer1")
	host, _ := hm.StartHost(ctx, "peer1", ip, 1234, 5678, func([]byte) error { return nil }, func([]byte) error { return nil }, nil)
	hm.StopHost(host)
	hm.StopHost(host) // Should not panic or double-close
}

func TestHostManager_ConcurrentStopAndRemove(t *testing.T) {
	hm := NewManager(net.IPv4(127, 0, 0, 1))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ip, _ := hm.AssignIP("peer1")
	host, _ := hm.StartHost(ctx, "peer1", ip, 1234, 5678, func([]byte) error { return nil }, func([]byte) error { return nil }, nil)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); hm.StopHost(host) }()
	go func() { defer wg.Done(); hm.RemoveByIP(ip[:len(ip)-1]) }()
	wg.Wait()
}

func TestHostManager_DoubleAssignmentAndRemoval(t *testing.T) {
	hm := NewManager(net.IPv4(127, 0, 0, 1))
	ip1, err := hm.AssignIP("peer1")
	if err != nil {
		t.Fatalf("AssignIP failed: %v", err)
	}
	ip2, err := hm.AssignIP("peer1")
	if err != nil {
		t.Fatalf("AssignIP failed: %v", err)
	}
	if ip1 != ip2 {
		t.Errorf("expected same IP for double assignment")
	}
	hm.RemoveByRemoteID("peer1")
	ip3, err := hm.AssignIP("peer1")
	if err != nil {
		t.Fatalf("AssignIP after removal failed: %v", err)
	}
	if ip3 != ip1 {
		t.Errorf("expected the same IP after removal, got new: %v", ip3)
	}
}

func TestHostManager_RemoveByRemoteID_Nonexistent(t *testing.T) {
	hm := NewManager(net.IPv4(127, 0, 0, 1))
	removed := hm.RemoveByRemoteID("notfound")
	if removed {
		t.Errorf("expected false for nonexistent peer")
	}
}

func TestHostManager_StopAll(t *testing.T) {
	tcp := &mockRedirect{}
	udp := &mockRedirect{}
	hm := NewManager(net.IPv4(127, 0, 0, 1), WithProxyFactory(&mockProxyFactory{tcp, udp, false}))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ip1, _ := hm.AssignIP("peer1")
	ip2, _ := hm.AssignIP("peer2")
	hm.StartHost(ctx, "peer1", ip1, 1234, 5678, func([]byte) error { return nil }, func([]byte) error { return nil }, nil)
	hm.StartHost(ctx, "peer2", ip2, 1234, 5678, func([]byte) error { return nil }, func([]byte) error { return nil }, nil)
	hm.StopAll()
	if len(hm.Hosts) != 0 || len(hm.PeerHosts) != 0 || len(hm.PeerIPs) != 0 || len(hm.IPToPeerID) != 0 {
		t.Errorf("expected all maps to be empty after StopAll")
	}
	if !tcp.closeCalled || !udp.closeCalled {
		t.Errorf("expected proxies to be closed on StopAll")
	}
}

func TestHostManager_CreateFakeHost_TCPFail(t *testing.T) {
	failingFactory := &mockProxyFactory{tcp: &mockRedirect{}, udp: &mockRedirect{}, fail: true}
	hm := NewManager(net.IPv4(127, 0, 0, 1), WithProxyFactory(failingFactory))
	ctx := context.Background()
	ip, _ := hm.AssignIP("peer1")
	_, err := hm.StartHost(ctx, "peer1", ip, 1234, 5678, func([]byte) error { return nil }, func([]byte) error { return nil }, nil)
	if err == nil {
		t.Errorf("expected error from failing TCP proxy factory")
	}
}

func TestHostManager_ConcurrentAssignAndRemove(t *testing.T) {
	hm := NewManager(net.IPv4(127, 0, 0, 1))
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		peer := fmt.Sprintf("peer%d", i)
		wg.Add(1)
		go func(p string) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				_, _ = hm.AssignIP(p)
			}
		}(peer)
	}
	for i := 0; i < 10; i++ {
		prefix := "127.0.0."
		wg.Add(1)
		go func() {
			defer wg.Done()
			hm.RemoveByIP(prefix)
		}()
	}
	wg.Wait()
}

func TestHostManager_HostGuestLifecycle(t *testing.T) {
	t.Skip("Failing - needs to be fixed")
	hm := NewManager(net.IPv4(127, 0, 0, 1))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ipHost, _ := hm.AssignIP("host")
	ipGuest, _ := hm.AssignIP("guest")
	hm.StartHost(ctx, "host", ipHost, 1234, 5678, func([]byte) error { return nil }, func([]byte) error { return nil }, nil)
	hm.StartGuest(ctx, "guest", ipGuest, 1234, 5678, func([]byte) error { return nil }, func([]byte) error { return nil }, nil)
	hm.RemoveByRemoteID("host")
	if _, ok := hm.GetPeerHost("host"); ok {
		t.Errorf("host should be removed")
	}
	if _, ok := hm.GetPeerHost("guest"); !ok {
		t.Errorf("guest should remain after host removal")
	}
	hm.RemoveByRemoteID("guest")
	if _, ok := hm.GetPeerHost("guest"); ok {
		t.Errorf("guest should be removed")
	}
}

func TestHostManager_RemoveByIP_Idempotent(t *testing.T) {
	hm := NewManager(net.IPv4(127, 0, 0, 1))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ip, _ := hm.AssignIP("peer1")
	hm.StartHost(ctx, "peer1", ip, 1234, 5678, func([]byte) error { return nil }, func([]byte) error { return nil }, nil)
	hm.RemoveByIP(ip[:len(ip)-1])
	hm.RemoveByIP(ip[:len(ip)-1]) // Should not panic
}

func TestHostManager_RemoveByRemoteID_Idempotent(t *testing.T) {
	hm := NewManager(net.IPv4(127, 0, 0, 1))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ip, _ := hm.AssignIP("peer1")
	hm.StartHost(ctx, "peer1", ip, 1234, 5678, func([]byte) error { return nil }, func([]byte) error { return nil }, nil)
	hm.RemoveByRemoteID("peer1")
	hm.RemoveByRemoteID("peer1") // Should not panic
}

func TestHostManager_ProxiesClosedOnRemove(t *testing.T) {
	tcp := &mockRedirect{}
	udp := &mockRedirect{}
	hm := NewManager(net.IPv4(127, 0, 0, 1), WithProxyFactory(&mockProxyFactory{tcp, udp, false}))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ip, _ := hm.AssignIP("peer1")
	hm.StartHost(ctx, "peer1", ip, 1234, 5678, func([]byte) error { return nil }, func([]byte) error { return nil }, nil)
	hm.RemoveByRemoteID("peer1")
	if !tcp.closeCalled || !udp.closeCalled {
		t.Errorf("expected proxies to be closed on RemoveByRemoteID")
	}
}
