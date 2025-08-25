package api

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
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
	createRenderingOptions := func(r *http.Request) (chromium.RenderingOptions, error) {
		renderingOptions := browser.RenderingOptionsPrototype // create a copy

		renderingOptions.URL = r.URL.Query().Get("url")
		if renderingOptions.URL == "" {
			return renderingOptions, errors.New("missing 'url' query parameter")
		}

		var err error
		width := r.URL.Query().Get("width")
		if width != "" {
			if renderingOptions.Width, err = strconv.Atoi(width); err != nil {
				return renderingOptions, fmt.Errorf("invalid 'width' query parameter: %w", err)
			}
		}

		height := r.URL.Query().Get("height")
		if height != "" {
			if renderingOptions.Height, err = strconv.Atoi(height); err != nil {
				return renderingOptions, fmt.Errorf("invalid 'height' query parameter: %w", err)
			}
		}

		timeout := r.URL.Query().Get("timeout")
		if timeout != "" {
			if renderingOptions.Timeout, err = time.ParseDuration(timeout); err != nil {
				return renderingOptions, fmt.Errorf("invalid 'timeout' query parameter: %w", err)
			}
		}

		renderingOptions.Landscape = r.URL.Query().Get("landscape") == "true"

		timeZone := r.URL.Query().Get("timezone")
		if timeZone != "" {
			renderingOptions.TimeZone = timeZone
		}

		paper := r.URL.Query().Get("paper")
		if paper != "" {
			if err := renderingOptions.PaperSize.UnmarshalText([]byte(paper)); err != nil {
				return renderingOptions, fmt.Errorf("invalid 'paper' query parameter: %w", err)
			}
		}

		return renderingOptions, nil
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		renderingOptions, err := createRenderingOptions(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		start := time.Now()
		body, err := browser.RenderPDF(r.Context(), renderingOptions)
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
