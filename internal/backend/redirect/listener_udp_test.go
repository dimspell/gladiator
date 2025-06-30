package redirect

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"sync"
	"testing"
	"time"
)

func TestListenerUDP_Run(t *testing.T) {
	t.Run("Using mock", func(t *testing.T) {
		conn := &fakeUDPConn{
			ReadData: [][]byte{
				[]byte("test-message"),
			},
		}

		listener := &ListenerUDP{
			logger: slog.Default(),
			conn:   conn,
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var received [][]byte
		var mu sync.Mutex
		go func() {
			time.Sleep(50 * time.Millisecond)
			cancel()
		}()

		err := listener.Run(ctx, func(p []byte) error {
			r := make([]byte, len(p))
			copy(r, p)
			mu.Lock()
			received = append(received, r)
			mu.Unlock()
			return nil
		})

		if err != nil && !errors.Is(err, context.Canceled) {
			t.Fatalf("unexpected run error: %v", err)
		}

		mu.Lock()
		if len(received) != 1 || string(received[0]) != "test-message" {
			t.Fatalf("unexpected received data: %v", received)
		}
		mu.Unlock()

		n, err := listener.Write([]byte("reply"))
		if err != nil {
			t.Fatalf("unexpected write error: %v", err)
		}
		if n != 5 {
			t.Fatalf("expected to write 5 bytes, wrote %d", n)
		}
	})

	t.Run("Real connection", func(t *testing.T) {
		listener, err := ListenUDP("127.0.0.1", "61100")
		if err != nil {
			t.Fatalf("failed to start listener: %v", err)
		}
		defer listener.Close()

		recvDone := make(chan struct{}, 1)
		expected := []byte("ping-pong")

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		wg := &sync.WaitGroup{}
		wg.Add(1)

		started := make(chan struct{}, 1)
		defer close(started)

		go func() {
			started <- struct{}{}
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
			wg.Done()
		}()

		// Wait for start
		<-started

		// Mimic sending a message from the game client
		gameClient, err := net.DialUDP("udp", nil, listener.conn.LocalAddr().(*net.UDPAddr))
		if err != nil {
			t.Fatalf("failed to create client UDP conn: %v", err)
		}
		defer gameClient.Close()

		_, err = gameClient.Write(expected)
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
	})
}
