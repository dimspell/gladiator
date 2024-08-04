package client

import (
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"

	"github.com/pion/webrtc/v4"
	"golang.org/x/sync/errgroup"
)

type Peer struct {
	ID   string
	Name string
	IP   net.IP

	Proxer     Proxer
	Connection *webrtc.PeerConnection
}

var _ io.ReadWriteCloser = (*Pipe)(nil)

type Pipe struct {
	dc     *webrtc.DataChannel
	done   chan struct{}
	dcData chan webrtc.DataChannelMessage
}

func NewPipe(parentCtx context.Context, dc *webrtc.DataChannel, room string, proxy Proxer) *Pipe {
	pipe := &Pipe{
		dc:     dc,
		done:   make(chan struct{}, 1),
		dcData: make(chan webrtc.DataChannelMessage, 1),
	}

	slog.Debug("Registered DataChannel.onMessage handler", "label", dc.Label())

	isTCP := dc.Label() == fmt.Sprintf("%s/tcp", room)
	isUDP := dc.Label() == fmt.Sprintf("%s/udp", room)

	ctx, cancel := context.WithCancel(context.TODO())
	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		if isTCP {
			return proxy.RunTCP(ctx, pipe)
		}
		if isUDP {
			return proxy.RunUDP(ctx, pipe)
		}
		return nil
	})
	g.Go(func() error {
		for {
			select {
			case <-ctx.Done():
				slog.Debug("context done", "error", ctx.Err())
				return ctx.Err()
			case <-pipe.done:
				return nil
			case msg := <-pipe.dcData:
				slog.Debug("Pipe.OnMessage.select", "data", msg.Data, "channel", pipe.dc.Label())

				if pipe.dcData == nil {
					return nil
				}

				if isUDP {
					if err := proxy.WriteUDPMessage(msg.Data); err != nil {
						slog.Warn("Failed to send data to peer", "error", err)
					}
					continue
				}
				if isTCP {
					if err := proxy.WriteTCPMessage(msg.Data); err != nil {
						slog.Warn("Failed to send data to peer", "error", err)
					}
					continue
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
			log.Println("Pipe.Read", (msg.Data), pipe.dc.Label())
			return 0, io.EOF
		}
		log.Println("Pipe.Read", (msg.Data), len(msg.Data), pipe.dc.Label())

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
	if pipe.dcData != nil {
		pipe.done <- struct{}{}
		close(pipe.dcData)
		close(pipe.done)
	}
	return pipe.dc.Close()
}
