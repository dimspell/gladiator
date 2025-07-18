package probe

import (
	"log/slog"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/dimspell/gladiator/internal/app/logger/logging"
)

const DefaultTimeout = 3 * time.Second

const (
	StatusNotRunning int32 = iota
	StatusStarting
	StatusFailingToStart
	StatusRunning
	StatusClosing
	StatusFailed
)

type Probe struct {
	Health       int32
	SignalChange chan int32

	mtx    sync.Mutex
	cancel chan struct{}
}

func NewProbe() *Probe {
	return &Probe{
		SignalChange: make(chan int32),
	}
}

func (p *Probe) Status() (int32, bool) {
	if p == nil {
		return StatusNotRunning, false
	}
	v := p.Health
	return v, v == StatusRunning
}

func (p *Probe) Signal(signalCode int32) {
	if p == nil {
		return
	}
	if v := p.Health; v == signalCode {
		return
	}
	p.Health = signalCode
	p.SignalChange <- signalCode
}

type StatusChecker interface {
	Check() error
}

func (p *Probe) Check(operation func() error) {
	// TODO: replace it with the context
	p.cancel = make(chan struct{})

	go func() {
		defer close(p.cancel)

		// Run the startup probe
		time.Sleep(200 * time.Millisecond)
		if err := retryUntil(operation, 10*time.Second); err != nil {
			p.Signal(StatusFailingToStart)
			return
		}
		p.Signal(StatusRunning)

		// Start the readiness & liveness probe
		ticker := backoff.NewTicker(backoff.NewConstantBackOff(5 * time.Second))
		for {
			select {
			case <-p.cancel:
				p.Signal(StatusClosing)
				return
			case <-ticker.C:
				if err := operation(); err != nil {
					slog.Error("error", logging.Error(err))
					p.Signal(StatusNotRunning)
					return
				}
				continue
			}
		}
	}()
}

func (p *Probe) StopStartupProbe() {
	if p.cancel == nil {
		return
	}
	p.cancel <- struct{}{}
	// close(p.cancel)
}

func retryUntil(operation func() error, maxElapsedTime time.Duration) error {
	exponentialBackOff := backoff.NewExponentialBackOff()
	exponentialBackOff.MaxElapsedTime = maxElapsedTime

	return backoff.RetryNotify(
		operation,
		exponentialBackOff,
		func(err error, duration time.Duration) {
			slog.Warn("Retrying operation",
				"duration", duration.String(),
				logging.Error(err))
		},
	)
}
