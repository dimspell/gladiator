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

	"github.com/dimspell/gladiator/internal/app/logger"
	"github.com/stretchr/testify/require"
)

// ---- MOCK IMPLEMENTATIONS ----

type mockConn struct {
	readData    []byte
	writeBuffer bytes.Buffer
	readErr     error
	writeErr    error
	closed      bool
	setDeadline bool
}

func (m *mockConn) Read(b []byte) (int, error) {
	if m.closed {
		return 0, io.EOF
	}
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

type mockTCPListener struct {
	conn   net.Conn
	closed bool
}

func (m *mockTCPListener) Accept() (net.Conn, error) {
	if m.closed {
		return nil, io.EOF
	}
	return m.conn, nil
}
func (m *mockTCPListener) Close() error   { m.closed = true; return nil }
func (m *mockTCPListener) Addr() net.Addr { return &net.TCPAddr{} }

// timeoutErr implements net.De
type timeoutErr struct{}

func (timeoutErr) Error() string   { return "timeout" }
func (timeoutErr) Timeout() bool   { return true }
func (timeoutErr) Temporary() bool { return true }

func (timeoutErr) Unwrap() error { return nil }

// ---- UNIT TESTS ----

func TestListenerTCP_Write2(t *testing.T) {
	mockConn := &mockTCPConn{}
	listener := &ListenerTCP{conn: mockConn}
	n, err := listener.Write([]byte("hello"))
	require.NoError(t, err)
	require.Equal(t, 5, n)
	require.Equal(t, "hello", string(mockConn.writeData[0]))
}

func TestListenerTCP_Write_NoConn(t *testing.T) {
	listener := &ListenerTCP{}
	_, err := listener.Write([]byte("fail"))
	require.Error(t, err)
}

func TestListenerTCP_Close_Idempotent(t *testing.T) {
	mockListener := &mockTCPListener{}
	listener := &ListenerTCP{listener: mockListener, logger: logger.NewDiscardLogger()}
	require.NoError(t, listener.Close())
	require.NoError(t, listener.Close()) // Should not error
}

func TestListenerTCP_handleHandshake_Valid(t *testing.T) {
	mockConn := &mockTCPConn{readData: [][]byte{[]byte("##username")}}
	listener := &ListenerTCP{logger: logger.NewDiscardLogger()}
	err := listener.handleHandshake(mockConn)
	require.NoError(t, err)
	require.Equal(t, mockConn, listener.conn)
}

func TestListenerTCP_handleHandshake_Invalid(t *testing.T) {
	mockConn := &mockTCPConn{readData: [][]byte{[]byte("bad")}}
	listener := &ListenerTCP{logger: logger.NewDiscardLogger()}
	err := listener.handleHandshake(mockConn)
	require.Error(t, err)
}

func TestListenerTCP_Run(t *testing.T) {
	t.Run("Got EOF", func(t *testing.T) {
		// Arrange
		mock := &mockConn{
			readErr: io.EOF,
		}
		listener := &ListenerTCP{logger: slog.Default()}

		// Act
		err := listener.handleConnection(context.Background(), mock, func(p []byte) error {
			t.Fatal("should not be called")
			return nil
		})

		// Assert
		if err == nil || !errors.Is(err, io.EOF) {
			t.Errorf("expected error io.EOF, got: %v", err)
		}
	})

	t.Run("Context canceled on listener", func(t *testing.T) {
		mockLn := &mockListener{acceptConns: make(chan net.Conn)}

		listener := &ListenerTCP{
			listener: mockLn,
			logger:   slog.Default(),
			OnReceive: func(p []byte) error {
				t.Fatal("onReceive should not be called")
				return nil
			},
		}

		ctx, cancel := context.WithCancel(context.Background())

		// cause Accept to return error
		go func() {
			time.Sleep(100 * time.Millisecond)
			cancel()
		}()

		err := listener.Run(ctx)

		if !errors.Is(err, context.Canceled) {
			t.Errorf("expected context.Canceled, got: %v", err)
		}
	})

	t.Run("Continue on deadline exceeded", func(t *testing.T) {
		// Arrange
		mock := &mockConn{
			readErr:  &timeoutErr{},
			readData: []byte("ignored"), // won't be used
		}
		listener := &ListenerTCP{logger: slog.Default()}

		// Act
		done := make(chan struct{})
		errCh := make(chan string, 1)
		go func() {
			// only allow a short loop
			_ = listener.handleConnection(context.Background(), mock, func(p []byte) error {
				errCh <- "should not be called on timeout"
				return nil
			})
			close(done)
		}()

		time.Sleep(50 * time.Millisecond)
		_ = mock.Close()

		// Assert
		select {
		case msg := <-errCh:
			if msg != "" {
				t.Fatal(msg)
			}
		case <-time.After(time.Second):
			// test passed, no error
		}
	})

	t.Run("Receive error", func(t *testing.T) {
		mock := &mockConn{
			readData: []byte("trigger"),
		}
		listener := &ListenerTCP{logger: slog.Default()}

		expectedErr := errors.New("callback failure")

		err := listener.handleConnection(context.Background(), mock, func(p []byte) error {
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
		mock := &mockConn{}

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
	mock := &mockConn{}
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

	done := make(chan struct{})
	listener := &ListenerTCP{
		listener: mockLn,
		logger:   slog.Default(),
		OnReceive: func(p []byte) error {
			if string(p) != "ping" {
				t.Errorf("expected 'ping', got: %s", string(p))
			}
			close(done)
			return nil
		},
	}

	wg := &sync.WaitGroup{}
	wg.Add(1)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	go func() {
		err := listener.Run(ctx)
		if err != nil && !errors.Is(err, io.ErrClosedPipe) {
			t.Errorf("Run returned unexpected error: %v", err)
		}
		wg.Done()
	}()

	time.Sleep(100 * time.Millisecond)
	if _, err := sendConn.Write([]byte("##testuser")); err != nil {
		t.Fatalf("unexpected error: %v", err)
		return
	}

	time.Sleep(5 * time.Millisecond)
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
			if got := p.Alive(now, 5*time.Second); got != tt.want {
				t.Errorf("Alive() = %v, want %v", got, tt.want)
			}
		})
	}
}

// ---- Acceptance Tests ----

func TestListenerTCP_Acceptance(t *testing.T) {
	t.Skip("Failing - needs to be fixed")
	var received []string
	done := make(chan struct{})

	listener, err := NewListenerTCP("127.0.0.1", "1234", func(p []byte) error {
		received = append(received, string(p))
		if string(p) == "payload" {
			close(done)
		}
		return nil
	})
	require.NoError(t, err)
	addr := listener.listener.Addr().String()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go func() {
		err := listener.Run(ctx)
		require.NoError(t, err)
	}()

	// Simulate a client dialing and sending handshake + payload
	conn, err := net.Dial("tcp", addr)
	require.NoError(t, err)
	defer conn.Close()

	// Send handshake
	_, err = conn.Write([]byte("##username"))
	require.NoError(t, err)
	time.Sleep(50 * time.Millisecond) // Give server time to process handshake

	// Send payload
	_, err = conn.Write([]byte("payload"))
	require.NoError(t, err)

	select {
	case <-done:
		require.Contains(t, received, "payload")
	case <-time.After(time.Second):
		t.Fatal("timeout: server did not receive payload")
	}

	_ = listener.Close()
}
