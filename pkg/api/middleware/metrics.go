package middleware

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	MetricRequestsInFlight = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "http_requests_in_flight",
		Help: "How many requests are in flight?",
	})
	MetricRequestDurations = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "http_request_duration",
		Help: "How long does a request take?",
		ConstLabels: prometheus.Labels{
			"unit": "seconds",
		},
		Buckets: []float64{0.5, 1, 3, 4, 5, 7, 9, 10, 11, 15, 19, 20, 21, 24, 27, 29, 30, 31, 35, 55, 95, 125, 305, 605},
	}, []string{"method", "path"}) // TODO: Output HTTP status codes, too
)

// RequestMetrics adds some Prometheus metrics to the HTTP handler, to ensure we know what's going on with it.
func RequestMetrics(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		now := time.Now()
		MetricRequestsInFlight.Inc()
		h.ServeHTTP(w, r)
		MetricRequestsInFlight.Dec()
		MetricRequestDurations.WithLabelValues(r.Method, r.Pattern).Observe(time.Since(now).Seconds())
	})
}
