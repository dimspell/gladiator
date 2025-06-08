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

func TestListenerTCP_Write(t *testing.T) {
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
}

func TestListenerTCP_Run(t *testing.T) {
	t.Run("Got EOF", func(t *testing.T) {
		// Arrange
		mock := &mockConn{
			readErr:    io.EOF,
			remoteAddr: &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 2222},
		}
		l := &ListenerTCP{
			logger: slog.Default(),
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Act
		err := l.handleConnection(ctx, mock, func(p []byte) error {
			t.Fatal("should not be called")
			return nil
		})

		// Assert
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if !mock.closed {
			t.Error("expected connection to be closed")
		}
	})

	t.Run("Context canceled", func(t *testing.T) {
		mock := &mockConn{
			readData:   []byte("test"),
			remoteAddr: &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 4444},
		}
		listener := &ListenerTCP{logger: slog.Default()}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := listener.handleConnection(ctx, mock, func(p []byte) error {
			return nil
		})

		if !errors.Is(err, context.Canceled) {
			t.Errorf("expected context.Canceled, got: %v", err)
		}
	})
}

// mockListener is a stub for testing Close()
type mockListener struct{}

func (m *mockListener) Accept() (net.Conn, error) { return nil, nil }
func (m *mockListener) Close() error              { return nil }
func (m *mockListener) Addr() net.Addr            { return &net.TCPAddr{} }

func TestListenerTCP_Close(t *testing.T) {
	mock := &mockConn{
		remoteAddr: &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 5555},
	}
	ln := &ListenerTCP{
		conn:     mock,
		listener: &mockListener{},
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
