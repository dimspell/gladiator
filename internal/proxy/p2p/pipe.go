package p2p

import (
	"context"
	"io"
	"log"
	"log/slog"

	"github.com/dimspell/gladiator/internal/proxy/redirect"
	"github.com/pion/webrtc/v4"
	"golang.org/x/sync/errgroup"
)

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
			log.Println("Pipe.Read", msg.Data, pipe.dc.Label())
			return 0, io.EOF
		}
		log.Println("Pipe.Read", msg.Data, len(msg.Data), pipe.dc.Label())

		copy(p, msg.Data)
		return len(msg.Data), nil
	}
}

func (pipe *Pipe) Write(p []byte) (n int, err error) {
	log.Println("Pipe.Write", (p), len(p), pipe.dc.Label())

	if err := pipe.dc.Send(p); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (pipe *Pipe) Close() error {
	pipe.done()
	return nil
}
