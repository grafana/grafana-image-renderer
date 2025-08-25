package api

import (
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"regexp"
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

	regexpOnlyNumbers = regexp.MustCompile(`^[0-9]+$`)
)

func HandlePostRender(browser *chromium.Browser) http.Handler {
	createRenderingOptions := func(r *http.Request) (chromium.RenderingOptions, int, error) {
		renderingOptions := browser.RenderingOptionsPrototype // create a copy

		renderingOptions.URL = r.URL.Query().Get("url")
		if renderingOptions.URL == "" {
			return renderingOptions, http.StatusBadRequest, errors.New("missing 'url' query parameter")
		}

		var err error
		width := r.URL.Query().Get("width")
		if width != "" {
			if renderingOptions.Width, err = strconv.Atoi(width); err != nil {
				return renderingOptions, http.StatusBadRequest, fmt.Errorf("invalid 'width' query parameter: %w", err)
			}
			if renderingOptions.Width < 10 {
				renderingOptions.Width = browser.RenderingOptionsPrototype.Width
			}
		}

		height := r.URL.Query().Get("height")
		if height != "" {
			if renderingOptions.Height, err = strconv.Atoi(height); err != nil {
				return renderingOptions, http.StatusBadRequest, fmt.Errorf("invalid 'height' query parameter: %w", err)
			}
			if renderingOptions.Height == -1 {
				renderingOptions.FullHeight = true
				renderingOptions.Height = int(math.Floor(float64(renderingOptions.Width) * 0.75))
			}
		}

		timeout := r.URL.Query().Get("timeout")
		if timeout != "" {
			if regexpOnlyNumbers.MatchString(timeout) {
				seconds, err := strconv.Atoi(timeout)
				if err != nil {
					return renderingOptions, http.StatusBadRequest, fmt.Errorf("invalid 'timeout' query parameter: %w", err)
				}
				renderingOptions.Timeout = time.Duration(seconds) * time.Second
			} else if renderingOptions.Timeout, err = time.ParseDuration(timeout); err != nil {
				return renderingOptions, http.StatusBadRequest, fmt.Errorf("invalid 'timeout' query parameter: %w", err)
			}
		}

		renderingOptions.Landscape = r.URL.Query().Get("landscape") != "false"

		timeZone := r.URL.Query().Get("timezone")
		if timeZone != "" {
			renderingOptions.TimeZone = timeZone
		}

		paper := r.URL.Query().Get("paper")
		if paper != "" {
			if err := renderingOptions.PaperSize.UnmarshalText([]byte(paper)); err != nil {
				return renderingOptions, http.StatusBadRequest, fmt.Errorf("invalid 'paper' query parameter: %w", err)
			}
		}

		encoding := r.URL.Query().Get("encoding")
		switch encoding {
		case "", "pdf":
			renderingOptions.Format = chromium.RenderingFormatPDF
		case "png":
			renderingOptions.Format = chromium.RenderingFormatPNG
		default:
			return renderingOptions, http.StatusBadRequest, fmt.Errorf("invalid 'encoding' query parameter: %q", encoding)
		}

		scaleFactor := r.URL.Query().Get("deviceScaleFactor")
		if scaleFactor != "" {
			if renderingOptions.DeviceScaleFactor, err = strconv.ParseFloat(scaleFactor, 64); err != nil {
				return renderingOptions, http.StatusBadRequest, fmt.Errorf("invalid 'deviceScaleFactor' query parameter: %w", err)
			}
			if renderingOptions.DeviceScaleFactor <= 0 {
				renderingOptions.DeviceScaleFactor *= -1
				renderingOptions.Width = int(math.Floor(float64(renderingOptions.Width) * renderingOptions.DeviceScaleFactor))
				renderingOptions.Height = int(math.Floor(float64(renderingOptions.Height) * renderingOptions.DeviceScaleFactor))
				// TODO: Resize image afterwards with our own logic
			}
		}

		renderKey := r.URL.Query().Get("renderKey")
		domain := r.URL.Query().Get("domain")
		if renderKey != "" && domain != "" {
			renderingOptions.Cookies = append(renderingOptions.Cookies, chromium.Cookie{
				Name:   "renderKey",
				Value:  renderKey,
				Domain: domain,
			})
		}

		// TODO: Copy headers like in JS

		// status is only used for errors, actually
		return renderingOptions, http.StatusOK, nil
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		renderingOptions, status, err := createRenderingOptions(r)
		if err != nil {
			http.Error(w, err.Error(), status)
			return
		}

		start := time.Now()
		body, err := browser.Render(r.Context(), renderingOptions)
		if err != nil {
			MetricRenderDuration.WithLabelValues("error").Observe(time.Since(start).Seconds())
			slog.ErrorContext(r.Context(), "failed to render", "error", err)
			http.Error(w, "Failed to render", http.StatusInternalServerError)
			return
		}
		MetricRenderDuration.WithLabelValues("success").Observe(time.Since(start).Seconds())

		w.Header().Set("Content-Type", renderingOptions.Format.ContentType())
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(body)))
		w.WriteHeader(http.StatusOK)
		// TODO(perf): we could stream the bytes through from the browser to the response...
		_, _ = w.Write(body)
	})
}
