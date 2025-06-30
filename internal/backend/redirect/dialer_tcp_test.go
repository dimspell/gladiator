package redirect

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"testing"
	"time"
)

type mockTCPConn struct {
	readData        [][]byte
	writeData       [][]byte
	readIndex       int
	readDeadline    time.Time
	closed          bool
	remoteAddrValue net.Addr
}

func (m *mockTCPConn) Read(b []byte) (int, error) {
	if m.readIndex >= len(m.readData) {
		return 0, io.EOF
	}
	copy(b, m.readData[m.readIndex])
	n := len(m.readData[m.readIndex])
	m.readIndex++
	return n, nil
}

func (m *mockTCPConn) Write(b []byte) (int, error) {
	buf := make([]byte, len(b))
	copy(buf, b)
	m.writeData = append(m.writeData, buf)
	return len(b), nil
}

func (m *mockTCPConn) Close() error {
	m.closed = true
	return nil
}

func (m *mockTCPConn) SetReadDeadline(t time.Time) error {
	m.readDeadline = t
	return nil
}

func (m *mockTCPConn) RemoteAddr() net.Addr {
	return m.remoteAddrValue
}

func TestDialerTCP_Mock_Run_Write(t *testing.T) {
	mock := &mockTCPConn{
		readData: [][]byte{
			[]byte("server-payload"),
		},
		remoteAddrValue: &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 5001},
	}

	dialer := &DialerTCP{
		conn:   mock,
		logger: slog.Default(),
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var received []byte
	done := make(chan struct{})

	go func() {
		err := dialer.Run(ctx, func(p []byte) error {
			received = append([]byte{}, p...)
			close(done)
			return nil
		})
		if err != nil && !errors.Is(err, context.Canceled) && err != io.EOF {
			t.Errorf("unexpected Run error: %v", err)
		}
	}()

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for dialer to receive message")
	}

	if string(received) != "server-payload" {
		t.Fatalf("expected 'server-payload', got: %s", string(received))
	}

	n, err := dialer.Write([]byte("client-payload"))
	if err != nil {
		t.Fatalf("unexpected write error: %v", err)
	}
	if n != len("client-payload") {
		t.Fatalf("expected %d bytes written, got %d", len("client-payload"), n)
	}

	if len(mock.writeData) != 1 || string(mock.writeData[0]) != "client-payload" {
		t.Fatalf("unexpected write data: %v", mock.writeData)
	}
}

// func TestDialerTCP_Run_Write(t *testing.T) {
// 	server, clientDone := startTestTCPServer(t)
// 	defer clientDone()
//
// 	host, port, err := net.SplitHostPort(server)
// 	if err != nil {
// 		t.Fatal(err)
// 		return
// 	}
//
// 	dialer, err := DialTCP(host, port)
// 	if err != nil {
// 		t.Fatalf("failed to dial test server: %v", err)
// 	}
// 	defer dialer.Close()
//
// 	ctx, cancel := context.WithCancel(context.Background())
// 	defer cancel()
//
// 	var received []byte
// 	done := make(chan struct{})
//
// 	// Run the dialer in a goroutine
// 	go func() {
// 		err := dialer.Run(ctx, func(p []byte) error {
// 			received = append([]byte{}, p...)
// 			close(done) // signal we got the message
// 			return nil
// 		})
// 		if err != nil && !errors.Is(err, context.Canceled) {
// 			t.Errorf("Run returned unexpected error: %v", err)
// 		}
// 	}()
//
// 	select {
// 	case <-done:
// 	case <-time.After(2 * time.Second):
// 		t.Fatal("did not receive message in time")
// 	}
//
// 	if string(received) != "hello-from-server" {
// 		t.Fatalf("unexpected received data: got %q, want %q", string(received), "hello-from-server")
// 	}
//
// 	// Send data back to server
// 	n, err := dialer.Write([]byte("reply-from-client"))
// 	if err != nil {
// 		t.Fatalf("unexpected write error: %v", err)
// 	}
// 	if n != len("reply-from-client") {
// 		t.Fatalf("expected to write %d bytes, wrote %d", len("reply-from-client"), n)
// 	}
// }
//
// func startTestTCPServer(t *testing.T) (addr string, cleanup func()) {
// 	t.Helper()
//
// 	l, err := net.Listen("tcp", "127.0.0.1:5001")
// 	if err != nil {
// 		t.Fatalf("failed to start test TCP server: %v", err)
// 	}
//
// 	done := make(chan struct{})
// 	go func() {
// 		defer close(done)
//
// 		conn, err := l.Accept()
// 		if err != nil {
// 			t.Logf("test server accept failed: %v", err)
// 			return
// 		}
// 		defer conn.Close()
//
// 		// Send a message to the client
// 		_, _ = conn.Write([]byte("hello-from-server"))
//
// 		// Read response
// 		buf := make([]byte, 1024)
// 		n, err := conn.Read(buf)
// 		if err != nil {
// 			t.Logf("test server read failed: %v", err)
// 			return
// 		}
//
// 		if got := string(buf[:n]); got != "reply-from-client" {
// 			t.Errorf("test server received unexpected data: %s", got)
// 		}
// 	}()
//
// 	cleanup = func() {
// 		_ = l.Close()
// 		<-done
// 	}
// 	return "127.0.0.1:5001", cleanup
// }
