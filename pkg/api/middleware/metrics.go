package middleware

import (
	"net/http"
	"strconv"
	"strings"
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
	}, []string{"method", "path", "status_code", "encoding"})
	MetricResponseSize = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "http_response_size",
		Help: "How large is the rendered response (e.g. PNG/PDF/CSV) returned to the client?",
		ConstLabels: prometheus.Labels{
			"unit": "bytes",
		},
		//   2.0KB   3.1KB    4.8KB    7.4KB  11.5KB 17.9KB  27.7KB  42.8KB  66.4KB  102.8KB
		// 159.3KB 246.8KB  382.4KB  592.4KB 917.8KB 1.39MB  2.15MB  3.33MB  5.16MB  8.00MB
		Buckets: prometheus.ExponentialBucketsRange(2*1024, 8*1024*1024, 20),
	}, []string{"method", "path", "status_code", "encoding"})
)

// RequestMetrics adds some Prometheus metrics to the HTTP handler, to ensure we know what's going on with it.
func RequestMetrics(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tracer := tracer(r.Context())
		ctx, span := tracer.Start(r.Context(), "RequestMetrics")
		defer span.End()
		r = r.WithContext(ctx)

		encoding := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("encoding")))
		if encoding != "png" && encoding != "pdf" {
			encoding = "unknown"
		}

		now := time.Now()
		recorder := &statusRecordingResponseWriter{rw: w}
		h.ServeHTTP(recorder, r)
		statusCode := strconv.Itoa(recorder.status)
		MetricRequestDurations.WithLabelValues(r.Method, r.Pattern, statusCode, encoding).Observe(time.Since(now).Seconds())
		MetricResponseSize.WithLabelValues(r.Method, r.Pattern, statusCode, encoding).Observe(float64(recorder.bytesWritten))
	})
}

func InFlightMetrics(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tracer := tracer(r.Context())
		ctx, span := tracer.Start(r.Context(), "InFlightMetrics")
		defer span.End()
		r = r.WithContext(ctx)

		MetricRequestsInFlight.Inc()
		defer MetricRequestsInFlight.Dec()
		h.ServeHTTP(w, r)
	})
}

var (
	_ http.ResponseWriter = (*statusRecordingResponseWriter)(nil)
	_ http.Flusher        = (*statusRecordingResponseWriter)(nil)
)

type statusRecordingResponseWriter struct {
	rw           http.ResponseWriter
	once         sync.Once
	status       int
	bytesWritten int
}

func (s *statusRecordingResponseWriter) Header() http.Header {
	return s.rw.Header()
}

func (s *statusRecordingResponseWriter) Write(b []byte) (int, error) {
	s.once.Do(func() {
		s.status = http.StatusOK
	})
	n, err := s.rw.Write(b)
	s.bytesWritten += n
	return n, err
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
