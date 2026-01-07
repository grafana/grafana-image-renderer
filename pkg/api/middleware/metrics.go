package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	MetricRequestsInFlight = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "http_requests_in_flight",
		Help: "How many expensive requests are in flight?",
	})
	MetricRequestDurations = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "http_request_duration",
		Help: "How long does a request take?",
		ConstLabels: prometheus.Labels{
			"unit": "seconds",
		},
		Buckets: []float64{
			0.1, 0.5, 1, 3, 4, 5, 7, 9,
			10, 11, 15, 19, 20, 21, 24,
			27, 29, 30, 31, 35, 55, 95,
			125, 305, 605, 905, 1205,
			1800, 3600,
		},
	}, []string{"method", "path", "status_code"})
)

// RequestMetrics adds some Prometheus metrics to the HTTP handler, to ensure we know what's going on with it.
func RequestMetrics(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tracer := tracer(r.Context())
		ctx, span := tracer.Start(r.Context(), "RequestMetrics")
		defer span.End()
		r = r.WithContext(ctx)

		now := time.Now()
		recorder := &statusRecordingResponseWriter{rw: w}
		h.ServeHTTP(recorder, r)
		MetricRequestDurations.WithLabelValues(r.Method, r.Pattern, strconv.Itoa(recorder.status)).Observe(time.Since(now).Seconds())
	})
}

func InFlightMetrics(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tracer := tracer(r.Context())
		ctx, span := tracer.Start(r.Context(), "InFlightMetrics")
		defer span.End()
		r = r.WithContext(ctx)

		MetricRequestsInFlight.Inc()
		defer MetricRequestsInFlight.Dec() // defer to run despite panics
		h.ServeHTTP(w, r)
	})
}

var (
	_ http.ResponseWriter = (*statusRecordingResponseWriter)(nil)
	_ http.Flusher        = (*statusRecordingResponseWriter)(nil)
)

type statusRecordingResponseWriter struct {
	rw     http.ResponseWriter
	once   sync.Once
	status int
}

func (s *statusRecordingResponseWriter) Header() http.Header {
	return s.rw.Header()
}

func (s *statusRecordingResponseWriter) Write(b []byte) (int, error) {
	s.once.Do(func() {
		s.status = http.StatusOK
	})
	return s.rw.Write(b)
}

func (s *statusRecordingResponseWriter) WriteHeader(statusCode int) {
	s.once.Do(func() {
		s.status = statusCode
	})
	s.rw.WriteHeader(statusCode)
}

func (s *statusRecordingResponseWriter) Flush() {
	if flusher, ok := s.rw.(http.Flusher); ok {
		flusher.Flush()
	}
}
