package proxy

import (
	"context"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/dimspell/gladiator/internal/app/logger"
	"github.com/pion/webrtc/v4"
	"go.uber.org/goleak"
	"golang.org/x/sync/errgroup"
)

type mockDataChannel struct {
	label     string
	onMessage func(msg webrtc.DataChannelMessage)
	onClose   func()
	onError   func(err error)
	closed    bool
	received  chan []byte
}

func (m *mockDataChannel) Label() string                                   { return m.label }
func (m *mockDataChannel) OnMessage(f func(msg webrtc.DataChannelMessage)) { m.onMessage = f }
func (m *mockDataChannel) OnClose(f func())                                { m.onClose = f }
func (m *mockDataChannel) OnError(f func(err error))                       { m.onError = f }
func (m *mockDataChannel) Send(data []byte) error {
	m.received <- data
	return nil
}
func (m *mockDataChannel) Close() error {
	m.closed = true
	if m.onClose != nil {
		close(m.received)
		m.onClose()
	}
	return nil
}
func (m *mockDataChannel) receive(t *testing.T, data []byte) {
	t.Logf("Writing data to data channel (data=%q)", string(data))
	m.onMessage(webrtc.DataChannelMessage{Data: data})
}

type mockRedirect struct {
	fromDataChannel chan []byte
	fromProxy       chan []byte
	t               *testing.T
}

func (m *mockRedirect) Run(ctx context.Context, dc io.Writer) error {
	g, ctx := errgroup.WithContext(ctx)

	// Writer
	// g.Go(func() error {
	// 	for {
	// 		data := make([]byte, 1024)
	// 		n, err := dc.Read(data)
	// 		if err != nil {
	// 			return err
	// 		}
	// 		m.t.Logf("reading data from redirect (data=%q)", string(data[:n]))
	// 		m.fromProxy <- data[:n]
	// 	}
	// })
	// Reader
	g.Go(func() error {
		// Proxy -> DataChannel
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case data := <-m.fromProxy:
				m.t.Logf("sending data to redirect (data=%q)", string(data))
				_, err := dc.Write(data)
				if err != nil {
					m.t.Errorf("Error writing data to redirect: %v", err)
					return err
				}
			}
		}
	})
	return g.Wait()
}

func (m *mockRedirect) Close() error {
	close(m.fromProxy)
	close(m.fromDataChannel)
	return nil
}

func (m *mockRedirect) Write(p []byte) (n int, err error) {
	// DataChannel -> Proxy
	m.fromDataChannel <- p
	return n, nil
}

func TestPipeMessageHandling(t *testing.T) {
	defer goleak.VerifyNone(t)

	logger.SetPlainTextLogger(os.Stderr, slog.LevelDebug)

	t.Run("handles incoming messages", func(t *testing.T) {
		dc := &mockDataChannel{
			label:    "test",
			received: make(chan []byte, 1),
		}
		// defer dc.Close()

		proxy := &mockRedirect{
			t:               t,
			fromDataChannel: make(chan []byte, 1),
			fromProxy:       make(chan []byte, 1),
		}
		// defer proxy.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		pipe := NewPipe(ctx, dc, proxy)
		defer pipe.Close()

		testData := []byte("test message")

		dc.receive(t, testData)

		testMsg := "message from the proxy"
		proxy.fromProxy <- []byte(testMsg)

		msg := string(<-dc.received)
		t.Logf("DataChannel received message from the redirect (data=%q)", msg)
		if msg != testMsg {
			t.Errorf("Unexpected message received from the redirect (data=%q)", msg)
		}
	})
}
