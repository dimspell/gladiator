package client

import (
	"io"
	"log"
	"log/slog"

	"github.com/pion/webrtc/v4"
)

var _ io.ReadWriteCloser = (*Pipe)(nil)

type Pipe struct {
	dc   *webrtc.DataChannel
	data chan webrtc.DataChannelMessage
}

func NewPipe(dc *webrtc.DataChannel) *Pipe {
	pipe := &Pipe{
		dc:   dc,
		data: make(chan webrtc.DataChannelMessage),
	}

	slog.Debug("Registered DataChannel.onMessage handler", "label", dc.Label())

	dc.OnMessage(func(msg webrtc.DataChannelMessage) {
		if pipe.data == nil {
			return
		}
		pipe.data <- msg
	})

	return pipe
}

func (pipe *Pipe) Read(p []byte) (n int, err error) {
	if pipe.data == nil {
		return 0, io.EOF
	}

	// TODO: Handle timeout
	select {
	case msg := <-pipe.data:
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
	if pipe.data != nil {
		close(pipe.data)
	}
	return pipe.dc.Close()
}
