package redirect

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
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
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if !mock.closed {
			t.Error("expected connection to be closed")
		}
	})

	t.Run("Context canceled", func(t *testing.T) {
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

	// _ = handleConn.SetDeadline(time.Now().Add(time.Second))

	mockLn := &mockListener{acceptConns: make(chan net.Conn, 1)}
	mockLn.acceptConns <- handleConn

	listener := &ListenerTCP{
		listener: mockLn,
		logger:   slog.Default(),
	}

	ctx, cancel := context.WithCancel(context.Background())
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
	}()

	time.Sleep(100 * time.Millisecond)
	sendConn.Write([]byte("ping"))

	select {
	case <-done:
		// success
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for onReceive to be called")
	}

	cancel()
	time.Sleep(50 * time.Millisecond)
}
