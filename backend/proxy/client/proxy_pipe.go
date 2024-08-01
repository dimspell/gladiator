package client

import (
	"io"

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

	dc.OnMessage(func(msg webrtc.DataChannelMessage) {
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
			return 0, io.EOF
		}

		copy(p, msg.Data)
		return len(msg.Data), nil
	}
}

func (pipe *Pipe) Write(p []byte) (n int, err error) {
	if err := pipe.dc.Send(p); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (pipe *Pipe) Close() error {
	close(pipe.data)
	return pipe.dc.Close()
}
