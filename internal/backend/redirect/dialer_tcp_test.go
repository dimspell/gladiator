package redirect

import (
	"context"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/dimspell/gladiator/internal/app/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---- Acceptance Tests ----

func startTestTCPServer(t *testing.T, handler func(conn net.Conn)) (addr string, stop func()) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go handler(conn)
		}
	}()
	return ln.Addr().String(), func() { ln.Close() }
}

func TestDialTCP_SuccessAndClose(t *testing.T) {
	addr, stop := startTestTCPServer(t, func(conn net.Conn) { conn.Close() })
	defer stop()

	dialer, err := NewDialTCP("127.0.0.1", addr[strings.LastIndex(addr, ":")+1:], func(p []byte) error { return nil })
	require.NoError(t, err)
	require.NotNil(t, dialer)
	require.NoError(t, dialer.Close())
	require.NoError(t, dialer.Close()) // double close should not error
}

func TestDialTCP_Failure(t *testing.T) {
	_, err := NewDialTCP("256.256.256.256", "9999", func(p []byte) error { return nil })
	require.Error(t, err)
}

func TestWriteAndRead(t *testing.T) {
	addr, stop := startTestTCPServer(t, func(conn net.Conn) {
		buf := make([]byte, 5)
		n, _ := conn.Read(buf)
		conn.Write([]byte("pong"))
		require.Equal(t, "ping", string(buf[:n]))
		conn.Close()
	})
	defer stop()

	dialer, err := NewDialTCP("127.0.0.1", addr[strings.LastIndex(addr, ":")+1:], func(p []byte) error { return nil })
	require.NoError(t, err)
	n, err := dialer.Write([]byte("ping"))
	require.NoError(t, err)
	require.Equal(t, 4, n)
	buf := make([]byte, 4)
	_, err = dialer.conn.Read(buf)
	require.NoError(t, err)
	require.Equal(t, "pong", string(buf))
	dialer.Close()
}

func TestRun_ContextCancel(t *testing.T) {
	addr, stop := startTestTCPServer(t, func(conn net.Conn) {
		time.Sleep(2 * time.Second)
		conn.Close()
	})
	defer stop()

	dialer, err := NewDialTCP("127.0.0.1", addr[strings.LastIndex(addr, ":")+1:], func(p []byte) error { return nil })
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	err = dialer.Run(ctx)
	require.Error(t, err)
	// require.Contains(t, err.Error(), "context canceled") // TODO: it is io.EOF
}

func TestRun_OnReceiveError(t *testing.T) {
	addr, stop := startTestTCPServer(t, func(conn net.Conn) {
		conn.Write([]byte("data"))
		time.Sleep(100 * time.Millisecond)
		conn.Close()
	})
	defer stop()

	dialer, err := NewDialTCP("127.0.0.1", addr[strings.LastIndex(addr, ":")+1:], func(p []byte) error { return assert.AnError })
	require.NoError(t, err)
	err = dialer.Run(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to handle data")
}

// ---- Mocks ----

type mockTCPConn struct {
	readData  [][]byte
	writeData [][]byte
	readIndex int
	closed    bool
}

func (m *mockTCPConn) Read(b []byte) (int, error) {
	if m.closed {
		return 0, io.EOF
	}
	if m.readIndex >= len(m.readData) {
		return 0, io.EOF
	}
	copy(b, m.readData[m.readIndex])
	n := len(m.readData[m.readIndex])
	m.readIndex++
	return n, nil
}
func (m *mockTCPConn) Write(b []byte) (int, error) {
	if m.closed {
		return 0, io.ErrClosedPipe
	}
	m.writeData = append(m.writeData, append([]byte{}, b...))
	return len(b), nil
}
func (m *mockTCPConn) Close() error                      { m.closed = true; return nil }
func (m *mockTCPConn) SetReadDeadline(_ time.Time) error { return nil }
func (m *mockTCPConn) RemoteAddr() net.Addr              { return nil }

// ---- Unit Tests ----

func TestDialerTCP_Close_Idempotent(t *testing.T) {
	mock := &mockTCPConn{}
	dialer := &DialerTCP{conn: mock, logger: logger.NewDiscardLogger()}
	require.NoError(t, dialer.Close())
	require.NoError(t, dialer.Close()) // Should not error
}

func TestDialerTCP_Write(t *testing.T) {
	mock := &mockTCPConn{}
	dialer := &DialerTCP{conn: mock, logger: logger.NewDiscardLogger()}
	n, err := dialer.Write([]byte("hello"))
	require.NoError(t, err)
	require.Equal(t, 5, n)
	require.Equal(t, "hello", string(mock.writeData[0]))
}

func TestDialerTCP_Write_AfterClose(t *testing.T) {
	mock := &mockTCPConn{}
	dialer := &DialerTCP{conn: mock, logger: logger.NewDiscardLogger()}
	_ = dialer.Close()
	_, err := dialer.Write([]byte("fail"))
	require.Error(t, err)
}

func TestDialerTCP_Run_NilConn(t *testing.T) {
	dialer := &DialerTCP{conn: nil, logger: logger.NewDiscardLogger(), OnReceive: func(p []byte) error { return nil }}
	err := dialer.Run(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "tcp connection is nil")
}

func TestDialerTCP_Run_OnReceiveError(t *testing.T) {
	mock := &mockTCPConn{readData: [][]byte{[]byte("data")}}
	dialer := &DialerTCP{conn: mock, logger: logger.NewDiscardLogger(), OnReceive: func(p []byte) error { return assert.AnError }}
	err := dialer.Run(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to handle data")
}
