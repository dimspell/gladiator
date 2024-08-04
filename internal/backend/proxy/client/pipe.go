package client

import (
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"

	"github.com/pion/webrtc/v4"
)

type Peer struct {
	ID   string
	Name string

	IP string

	Host   *HostListener
	Guest  *GuestProxy
	Proxer Proxer

	Connection *webrtc.PeerConnection
}

var _ io.ReadWriteCloser = (*Pipe)(nil)

type Pipe struct {
	dc     *webrtc.DataChannel
	done   chan struct{}
	dcData chan webrtc.DataChannelMessage
}

func NewPipe(dc *webrtc.DataChannel, room string, proxy Proxer) *Pipe {
	pipe := &Pipe{
		dc:     dc,
		done:   make(chan struct{}, 1),
		dcData: make(chan webrtc.DataChannelMessage, 1),
	}

	slog.Debug("Registered DataChannel.onMessage handler", "label", dc.Label())

	ctx, cancel := context.WithCancel(context.TODO())

	switch dc.Label() {
	case fmt.Sprintf("%s/tcp", room):
		go func() {
			if err := proxy.RunUDP(ctx, pipe); err != nil {
				panic(err)
			}
		}()
	case fmt.Sprintf("%s/udp", room):
		go func() {
			if err := proxy.RunUDP(ctx, pipe); err != nil {
				panic(err)
			}
		}()
	}

	dc.OnClose(func() {
		pipe.Close()
		cancel()
	})

	go func() {
		for {
			select {
			case <-pipe.done:
				return
			case msg := <-pipe.dcData:
				if pipe.dcData == nil {
					return
				}
				log.Println("channel", pipe.dc.Label(), "msg", msg.Data)
				// guest.connUDP.Write(msg.Data)
				// log.Println("Pipe.OnMessage", msg.Data, pipe.dc.Label())
				// case guest.
			}
		}
	}()

	dc.OnMessage(func(msg webrtc.DataChannelMessage) {
		log.Println("Pipe.OnMessage", msg.Data, pipe.dc.Label())

		if pipe.dcData == nil {
			return
		}
		pipe.dcData <- msg
	})

	dc.OnError(func(err error) {
		slog.Warn("Data channel error", "error", err)
	})

	return pipe
}

// Read from the DC channel
func (pipe *Pipe) Read(p []byte) (n int, err error) {
	if pipe.dcData == nil {
		return 0, io.EOF
	}

	// TODO: Handle timeout
	select {
	case msg := <-pipe.dcData:
		if len(msg.Data) == 0 {
			log.Println("Pipe.Read", (msg.Data), pipe.dc.Label())
			return 0, io.EOF
		}

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
