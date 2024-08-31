package redirect

import (
	"context"
	"io"
)

type Redirect interface {
	Run(ctx context.Context, rw io.ReadWriteCloser) error

	io.Writer
	io.Closer
}
