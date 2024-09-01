package proxytesthelper

import (
	"fmt"
	"log"
)

type FakeWebsocket struct {
	Buffer [][]byte
	i      int
}

func (f *FakeWebsocket) Close() error {
	// TODO implement me
	panic("implement me")
}

func (f *FakeWebsocket) Read(p []byte) (n int, err error) {
	log.Println("FakeWebsocket.Read", f.i)
	f.i++
	if f.i > len(f.Buffer) {
		return 0, fmt.Errorf("no more messages")
	}
	return copy(p, f.Buffer[f.i-1]), nil
}

func (f *FakeWebsocket) Write(p []byte) (n int, err error) {
	log.Println("FakeWebsocket.Write", len(f.Buffer))
	f.Buffer = append(f.Buffer, p)
	return len(p), nil
}
