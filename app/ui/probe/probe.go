package probe

import (
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
)

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

func (p *Probe) Get() (int32, bool) {
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

func (p *Probe) StartupProbe(operation func() error) {
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
					slog.Error("error", "err", err)
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
				"error", err)
		},
	)
}

func HttpGet(httpURL string) func() error {
	httpClient := &http.Client{Timeout: 3 * time.Second}

	return func() error {
		resp, err := httpClient.Get(httpURL)
		if err != nil {
			return err
		}
		if resp == nil {
			return fmt.Errorf("response is nil")
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("bad status: %s (%d)", resp.Status, resp.StatusCode)
		}
		return nil
	}
}
