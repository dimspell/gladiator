package redirect

import (
	"context"
	"testing"
)

// NewTestManager builds a HostManager wired with the InMemoryProxyFactory so
// that StartHost/StartGuest do not bind real sockets. Intended for tests only.
func NewTestManager(opts ...func(*HostManager)) *HostManager {
	base := []func(*HostManager){
		WithProxyFactory(&InMemoryProxyFactory{}),
		WithDisabledLogger(),
	}
	return NewManager(append(base, opts...)...)
}

// TestHostManager_StartHost_NoRealBind verifies that StartHost with the
// InMemoryProxyFactory succeeds without binding any OS socket (no loopback IP
// required). This is the regression guard for the macOS "bind: can't assign
// requested address" failure.
func TestHostManager_StartHost_NoRealBind(t *testing.T) {
	hm := NewTestManager()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	host, err := hm.StartHost(ctx, "peer1", "127.0.0.2", 6114, 6113,
		func([]byte) error { return nil },
		func([]byte) error { return nil },
		nil,
	)
	if err != nil {
		t.Fatalf("StartHost should not bind a real socket: %v", err)
	}
	if host == nil {
		t.Fatal("expected a non-nil host")
	}

	// The proxy must be the in-memory stub, not a real listener.
	if _, ok := host.ProxyTCP.(*inMemoryRedirect); !ok {
		t.Fatalf("expected inMemoryRedirect proxy, got %T", host.ProxyTCP)
	}

	hm.StopHost(host)
}
