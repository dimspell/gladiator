package console

import (
	"fmt"
	"log/slog"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/cenkalti/backoff/v4"
)

const (
	StatusNotRunning int32 = iota
	StatusStarting
	StatusFailingToStart
	StatusRunning
	StatusClosing
)

type Probe struct {
	Health       atomic.Int32
	SignalChange chan int32

	cancel chan struct{}
}

func NewProbe() *Probe {
	return &Probe{
		SignalChange: make(chan int32),
	}
}

func (p *Probe) Signal(signalCode int32) {
	p.Health.Store(signalCode)
	p.SignalChange <- signalCode
}

func (p *Probe) StartupProbe(operation func() error) {
	p.cancel = make(chan struct{})
	p.Signal(StatusStarting)

	go func() {
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
					p.Signal(StatusNotRunning)
					return
				}
				continue
			}
		}
	}()
}

func (p *Probe) Stop() {
	p.cancel <- struct{}{}
	close(p.cancel)
}

func (p *Probe) OnChange(handle func(code int32, isRunning bool), closer <-chan struct{}) {
	go func() {
		for {
			select {
			case code := <-p.SignalChange:
				handle(code, code == StatusRunning)
			case <-closer:
				return
			}
		}
	}()
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

func HealthCheckProbe(httpURL string) func() error {
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
