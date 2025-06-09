package redirect

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"sync"
	"testing"
	"time"
)

// ---- MOCK IMPLEMENTATIONS ----

type mockConn struct {
	readData    []byte
	writeBuffer bytes.Buffer
	readErr     error
	writeErr    error
	closed      bool
	setDeadline bool
	remoteAddr  net.Addr
}

func (m *mockConn) Read(b []byte) (int, error) {
	if m.readErr != nil {
		return 0, m.readErr
	}
	copy(b, m.readData)
	return len(m.readData), nil
}

func (m *mockConn) Write(b []byte) (int, error) {
	if m.writeErr != nil {
		return 0, m.writeErr
	}
	return m.writeBuffer.Write(b)
}

func (m *mockConn) Close() error {
	m.closed = true
	return nil
}

func (m *mockConn) SetReadDeadline(t time.Time) error {
	m.setDeadline = true
	return nil
}

func (m *mockConn) RemoteAddr() net.Addr {
	return m.remoteAddr
}

// mockListener implements net.Listener for testing Run
type mockListener struct {
	acceptConns chan net.Conn
	closeCalled bool
}

func (m *mockListener) Accept() (net.Conn, error) {
	conn, ok := <-m.acceptConns
	if !ok {
		return nil, io.EOF
	}
	return conn, nil
}

func (m *mockListener) Close() error {
	m.closeCalled = true
	close(m.acceptConns)
	return nil
}

func (m *mockListener) Addr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 9999}
}

// timeoutErr implements net.De
type timeoutErr struct{}

func (timeoutErr) Error() string   { return "timeout" }
func (timeoutErr) Timeout() bool   { return true }
func (timeoutErr) Temporary() bool { return true }

func (timeoutErr) Unwrap() error { return nil }

// ---- UNIT TESTS ----

func TestListenerTCP_Run(t *testing.T) {
	t.Run("Got EOF", func(t *testing.T) {
		// Arrange
		mock := &mockConn{
			readErr:    io.EOF,
			remoteAddr: &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 3333},
		}
		listener := &ListenerTCP{logger: slog.Default()}
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Act
		err := listener.handleConnection(ctx, mock, func(p []byte) error {
			t.Fatal("should not be called")
			return nil
		})

		// Assert
		if err == nil || !errors.Is(err, io.EOF) {
			t.Fatalf("expected error io.EOF, got: %v", err)
		}
		if !mock.closed {
			t.Error("expected connection to be closed")
		}
	})

	t.Run("Context cancelled on handle connections", func(t *testing.T) {
		// Arrange
		mock := &mockConn{
			readData:   []byte("test"),
			remoteAddr: &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 4444},
		}
		listener := &ListenerTCP{logger: slog.Default()}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		// Act
		err := listener.handleConnection(ctx, mock, func(p []byte) error {
			return nil
		})

		// Assert
		if !errors.Is(err, context.Canceled) {
			t.Errorf("expected context.Canceled, got: %v", err)
		}
	})

	t.Run("Context canceled on listener", func(t *testing.T) {
		mockLn := &mockListener{acceptConns: make(chan net.Conn)}

		listener := &ListenerTCP{
			listener: mockLn,
			logger:   slog.Default(),
		}

		ctx, cancel := context.WithCancel(context.Background())

		// cause Accept to return error
		go func() {
			time.Sleep(100 * time.Millisecond)
			cancel()
		}()

		err := listener.Run(ctx, func(p []byte) error {
			t.Fatal("onReceive should not be called")
			return nil
		})

		if !errors.Is(err, context.Canceled) {
			t.Errorf("expected context.Canceled, got: %v", err)
		}
	})

	t.Run("Continue on deadline exceeded", func(t *testing.T) {
		// Arrange
		mock := &mockConn{
			readErr:    timeoutErr{},
			readData:   []byte("ignored"), // won't be used
			remoteAddr: &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 6666},
		}
		listener := &ListenerTCP{logger: slog.Default()}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Act
		done := make(chan struct{})
		go func() {
			// only allow a short loop
			_ = listener.handleConnection(ctx, mock, func(p []byte) error {
				t.Fatal("should not be called on timeout")
				return nil
			})
			close(done)
		}()

		time.Sleep(50 * time.Millisecond)
		cancel()

		// Assert
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("handleConnection did not return after cancel")
		}
	})

	t.Run("Receive error", func(t *testing.T) {
		mock := &mockConn{
			readData:   []byte("trigger"),
			remoteAddr: &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 7777},
		}
		listener := &ListenerTCP{logger: slog.Default()}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		expectedErr := errors.New("callback failure")

		err := listener.handleConnection(ctx, mock, func(p []byte) error {
			return expectedErr
		})

		if err == nil || !errors.Is(err, expectedErr) {
			t.Fatalf("expected callback error, got: %v", err)
		}
	})
}

