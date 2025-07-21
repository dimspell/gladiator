package redirect

import (
	"context"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"github.com/dimspell/gladiator/internal/app/logger"
	"github.com/stretchr/testify/require"
)

// ---- Mocks ----

type mockUDPConn struct {
	readData  [][]byte
	writeData [][]byte
	readIndex int
	closed    bool
	remote    *net.UDPAddr
}

func (m *mockUDPConn) ReadFromUDP(b []byte) (int, *net.UDPAddr, error) {
	if m.closed {
		return 0, nil, io.EOF
	}
	if m.readIndex >= len(m.readData) {
		return 0, nil, io.EOF
	}
	copy(b, m.readData[m.readIndex])
	n := len(m.readData[m.readIndex])
	addr := m.remote
	m.readIndex++
	return n, addr, nil
}
func (m *mockUDPConn) Write(b []byte) (int, error) {
	if m.closed {
		return 0, io.ErrClosedPipe
	}
	if string(b) == "fail" {
		return 0, io.ErrUnexpectedEOF
	}
	m.writeData = append(m.writeData, append([]byte{}, b...))
	return len(b), nil
}
func (m *mockUDPConn) WriteTo(b []byte, addr net.Addr) (int, error) { return m.Write(b) }
func (m *mockUDPConn) Close() error                                 { m.closed = true; return nil }
func (m *mockUDPConn) SetReadDeadline(t time.Time) error            { return nil }
func (m *mockUDPConn) LocalAddr() net.Addr                          { return nil }
func (m *mockUDPConn) RemoteAddr() net.Addr                         { return nil }

// ---- Unit Tests ----

func TestDialerUDP_Close_Idempotent(t *testing.T) {
	mock := &mockUDPConn{}
	dialer := &DialerUDP{conn: mock, logger: logger.NewDiscardLogger()}
	require.NoError(t, dialer.Close())
	require.NoError(t, dialer.Close()) // Should not error
}

func TestDialerUDP_Write(t *testing.T) {
	mock := &mockUDPConn{}
	dialer := &DialerUDP{conn: mock, logger: logger.NewDiscardLogger()}
	n, err := dialer.Write([]byte("hello"))
	require.NoError(t, err)
	require.Equal(t, 5, n)
	require.Equal(t, "hello", string(mock.writeData[0]))
}

func TestDialerUDP_Write_AfterClose(t *testing.T) {
	mock := &mockUDPConn{}
	dialer := &DialerUDP{conn: mock, logger: logger.NewDiscardLogger()}
	_ = dialer.Close()
	_, err := dialer.Write([]byte("fail"))
	require.Error(t, err)
}

func TestDialerUDP_Run_NilConn(t *testing.T) {
	dialer := &DialerUDP{conn: nil, logger: logger.NewDiscardLogger()}
	err := dialer.Run(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "UDP connection is nil")
}

func TestDialerUDP_Run_OnReceiveError(t *testing.T) {
	mock := &mockUDPConn{readData: [][]byte{[]byte("data")}}
	dialer := &DialerUDP{conn: mock, logger: logger.NewDiscardLogger(), OnReceive: func(p []byte) error { return errors.New("fail") }}
	err := dialer.Run(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to handle data")
}

func TestDialerUDP_Run_EOF(t *testing.T) {
	count := 0
	mock := &mockUDPConn{readData: [][]byte{[]byte("msg")}}
	dialer := &DialerUDP{conn: mock, logger: logger.NewDiscardLogger(), OnReceive: func(p []byte) error {
		count++
		return nil
	}}
	// After one message, mock returns EOF
	err := dialer.Run(context.Background())
	require.Error(t, err)
	require.Equal(t, 1, count)
}

func TestDialerUDP_WriteTo(t *testing.T) {
	mock := &mockUDPConn{}
	dialer := &DialerUDP{conn: mock, logger: logger.NewDiscardLogger()}
	n, err := dialer.conn.WriteTo([]byte("hello"), &net.UDPAddr{})
	require.NoError(t, err)
	require.Equal(t, 5, n)
	require.Equal(t, "hello", string(mock.writeData[0]))
}

func TestDialerUDP_Close_AfterAlreadyClosed(t *testing.T) {
	mock := &mockUDPConn{}
	dialer := &DialerUDP{conn: mock, logger: logger.NewDiscardLogger()}
	require.NoError(t, dialer.Close())
	require.NoError(t, dialer.Close()) // Should not error
}

func TestDialerUDP_Run_HandlerPanic(t *testing.T) {
	mock := &mockUDPConn{readData: [][]byte{[]byte("panic")}}
	dialer := &DialerUDP{conn: mock, logger: logger.NewDiscardLogger(), OnReceive: func(p []byte) error {
		panic("handler panic")
	}}
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic to propagate")
		}
	}()
	_ = dialer.Run(context.Background())
}

func TestDialerUDP_Run_Timeout(t *testing.T) {
	mock := &mockUDPConn{}
	dialer := &DialerUDP{conn: mock, logger: logger.NewDiscardLogger(), OnReceive: func(p []byte) error { return nil }}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	err := dialer.Run(ctx)
	require.Error(t, err)
}

func TestDialerUDP_Write_Error(t *testing.T) {
	mock := &mockUDPConn{}
	dialer := &DialerUDP{conn: mock, logger: logger.NewDiscardLogger()}
	_, err := dialer.Write([]byte("fail"))
	require.Error(t, err)
}

// ---- Acceptance Tests ----

func startTestUDPServer(t *testing.T, handler func(conn *net.UDPConn, addr *net.UDPAddr, data []byte)) (addr string, stop func()) {
	udpAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	require.NoError(t, err)
	conn, err := net.ListenUDP("udp", udpAddr)
	require.NoError(t, err)
	done := make(chan struct{})
	go func() {
		defer close(done)
		buf := make([]byte, 1024)
		for {
			n, addr, err := conn.ReadFromUDP(buf)
			if err != nil {
				return
			}
			handler(conn, addr, buf[:n])
		}
	}()
	return conn.LocalAddr().String(), func() { conn.Close(); <-done }
}

func TestDialUDP_SuccessAndClose(t *testing.T) {
	addr, stop := startTestUDPServer(t, func(conn *net.UDPConn, addr *net.UDPAddr, data []byte) {
		conn.WriteTo([]byte("pong"), addr)
	})
	defer stop()

	host, port, _ := net.SplitHostPort(addr)
	dialer, err := NewDialUDP(host, port, func(p []byte) error { return nil })
	require.NoError(t, err)
	n, err := dialer.Write([]byte("ping"))
	require.NoError(t, err)
	require.Equal(t, 4, n)
	buf := make([]byte, 4)
	dialer.conn.SetReadDeadline(time.Now().Add(time.Second))
	_, _, err = dialer.conn.ReadFromUDP(buf)
	require.NoError(t, err)
	require.Equal(t, "pong", string(buf))
	dialer.Close()
}
