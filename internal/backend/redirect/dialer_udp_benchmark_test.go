package redirect

import (
	"context"
	"log/slog"
	"net"
	"testing"
	"time"
)

// fastFakeUDPConn simulates a UDP connection with minimal overhead.
type fastFakeUDPConn struct {
	WriteCount int
	ReadBuf    []byte
}

func (f *fastFakeUDPConn) ReadFromUDP(b []byte) (int, *net.UDPAddr, error) {
	copy(b, f.ReadBuf)
	return len(f.ReadBuf), &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 9999}, nil
}

func (f *fastFakeUDPConn) Write(b []byte) (int, error) {
	f.WriteCount++
	return len(b), nil
}

func (f *fastFakeUDPConn) Close() error                      { return nil }
func (f *fastFakeUDPConn) SetReadDeadline(t time.Time) error { return nil }
func (f *fastFakeUDPConn) LocalAddr() net.Addr               { return &net.UDPAddr{} }
func (f *fastFakeUDPConn) RemoteAddr() net.Addr              { return &net.UDPAddr{} }

// Benchmark writing messages to the UDP connection.
func BenchmarkDialerUDP_Write(b *testing.B) {
	conn := &fastFakeUDPConn{}
	d := &DialerUDP{
		conn:   conn,
		logger: slog.New(slog.NewTextHandler(nil, &slog.HandlerOptions{Level: slog.LevelError})),
	}

	msg := []byte("benchmark-payload")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := d.Write(msg); err != nil {
			b.Fatalf("Write failed: %v", err)
		}
	}
}

// Benchmark reading packets and calling the onReceive handler.
func BenchmarkDialerUDP_Run(b *testing.B) {
	conn := &fastFakeUDPConn{
		ReadBuf: []byte("benchmark-read-payload"),
	}
	d := &DialerUDP{
		conn:   conn,
		logger: slog.New(slog.NewTextHandler(nil, &slog.HandlerOptions{Level: slog.LevelError})),
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	count := 0
	go func() {
		_ = d.Run(ctx, func(p []byte) error {
			count++
			if count >= b.N {
				cancel()
			}
			return nil
		})
	}()

	// Wait for benchmark to complete
	for ctx.Err() == nil {
		time.Sleep(time.Microsecond)
	}
}
