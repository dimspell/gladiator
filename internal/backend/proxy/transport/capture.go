package transport

import (
	"bytes"
	"context"
	"sync"
	"time"

	"github.com/dimspell/gladiator/internal/backend/redirect"
)

// CaptureRedirect is a redirect.Redirect that records everything written to it.
// Run blocks forever so the FakeHost stays alive (the cleanup goroutine waits
// on g.Wait() which waits on Run). Close is a no-op because StopAll cleans up
// the PeerHosts/Hosts maps directly and the blocked goroutines exit when the
// test process finishes.
//
// It is exported (and lives in a non-test file) so it can be shared as a test
// fixture by both this package's tests and the relay integration tests.
type CaptureRedirect struct {
	mu     sync.Mutex
	buf    bytes.Buffer
	closed bool
}

func (c *CaptureRedirect) Run(ctx context.Context) error { select {} }
func (c *CaptureRedirect) Alive(time.Time, time.Duration) bool              { return true }
func (c *CaptureRedirect) Write(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.buf.Write(p)
}
func (c *CaptureRedirect) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
	return nil
}
func (c *CaptureRedirect) Bytes() []byte {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]byte(nil), c.buf.Bytes()...)
}

// CaptureFactory returns the same CaptureRedirect for every proxy so tests can
// observe what PacketRouter writes to a peer's ProxyTCP/ProxyUDP.
type CaptureFactory struct {
	Shared *CaptureRedirect
}

func (f *CaptureFactory) NewDialTCP(ip, port string, onReceive redirect.ReceiveFunc) (redirect.Redirect, error) {
	return f.Shared, nil
}
func (f *CaptureFactory) NewDialUDP(ip, port string, onReceive redirect.ReceiveFunc) (redirect.Redirect, error) {
	return f.Shared, nil
}
func (f *CaptureFactory) NewListenerTCP(ip, port string, onReceive redirect.ReceiveFunc) (redirect.Redirect, error) {
	return f.Shared, nil
}
func (f *CaptureFactory) NewListenerUDP(ip, port string, onReceive redirect.ReceiveFunc) (redirect.Redirect, error) {
	return f.Shared, nil
}
