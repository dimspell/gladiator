package redirect

import (
	"context"
	"sync"
	"time"
)

// InMemoryProxyFactory is a test-only ProxyFactory that returns in-memory
// Redirect stubs instead of real OS sockets. This lets proxy/redirect tests run
// without binding loopback addresses (e.g. 127.0.0.2) that are unavailable on
// some hosts (notably macOS without loopback aliases).
type InMemoryProxyFactory struct{}

func (f *InMemoryProxyFactory) NewDialTCP(ip, port string, onReceive ReceiveFunc) (Redirect, error) {
	return newInMemoryRedirect(onReceive), nil
}

func (f *InMemoryProxyFactory) NewDialUDP(ip, port string, onReceive ReceiveFunc) (Redirect, error) {
	return newInMemoryRedirect(onReceive), nil
}

func (f *InMemoryProxyFactory) NewListenerTCP(ip, port string, onReceive ReceiveFunc) (Redirect, error) {
	return newInMemoryRedirect(onReceive), nil
}

func (f *InMemoryProxyFactory) NewListenerUDP(ip, port string, onReceive ReceiveFunc) (Redirect, error) {
	return newInMemoryRedirect(onReceive), nil
}

// inMemoryRedirect is an in-memory Redirect that never touches the network.
// It is exported only via the InMemoryProxyFactory.
type inMemoryRedirect struct {
	mu        sync.Mutex
	closed    bool
	onReceive ReceiveFunc
	buf       []byte
}

func newInMemoryRedirect(onReceive ReceiveFunc) *inMemoryRedirect {
	return &inMemoryRedirect{onReceive: onReceive}
}

// Write records the payload and forwards it to the receive callback if set.
func (m *inMemoryRedirect) Write(p []byte) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return 0, ErrClosed
	}
	m.buf = append(m.buf, p...)
	if m.onReceive != nil {
		_ = m.onReceive(p)
	}
	return len(p), nil
}

// Close marks the redirect as closed.
func (m *inMemoryRedirect) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

// Run blocks until the context is cancelled, mimicking a running proxy.
func (m *inMemoryRedirect) Run(ctx context.Context) error {
	<-ctx.Done()
	return ctx.Err()
}

// Alive reports whether the redirect is still open.
func (m *inMemoryRedirect) Alive(now time.Time, timeout time.Duration) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return !m.closed
}
