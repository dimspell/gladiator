package proxy

import (
	"context"
	"io"
	"log/slog"

	"github.com/dimspell/gladiator/internal/backend/redirect"
	"github.com/pion/webrtc/v4"
	"golang.org/x/sync/errgroup"
)

// Define custom error types
// var (
// 	ErrReadTimeout = errors.New("read timeout")
// 	ErrClosedPipe = errors.New("pipe closed")
// )

var _ io.ReadWriteCloser = (*Pipe)(nil)

type Pipe struct {
	dc     DataChannel
	done   func()
	dcData chan webrtc.DataChannelMessage
	proxy  redirect.Redirect
}

type DataChannel interface {
	Label() string
	OnError(func(err error))
	OnMessage(func(msg webrtc.DataChannelMessage))
	OnClose(f func())
	Send([]byte) error

	io.Closer
}

func NewPipe(dc DataChannel, proxy redirect.Redirect) *Pipe {
	if proxy == nil {
		panic("proxy is nil")
	}

	pipe := &Pipe{
		dc:     dc,
		proxy:  proxy,
		dcData: make(chan webrtc.DataChannelMessage),
		// dcData: make(chan webrtc.DataChannelMessage, 1),
	}

	ctx, cancel := context.WithCancel(context.TODO())
	pipe.done = cancel

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		return proxy.Run(ctx, pipe)
	})
	g.Go(func() error {
		for {
			select {
			case <-ctx.Done():
				slog.Debug("context done", "error", ctx.Err())
				return ctx.Err()
			case msg := <-pipe.dcData:
				slog.Debug("Pipe.OnMessage.select", "data", msg.Data, "channel", pipe.dc.Label())

				if pipe.dcData == nil {
					return nil
				}

				if _, err := proxy.Write(msg.Data); err != nil {
					slog.Warn("Failed to send data to peer", "error", err)
					return err
				}
			}
		}
	})
	go func() {
		if err := g.Wait(); err != nil {
			slog.Warn("Error running proxy", "error", err)
			cancel()
		}
	}()

	dc.OnError(func(err error) {
		slog.Warn("Data channel error", "error", err)
	})
	dc.OnClose(func() {
		if err := pipe.Close(); err != nil {
			slog.Error("Error closing pipe", "error", err)
		}
		cancel()
	})

	dc.OnMessage(func(msg webrtc.DataChannelMessage) {
		slog.Debug("Pipe.OnMessage.callback", "data", msg.Data, "channel", pipe.dc.Label())

		if pipe.dcData == nil {
			return
		}
		pipe.dcData <- msg
	})

	return pipe
}

// Read from the DC channel
func (pipe *Pipe) Read(p []byte) (n int, err error) {
	if pipe.dcData == nil {
		slog.Warn("DC channel closed", "channel", pipe.dc.Label())
		return 0, io.EOF
	}

	// TODO: Handle timeout
	select {
	case msg := <-pipe.dcData:
		if len(msg.Data) == 0 {
			slog.Debug("pipe read operation",
				"bytes", len(msg.Data),
				"channel", pipe.dc.Label(),
				"data", msg.Data,
			)

			return 0, io.EOF
		}
		slog.Debug("pipe read operation",
			"bytes", len(p),
			"channel", pipe.dc.Label(),
			"data", msg.Data,
		)

		copy(p, msg.Data)
		return len(msg.Data), nil
		// case <-time.After(readTimeout):
		// 	return 0, ErrReadTimeout
	}
}

func (pipe *Pipe) Write(p []byte) (n int, err error) {
	slog.Debug("pipe write operation",
		"bytes", len(p),
		"channel", pipe.dc.Label(),
		"data", p,
	)

	if err := pipe.dc.Send(p); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (pipe *Pipe) Close() error {
	pipe.done()

	// Add timeout context for cleanup
	// ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	// defer cancel()
	// select {
	// case <-ctx.Done():
	// 	return ctx.Err()
	// default:
	// 	close(pipe.dcData)
	// 	pipe.dcData = nil
	// }

	close(pipe.dcData)
	pipe.dcData = nil
	return nil
}
