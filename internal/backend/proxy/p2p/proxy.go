package p2p

import (
	"context"
	"io"
)

type ProxyReaderFunc func(ctx context.Context, rw io.ReadWriteCloser) error
type ProxyWriterFunc func(msg []byte) error

type Redirector interface {
	Run(ctx context.Context, rw io.ReadWriteCloser) error

	io.Writer
	io.Closer
}
