package redirect

import (
	"context"
	"io"
)

var _ Redirect = (*Noop)(nil)

type Noop struct{}

func (r *Noop) Write(_ []byte) (n int, err error) {
	return 0, nil
}

func (r *Noop) Close() error {
	return nil
}

func (r *Noop) Run(_ context.Context, _ io.ReadWriteCloser) error {
	return nil
}
