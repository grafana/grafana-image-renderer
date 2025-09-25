package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/grafana/grafana-image-renderer/pkg/service"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// This also implicitly gives us a count for each result type, so we can calculate success rate.
	MetricRenderCSVDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:        "http_render_csv_request_duration",
		Help:        "How long does a single CSV render take?",
		ConstLabels: prometheus.Labels{"unit": "seconds"},
		Buckets:     []float64{0.5, 1, 3, 4, 5, 7, 9, 10, 11, 15, 19, 20, 21, 24, 27, 29, 30, 31, 35, 55, 95, 125, 305, 605},
	}, []string{"result"})
)

func HandlePostRenderCSV(browser *service.BrowserService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		url := r.URL.Query().Get("url")
		if url == "" {
			http.Error(w, "missing 'url' query parameter", http.StatusBadRequest)
			return
		}
		if encoding := r.URL.Query().Get("encoding"); encoding != "" && encoding != "csv" {
			http.Error(w, "invalid 'encoding' query parameter: must be 'csv' or empty/missing", http.StatusBadRequest)
			return
		}
		ctx := r.Context()
		if timeout := r.URL.Query().Get("timeout"); timeout != "" {
			if regexpOnlyNumbers.MatchString(timeout) {
				seconds, err := strconv.Atoi(timeout)
				if err != nil {
					http.Error(w, fmt.Sprintf("invalid 'timeout' query parameter: %v", err), http.StatusBadRequest)
					return
				}
				timeoutCtx, cancelTimeout := context.WithTimeout(r.Context(), time.Duration(seconds)*time.Second)
				defer cancelTimeout()
				ctx = timeoutCtx
			} else {
				timeout, err := time.ParseDuration(timeout)
				if err != nil {
					http.Error(w, fmt.Sprintf("invalid 'timeout' query parameter: %v", err), http.StatusBadRequest)
					return
				}
				timeoutCtx, cancelTimeout := context.WithTimeout(r.Context(), timeout)
				defer cancelTimeout()
				ctx = timeoutCtx
			}
		}
		renderKey := r.URL.Query().Get("renderKey")
		domain := r.URL.Query().Get("domain")
		acceptLanguage := r.Header.Get("Accept-Language") // if empty, we just don't set it

		start := time.Now()
		contents, err := browser.RenderCSV(ctx, url, renderKey, domain, acceptLanguage)
		if err != nil {
			MetricRenderCSVDuration.WithLabelValues("error").Observe(time.Since(start).Seconds())
			http.Error(w, "CSV rendering failed", http.StatusInternalServerError)
			slog.ErrorContext(ctx, "failed to render CSV", "err", err)
			return
		}
		MetricRenderCSVDuration.WithLabelValues("success").Observe(time.Since(start).Seconds())

		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", `attachment; filename="data.csv"`)
		_, _ = w.Write(contents)
	})
}
