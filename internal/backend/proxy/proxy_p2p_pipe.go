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

type Pipe struct {
	dc     DataChannel
	done   func()
	proxy  redirect.Redirect
	logger *slog.Logger
}

type DataChannel interface {
	Label() string
	OnError(func(err error))
	OnMessage(func(msg webrtc.DataChannelMessage))
	OnClose(f func())
	Send([]byte) error

	io.Closer
}

func NewPipe(ctx context.Context, dc DataChannel, proxy redirect.Redirect) *Pipe {
	// FIXME: Return an error instead of panicking
	if proxy == nil {
		panic("proxy is nil")
	}

	pipe := &Pipe{
		dc:     dc,
		proxy:  proxy,
		logger: slog.With("label", dc.Label()),
	}

	// FIXME: Pass the context from the caller
	ctx, cancel := context.WithCancel(ctx)
	pipe.done = cancel

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		return proxy.Run(ctx, pipe)
	})
	g.Go(func() error {
		select {
		case <-ctx.Done():
			slog.Debug("context done", "error", ctx.Err())
			return ctx.Err()
		}
	})
	go func() {
		if err := g.Wait(); err != nil {
			pipe.logger.Warn("proxy has failed", "error", err)
			cancel()
		}
	}()

	dc.OnError(func(err error) {
		pipe.logger.Warn("datachannel reports an error", "error", err)
	})
	dc.OnClose(func() {
		if err := pipe.Close(); err != nil {
			pipe.logger.Error("could not close the pipe after the datachannel has closed", "error", err)
		}
		cancel()
	})

	dc.OnMessage(func(msg webrtc.DataChannelMessage) {
		if _, err := proxy.Write(msg.Data); err != nil {
			pipe.logger.Warn("could not write to the proxy", "error", err, "data", msg.Data)
			return
		}
	})

	return pipe
}

func (pipe *Pipe) Write(p []byte) (n int, err error) {
	pipe.logger.Debug("pipe sending data to data channel", "data", p)

	// Proxy -> DataChannel
	if err := pipe.dc.Send(p); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (pipe *Pipe) Close() error {
	pipe.done()
	return nil
}
