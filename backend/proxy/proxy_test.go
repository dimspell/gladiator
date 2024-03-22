package proxy

import (
	"bytes"
	"io"
	"net"
	"time"
)

var _ net.Conn = (*fakeConn)(nil)

type fakeConn struct {
	BufWrite *bytes.Buffer
	BufRead  *bytes.Buffer
}

func (f *fakeConn) Read(b []byte) (n int, err error) {
	data, err := f.BufRead.ReadBytes('|')
	n = len(data) - 1
	if n < 0 {
		return 0, io.EOF
	}
	copy(b, data[:n])
	return n, err
}

func (f *fakeConn) Write(b []byte) (n int, err error) {
	n, err = f.BufWrite.Write(b)
	f.BufWrite.WriteByte('|')
	return
}

func (f *fakeConn) Close() error {
	return nil
}

func (f *fakeConn) LocalAddr() net.Addr {
	// TODO implement me
	panic("implement me")
}

func (f *fakeConn) RemoteAddr() net.Addr {
	// TODO implement me
	panic("implement me")
}

func (f *fakeConn) SetDeadline(t time.Time) error {
	// TODO implement me
	panic("implement me")
}

func (f *fakeConn) SetReadDeadline(t time.Time) error {
	// TODO implement me
	panic("implement me")
}

func (f *fakeConn) SetWriteDeadline(t time.Time) error {
	// TODO implement me
	panic("implement me")
}
