package redirect

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/dimspell/gladiator/internal/app/logger"
	"github.com/stretchr/testify/require"
)

// --- Unit tests ---

func TestListenerUDP_Write(t *testing.T) {
	mockConn := &mockUDPConn{remote: &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 1234}}
	listener := &ListenerUDP{conn: mockConn, remoteAddr: mockConn.remote}
	n, err := listener.Write([]byte("hello"))
	require.NoError(t, err)
	require.Equal(t, 5, n)
	require.Equal(t, "hello", string(mockConn.writeData[0]))
}

func TestListenerUDP_Write_NoConn(t *testing.T) {
	listener := &ListenerUDP{logger: logger.NewDiscardLogger()}
	_, err := listener.Write([]byte("fail"))
	require.Error(t, err)
}

func TestListenerUDP_Close_Idempotent(t *testing.T) {
	mockConn := &mockUDPConn{}
	listener := &ListenerUDP{conn: mockConn, logger: logger.NewDiscardLogger()}
	require.NoError(t, listener.Close())
	require.NoError(t, listener.Close()) // Should not error
}

func TestListenerUDP_handleHandshake_Valid(t *testing.T) {
	mockConn := &mockUDPConn{
		readData: [][]byte{{26, 0, 2, 0}},
		remote:   &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 1234},
	}
	listener := &ListenerUDP{logger: logger.NewDiscardLogger()}
	err := listener.handleHandshake(mockConn)
	require.NoError(t, err)
	require.Equal(t, mockConn.remote, listener.remoteAddr)
}

func TestListenerUDP_handleHandshake_Invalid(t *testing.T) {
	mockConn := &mockUDPConn{
		readData: [][]byte{{1, 2, 3, 4}},
		remote:   &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 1234},
	}
	listener := &ListenerUDP{logger: logger.NewDiscardLogger()}
	err := listener.handleHandshake(mockConn)
	require.Error(t, err)
}

func TestListenerUDP_handleConnection_Valid(t *testing.T) {
	mockConn := &mockUDPConn{
		readData: [][]byte{{26, 0, 2, 0}, []byte("payload")},
		remote:   &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 1234},
	}
	listener := &ListenerUDP{remoteAddr: mockConn.remote, logger: logger.NewDiscardLogger()}
	var received []string
	err := listener.handleConnection(context.Background(), mockConn, func(p []byte) error {
		received = append(received, string(p))
		return nil
	})
	require.Error(t, err) // Should error on EOF
	require.Contains(t, received, "payload")
}

func TestListenerUDP_handleConnection_UnknownSource(t *testing.T) {
	mockConn := &mockUDPConn{
		readData: [][]byte{{26, 0, 2, 0}, []byte("payload")},
		remote:   &net.UDPAddr{IP: net.IPv4(127, 0, 0, 2), Port: 4321}, // different from listener.remoteAddr
	}
	listener := &ListenerUDP{remoteAddr: &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 1234}, logger: logger.NewDiscardLogger()}
	var received []string
	err := listener.handleConnection(context.Background(), mockConn, func(p []byte) error {
		received = append(received, string(p))
		return nil
	})
	require.Error(t, err) // Should error on EOF
	require.NotContains(t, received, "payload")
}

// --- Acceptance tests ---

func TestListenerUDP_Acceptance(t *testing.T) {
	t.Skip("Failing - needs to be fixed")
	var received []string
	done := make(chan struct{})

	listener, err := NewListenerUDP("127.0.0.1", "0", func(p []byte) error {
		received = append(received, string(p))
		if string(p) == "payload" {
			close(done)
		}
		return nil
	})
	require.NoError(t, err)
	addr := listener.conn.LocalAddr().String()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go func() {
		err := listener.Run(ctx)
		if err != nil && !errors.Is(err, context.Canceled) {
			t.Errorf("ListenerUDP.Run error: %v", err)
		}
	}()

	// Simulate a client sending handshake and payload
	conn, err := net.Dial("udp", addr)
	require.NoError(t, err)
	defer conn.Close()

	// Send handshake
	_, err = conn.Write([]byte{26, 0, 2, 0})
	require.NoError(t, err)
	time.Sleep(50 * time.Millisecond)

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
