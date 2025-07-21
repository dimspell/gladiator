package console

import (
	"bytes"
	"context"
	"net"

	"github.com/quic-go/quic-go"
)

type MockStream struct {
	Writer bytes.Buffer
	Reader bytes.Buffer
}

func (ms *MockStream) CancelWrite(code quic.StreamErrorCode) {}
func (ms *MockStream) CancelRead(code quic.StreamErrorCode)  {}

func (ms *MockStream) Write(p []byte) (int, error) {
	return ms.Writer.Write(p)
}
func (ms *MockStream) Read(p []byte) (int, error) {
	return ms.Reader.Read(p)
}

type MockConn struct{}

func (mc *MockConn) AcceptStream(ctx context.Context) (*quic.Stream, error) {
	// TODO implement me
	panic("implement me")
}

func (mc *MockConn) RemoteAddr() net.Addr {
	return &net.UDPAddr{}
}

func (mc *MockConn) CloseWithError(code quic.ApplicationErrorCode, msg string) error {
	return nil
}
