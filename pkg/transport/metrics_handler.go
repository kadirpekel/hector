package transport

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func MetricsHandler() http.Handler {
	return promhttp.Handler()
}
