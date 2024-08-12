package p2p

import (
	"context"
	"io"
)

type Redirector interface {
	Run(ctx context.Context, rw io.ReadWriteCloser) error

	io.Writer
	io.Closer
}
