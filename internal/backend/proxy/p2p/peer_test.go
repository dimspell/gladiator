package p2p

import (
	"context"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/dimspell/gladiator/internal/app/logger"
	"github.com/pion/webrtc/v4"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/errgroup"
)

type mockDataChannel struct {
	label     string
	onMessage func(msg webrtc.DataChannelMessage)
	onClose   func()
	onError   func(err error)
	closed    bool

	received chan []byte
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

type mockRedirect struct {
	toProxy       chan []byte
	toDataChannel chan []byte
	t             *testing.T
}

func (m *mockRedirect) Run(ctx context.Context, dc io.Writer) error {
	g, ctx := errgroup.WithContext(ctx)

	// Proxy -> DataChannel
	g.Go(func() error {
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case data := <-m.toDataChannel:
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
	close(m.toDataChannel)
	close(m.toProxy)
	m.toDataChannel = nil
	m.toProxy = nil
	return nil
}

func (m *mockRedirect) Write(p []byte) (n int, err error) {
	// DataChannel -> Proxy
	m.toProxy <- p
	return n, nil
}

func TestNewPipe(t *testing.T) {
	// TODO: fixme pion's webrtc has a leak
	// defer goleak.VerifyNone(t)

	logger.SetPlainTextLogger(os.Stderr, slog.LevelDebug)

	t.Run("handles incoming messages", func(t *testing.T) {
		dc := &mockDataChannel{
			label:    "test",
			received: make(chan []byte, 1),
		}
		defer dc.Close()

		proxy := &mockRedirect{
			t:             t,
			toProxy:       make(chan []byte, 1),
			toDataChannel: make(chan []byte, 1),
		}
		defer proxy.Close()

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		pipe := NewPipe(ctx, dc, proxy)
		defer pipe.Close()

		// Test that the DataChannel receives messages from the proxy
		msgFromProxy := "Message from the Proxy"
		proxy.toDataChannel <- []byte(msgFromProxy)
		select {
		case msg := <-dc.received:
			assert.Equal(t, msgFromProxy, string(msg))
		case <-ctx.Done():
			t.Fatal("timeout")
		}

		// Test that the DataChannel sends messages to the proxy
		msgFromDataChannel := "Message from the DataChannel"
		dc.onMessage(webrtc.DataChannelMessage{Data: []byte(msgFromDataChannel)})
		select {
		case msg := <-proxy.toProxy:
			assert.Equal(t, msgFromDataChannel, string(msg))
		case <-ctx.Done():
			t.Fatal("timeout")
		}
	})
}
