package redirect

import (
	"context"
	"time"
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

func (r *Noop) Run(_ context.Context) error {
	return nil
}

func (r *Noop) Alive(_ time.Time, _ time.Duration) bool {
	return true
}
