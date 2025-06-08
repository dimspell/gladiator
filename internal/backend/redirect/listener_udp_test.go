package redirect

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"
)

func TestListenerUDP_RunAndWrite(t *testing.T) {
	listener, err := ListenUDP("127.0.0.1", "61100")
	if err != nil {
		t.Fatalf("failed to start listener: %v", err)
	}
	defer listener.Close()

	recvDone := make(chan struct{})
	expected := []byte("ping-pong")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		err := listener.Run(ctx, func(p []byte) error {
			if string(p) != string(expected) {
				t.Errorf("unexpected message: got %s, want %s", p, expected)
			}
			close(recvDone)
			return nil
		})
		if err != nil && !errors.Is(err, context.Canceled) {
			t.Errorf("listener run error: %v", err)
		}
	}()

	// Wait for start
	time.Sleep(100 * time.Millisecond)

	// Send a message
	clientConn, err := net.DialUDP("udp", nil, listener.conn.LocalAddr().(*net.UDPAddr))
	if err != nil {
		t.Fatalf("failed to create client UDP conn: %v", err)
	}
	defer clientConn.Close()

	_, err = clientConn.Write(expected)
	if err != nil {
		t.Fatalf("failed to send UDP message: %v", err)
	}

	select {
	case <-recvDone:
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for message to be received")
	}

	// Test Write()
	time.Sleep(100 * time.Millisecond)
	resp := []byte("response")
	n, err := listener.Write(resp)
	if err != nil {
		t.Fatalf("listener write failed: %v", err)
	}
	if n != len(resp) {
		t.Errorf("expected to write %d bytes, wrote %d", len(resp), n)
	}
}
