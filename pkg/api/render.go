package api

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/grafana/grafana-image-renderer/pkg/chromium"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// This also implicitly gives us a count for each result type, so we can calculate success rate.
	MetricRenderDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:        "http_render_browser_request_duration",
		Help:        "How long does a single render take, limited to the browser?",
		ConstLabels: prometheus.Labels{"unit": "seconds"},
		Buckets:     []float64{0.5, 1, 3, 4, 5, 7, 9, 10, 11, 15, 19, 20, 21, 24, 27, 29, 30, 31, 35, 55, 95, 125, 305, 605},
	}, []string{"result"})
)

func HandlePostRender(browser *chromium.Browser) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		url := r.URL.Query().Get("url")
		if url == "" {
			http.Error(w, "Missing 'url' query parameter", http.StatusBadRequest)
			return
		}

		start := time.Now()
		body, err := browser.RenderPDF(r.Context(), url)
		if err != nil {
			MetricRenderDuration.WithLabelValues("error").Observe(time.Since(start).Seconds())
			slog.ErrorContext(r.Context(), "failed to render PDF", "error", err)
			http.Error(w, "Failed to render PDF", http.StatusInternalServerError)
			return
		}
		MetricRenderDuration.WithLabelValues("success").Observe(time.Since(start).Seconds())

		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(body)))
		w.WriteHeader(http.StatusOK)
		// TODO(perf): we could stream the bytes through from the browser to the response...
		_, _ = w.Write(body)
	})
}
