package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	startTime = time.Now()

	Uptime = prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Name: "gladiator_uptime_seconds",
			Help: "Console server uptime in seconds",
		}, func() float64 {
			return time.Since(startTime).Seconds()
		})

	ConnectionErrs = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "gladiator_websocket_connection_errors",
			Help: "Number of connection errors",
		})
)

func InitConsole() {
	prometheus.MustRegister(Uptime, ConnectionErrs)
}
