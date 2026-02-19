package api

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"mime"
	"net/http"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/grafana/grafana-image-renderer/pkg/service"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
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

func HandleGetRenderCSV(browser *service.BrowserService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tracer := tracer(r.Context())
		ctx, span := tracer.Start(r.Context(), "HandleGetRenderCSV")
		defer span.End()
		r = r.WithContext(ctx)

		url := r.URL.Query().Get("url")
		if url == "" {
			span.SetStatus(codes.Error, "url query param empty")
			http.Error(w, "missing 'url' query parameter", http.StatusBadRequest)
			return
		}
		span.SetAttributes(attribute.String("url", url))

		if encoding := r.URL.Query().Get("encoding"); encoding != "" && !strings.EqualFold(encoding, "csv") {
			span.SetStatus(codes.Error, "invalid encoding query param")
			span.SetAttributes(attribute.String("encoding", encoding))
			http.Error(w, "invalid 'encoding' query parameter: must be 'csv' or empty/missing", http.StatusBadRequest)
			return
		}

		if timeout := r.URL.Query().Get("timeout"); timeout != "" {
			dur, err := parseTimeout(timeout)
			if err != nil {
				span.SetStatus(codes.Error, "invalid timeout query param")
				span.RecordError(err, trace.WithAttributes(attribute.String("timeout", timeout)))
				http.Error(w, fmt.Sprintf("invalid 'timeout' query parameter: %v", err), http.StatusBadRequest)
				return
			}
			if dur > 0 {
				timeoutCtx, cancelTimeout := context.WithTimeout(ctx, dur)
				defer cancelTimeout()
				ctx = timeoutCtx
			}
			span.SetAttributes(attribute.String("timeout", dur.String()))
		}
		renderKey := r.URL.Query().Get("renderKey")
		domain := r.URL.Query().Get("domain")
		acceptLanguage := r.Header.Get("Accept-Language") // if empty, we just don't set it
		span.SetAttributes(
			attribute.String("acceptLanguage", acceptLanguage),
			attribute.String("renderKeyDomain", domain))

		start := time.Now()
		contents, fileName, err := browser.RenderCSV(ctx, url, renderKey, domain, acceptLanguage)
		if err != nil {
			MetricRenderCSVDuration.WithLabelValues("error").Observe(time.Since(start).Seconds())
			span.SetStatus(codes.Error, "csv rendering failed")
			span.RecordError(err)
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
				http.Error(w, "request timed out", http.StatusRequestTimeout)
			} else if errors.Is(err, service.ErrInvalidBrowserOption) {
				http.Error(w, fmt.Sprintf("invalid request: %v", err), http.StatusBadRequest)
			} else {
				http.Error(w, "CSV rendering failed", http.StatusInternalServerError)
				slog.ErrorContext(ctx, "failed to render CSV", "err", err)
			}
			return
		}
		MetricRenderCSVDuration.WithLabelValues("success").Observe(time.Since(start).Seconds())
		span.SetStatus(codes.Ok, "csv rendered successfully")

		requestedFilePath := r.URL.Query().Get("filePath")
		requestedFilePath = filepath.ToSlash(requestedFilePath)
		requestedFilePath = strings.TrimPrefix(requestedFilePath, "/")
		requestedFilePath = path.Base(requestedFilePath)
		if requestedFilePath == "." || requestedFilePath == "/" || requestedFilePath == "" {
			requestedFilePath = fileName
		}
		if requestedFilePath == "" {
			requestedFilePath = "data.csv"
		}
		if !strings.HasSuffix(strings.ToLower(requestedFilePath), ".csv") {
			requestedFilePath += ".csv"
		}

		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{
			"filename": requestedFilePath,
		}))
		_, _ = w.Write(contents)
	})
}
