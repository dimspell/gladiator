package redirect

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"testing"
	"time"
)

// ---- MOCK IMPLEMENTATIONS ----

// fakeUDPConn implements udp.UDPConn
type fakeUDPConn struct {
	ReadDeadline time.Time
	ReadData     [][]byte
	WriteData    [][]byte
	ReadIndex    int
	CloseCalled  bool
}

func (f *fakeUDPConn) ReadFromUDP(b []byte) (int, *net.UDPAddr, error) {
	if f.ReadIndex >= len(f.ReadData) {
		return 0, nil, errors.New("no more data")
	}
	copy(b, f.ReadData[f.ReadIndex])
	n := len(f.ReadData[f.ReadIndex])
	f.ReadIndex++
	return n, &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 9999}, nil
}

func (f *fakeUDPConn) Write(b []byte) (int, error) {
	data := make([]byte, len(b))
	copy(data, b)
	f.WriteData = append(f.WriteData, data)
	return len(b), nil
}

func (f *fakeUDPConn) Close() error {
	f.CloseCalled = true
	return nil
}

func (f *fakeUDPConn) SetReadDeadline(t time.Time) error {
	f.ReadDeadline = t
	return nil
}

func (f *fakeUDPConn) LocalAddr() net.Addr {
	return &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 1234}
}

func (f *fakeUDPConn) RemoteAddr() net.Addr {
	return &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 4321}
}

// ---- UNIT TESTS ----

func TestDialerUDP_Run(t *testing.T) {
	t.Run("Read and forward", func(t *testing.T) {
		// Arrange
		fakeConn := &fakeUDPConn{
			ReadData: [][]byte{
				[]byte("one"),
				[]byte("two"),
			},
		}
		d := &DialerUDP{
			conn:   fakeConn,
			logger: slog.Default(),
		}

		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		var received [][]byte
		errCh := make(chan error, 1)
		defer close(errCh)

		// Act
		go func() {
			err := d.Run(ctx, func(p []byte) error {
				data := make([]byte, len(p))
				copy(data, p)
				received = append(received, data)

				// Stop the loop after 2 messages
				if len(received) == 2 {
					cancel()
				}
				return nil
			})
			errCh <- err
		}()

		// Assert
		select {
		case err := <-errCh:
			if err != nil && !errors.Is(err, context.Canceled) {
				t.Fatalf("unexpected error: %v", err)
			}
		case <-time.After(1 * time.Second):
			t.Fatal("test timeout: Run did not exit")
		}

		if len(received) != 2 || string(received[0]) != "one" || string(received[1]) != "two" {
			t.Fatalf("unexpected received data: %v", received)
		}
	})

	t.Run("Context cancelled", func(t *testing.T) {
		// Arrange
		fakeConn := &fakeUDPConn{
			ReadData: [][]byte{}, // no data - it will block
		}
		d := &DialerUDP{
			conn:   fakeConn,
			logger: slog.Default(),
		}

		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		// Act
		err := d.Run(ctx, func(p []byte) error {
			return nil
		})

		// Assert
		if err == nil || err.Error() == "" {
			t.Fatalf("expected context canceled error, got: %v", err)
		}
	})
}

func TestDialerUDP_Write(t *testing.T) {
	// Arrange
	fakeConn := &fakeUDPConn{}
	d := &DialerUDP{
		conn:   fakeConn,
		logger: slog.Default(),
	}

	// Act
	msg := []byte("hello")
	n, err := d.Write(msg)

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if n != len(msg) {
		t.Fatalf("expected %d bytes written, got %d", len(msg), n)
	}
	if len(fakeConn.WriteData) != 1 || string(fakeConn.WriteData[0]) != "hello" {
		t.Fatalf("unexpected data written: %v", fakeConn.WriteData)
	}
}
