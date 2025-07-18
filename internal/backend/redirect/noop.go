package redirect

import (
	"context"
)

var _ Redirect = (*Noop)(nil)

type Noop struct{}

func NewNoop(_ Mode, _ *Addressing) (Redirect, error) {
	return &Noop{}, nil
}

func (r *Noop) Write(_ []byte) (n int, err error) {
	return 0, nil
}

func (r *Noop) Close() error {
	return nil
}

func (r *Noop) Run(_ context.Context, _ func(p []byte) (err error)) error {
	return nil
}
