package api

import (
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"time"
	_ "time/tzdata" // fallback where we have no tzdata on the distro; used in LoadLocation

	"github.com/grafana/grafana-image-renderer/pkg/service"
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

	regexpOnlyNumbers = regexp.MustCompile(`^[0-9]+$`)
)

func HandlePostRender(browser *service.BrowserService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		url := r.URL.Query().Get("url")
		if url == "" {
			http.Error(w, "missing 'url' query parameter", http.StatusBadRequest)
			return
		}
		var options []service.RenderingOption

		width, height := -1, -1
		if widthStr := r.URL.Query().Get("width"); widthStr != "" {
			var err error
			width, err = strconv.Atoi(widthStr)
			if err != nil {
				http.Error(w, fmt.Sprintf("invalid 'width' query parameter: %v", err), http.StatusBadRequest)
				return
			}
		}
		if heightStr := r.URL.Query().Get("height"); heightStr != "" {
			var err error
			height, err = strconv.Atoi(heightStr)
			if err != nil {
				http.Error(w, fmt.Sprintf("invalid 'height' query parameter: %v", err), http.StatusBadRequest)
				return
			}
		}
		options = append(options, service.WithViewport(width, height))
		if timeout := r.URL.Query().Get("timeout"); timeout != "" {
			if regexpOnlyNumbers.MatchString(timeout) {
				seconds, err := strconv.Atoi(timeout)
				if err != nil {
					http.Error(w, fmt.Sprintf("invalid 'timeout' query parameter: %v", err), http.StatusBadRequest)
					return
				}
				options = append(options, service.WithTimeout(time.Duration(seconds)*time.Second))
			} else {
				timeout, err := time.ParseDuration(timeout)
				if err != nil {
					http.Error(w, fmt.Sprintf("invalid 'timeout' query parameter: %v", err), http.StatusBadRequest)
					return
				}
				options = append(options, service.WithTimeout(timeout))
			}
		}
		if scaleFactor := r.URL.Query().Get("deviceScaleFactor"); scaleFactor != "" {
			scaleFactor, err := strconv.ParseFloat(scaleFactor, 64)
			if err != nil {
				http.Error(w, fmt.Sprintf("invalid 'deviceScaleFactor' query parameter: %v", err), http.StatusBadRequest)
				return
			}
			options = append(options, service.WithPageScaleFactor(scaleFactor))
		}
		if timeZone := r.URL.Query().Get("timeZone"); timeZone != "" {
			timeZone, err := time.LoadLocation(timeZone)
			if err != nil {
				http.Error(w, fmt.Sprintf("invalid 'timeZone' query parameter: %v", err), http.StatusBadRequest)
				return
			}
			options = append(options, service.WithTimeZone(timeZone))
		}
		if landscape := r.URL.Query().Get("landscape"); landscape != "" {
			options = append(options, service.WithLandscape(landscape == "true"))
		}
		renderKey := r.URL.Query().Get("renderKey")
		domain := r.URL.Query().Get("domain")
		if renderKey != "" && domain != "" {
			options = append(options, service.WithCookie("renderKey", renderKey, domain))
		}
		encoding := r.URL.Query().Get("encoding")
		switch encoding {
		case "", "pdf":
			options = append(options, service.WithPDFPrinter())
		case "png":
			var printerOpts []service.PNGPrinterOption
			if height == -1 {
				printerOpts = append(printerOpts, service.WithFullHeight(true))
			}
			options = append(options, service.WithPNGPrinter(printerOpts...))
		default:
			http.Error(w, fmt.Sprintf("invalid 'encoding' query parameter: %q", encoding), http.StatusBadRequest)
			return
		}

		start := time.Now()
		body, contentType, err := browser.Render(r.Context(), url, options...)
		if err != nil {
			MetricRenderDuration.WithLabelValues("error").Observe(time.Since(start).Seconds())
			slog.ErrorContext(r.Context(), "failed to render", "error", err)
			http.Error(w, "Failed to render", http.StatusInternalServerError)
			return
		}
		MetricRenderDuration.WithLabelValues("success").Observe(time.Since(start).Seconds())

		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(body)))
		w.WriteHeader(http.StatusOK)
		// TODO(perf): we could stream the bytes through from the browser to the response...
		_, _ = w.Write(body)
	})
}