func TestListenerTCP_Write(t *testing.T) {
	t.Run("No connection", func(t *testing.T) {
		// Arrange
		listener := &ListenerTCP{logger: slog.Default()}

		// Act
		_, err := listener.Write([]byte("test"))

		// Assert
		if err == nil {
			t.Fatal("expected error due to no active connection")
		}
	})

	t.Run("Success", func(t *testing.T) {
		// Arrange
		mock := &mockConn{
			remoteAddr: &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 12345},
		}

		l := &ListenerTCP{
			conn:   mock,
			logger: slog.Default(),
		}

		// Act
		msg := []byte("hello world")
		n, err := l.Write(msg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Assert
		if n != len(msg) {
			t.Errorf("expected to write %d bytes, wrote %d", len(msg), n)
		}
		if mock.writeBuffer.String() != string(msg) {
			t.Errorf("expected message %q, got %q", msg, mock.writeBuffer.String())
		}
	})
}

func TestListenerTCP_Close(t *testing.T) {
	mock := &mockConn{
		remoteAddr: &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 5555},
	}
	ln := &ListenerTCP{
		conn:     mock,
		listener: &mockListener{acceptConns: make(chan net.Conn)},
		logger:   slog.Default(),
	}

	err := ln.Close()
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	if !mock.closed {
		t.Error("expected connection to be closed")
	}
}

func TestListenerTCP_ReceivesAndCallsCallback(t *testing.T) {
	sendConn, handleConn := net.Pipe()
	defer func() {
		_ = sendConn.Close()
		_ = handleConn.Close()
	}()

	mockLn := &mockListener{acceptConns: make(chan net.Conn, 1)}
	mockLn.acceptConns <- handleConn

	listener := &ListenerTCP{
		listener: mockLn,
		logger:   slog.Default(),
	}

	wg := &sync.WaitGroup{}
	wg.Add(1)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	done := make(chan struct{})
	go func() {
		err := listener.Run(ctx, func(p []byte) error {
			if string(p) != "ping" {
				t.Errorf("expected 'ping', got: %s", string(p))
			}
			close(done)
			return nil
		})
		if err != nil && !errors.Is(err, context.Canceled) {
			t.Errorf("Run returned unexpected error: %v", err)
		}
		wg.Done()
	}()

	time.Sleep(100 * time.Millisecond)
	if _, err := sendConn.Write([]byte("ping")); err != nil {
		t.Fatalf("unexpected error: %v", err)
		return
	}

	select {
	case <-done:
		// success
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for onReceive to be called")
	}

	cancel()
	wg.Wait()
}

func TestListenerTCP_Alive(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		want     bool
	}{
		{
			name:     "2 seconds before",
			duration: -2 * time.Second,
			want:     true,
		},
		{
			name:     "11 seconds before",
			duration: -11 * time.Second,
			want:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := time.Now()

			p := &ListenerTCP{
				conn:       &mockConn{},
				lastActive: now.Add(tt.duration),
			}
			if got := p.Alive(now); got != tt.want {
				t.Errorf("Alive() = %v, want %v", got, tt.want)
			}
		})
	}
}
