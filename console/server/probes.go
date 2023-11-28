package server

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func (c *ConsoleServer) StartupProbe(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK) // or 503
}

func (c *ConsoleServer) ReadinessProbe(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (c *ConsoleServer) LivenessProbe(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (c *ConsoleServer) Metrics() http.Handler {
	return promhttp.Handler()
}
