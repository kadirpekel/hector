package transport

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MetricsHandler returns the HTTP handler for Prometheus metrics endpoint
// This handler serves metrics in the Prometheus text format at /metrics
//
// Usage:
//
//	mux := http.NewServeMux()
//	mux.Handle("/metrics", MetricsHandler())
//	http.ListenAndServe(":8080", mux)
//
// The endpoint will be available at: http://localhost:8080/metrics
func MetricsHandler() http.Handler {
	return promhttp.Handler()
}
