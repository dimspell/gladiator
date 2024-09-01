package proxytesthelper

import (
	"fmt"

	"github.com/pion/webrtc/v4"
)

type FakeDataChannel struct {
	label string

	Buffer [][]byte
	i      int

	msgChan chan []byte
	closed  bool

	onClose   func()
	onMessage func(msg webrtc.DataChannelMessage)
}

func NewFakeDataChannel(label string) *FakeDataChannel {
	return &FakeDataChannel{
		label:  label,
		Buffer: [][]byte{},
	}
}

func (f *FakeDataChannel) Label() string { return f.label }

func (f *FakeDataChannel) OnError(fn func(err error)) {
	// TODO implement me
}

func (f *FakeDataChannel) OnMessage(fn func(msg webrtc.DataChannelMessage)) {
	f.onMessage = fn
}

func (f *FakeDataChannel) OnClose(fn func()) {
	f.onClose = fn
}

func (f *FakeDataChannel) Send(p []byte) error {
	f.Buffer = append(f.Buffer, p)
	return nil
}

func (f *FakeDataChannel) Close() error {
	if f.closed {
		return fmt.Errorf("already closed")
	}
	f.onClose()
	return nil
}
