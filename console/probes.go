package console

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func (c *Console) StartupProbe(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK) // or 503
}

func (c *Console) ReadinessProbe(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (c *Console) LivenessProbe(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (c *Console) Metrics() http.Handler {
	return promhttp.Handler()
}
