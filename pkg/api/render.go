package api

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
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

func HandleGetRender(browser *service.BrowserService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tracer := tracer(r.Context())
		ctx, span := tracer.Start(r.Context(), "HandleGetRender")
		defer span.End()
		r = r.WithContext(ctx)

		rawTargetURL := r.URL.Query().Get("url")
		if rawTargetURL == "" {
			http.Error(w, "missing 'url' query parameter", http.StatusBadRequest)
			return
		}
		targetURL, err := url.Parse(rawTargetURL)
		if err != nil {
			http.Error(w, fmt.Sprintf("invalid 'url' query parameter: %v", err), http.StatusBadRequest)
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
			var dur time.Duration
			if regexpOnlyNumbers.MatchString(timeout) {
				seconds, err := strconv.Atoi(timeout)
				if err != nil {
					http.Error(w, fmt.Sprintf("invalid 'timeout' query parameter: %v", err), http.StatusBadRequest)
					return
				}
				dur = time.Duration(seconds) * time.Second
			} else {
				var err error
				dur, err = time.ParseDuration(timeout)
				if err != nil {
					http.Error(w, fmt.Sprintf("invalid 'timeout' query parameter: %v", err), http.StatusBadRequest)
					return
				}
			}
			if dur > 0 {
				timeoutCtx, cancelTimeout := context.WithTimeout(ctx, dur)
				defer cancelTimeout()
				ctx = timeoutCtx
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
		var printer service.Printer
		switch encoding {
		case "", "pdf":
			var printerOpts []service.PDFPrinterOption

			paper := r.URL.Query().Get("pdf.format")
			if paper == "" {
				// FIXME: legacy support; remove in some future release.
				paper = targetURL.Query().Get("pdf.format")
			}
			if paper != "" {
				var psz service.PaperSize
				if err := psz.UnmarshalText([]byte(paper)); err != nil {
					http.Error(w, fmt.Sprintf("invalid 'pdf.format' query parameter: %v", err), http.StatusBadRequest)
					return
				}
				printerOpts = append(printerOpts, service.WithPaperSize(psz))
			}

			printBackground := r.URL.Query().Get("pdf.printBackground")
			if printBackground == "" {
				// FIXME: legacy support; remove in some future release.
				printBackground = targetURL.Query().Get("pdf.printBackground")
			}
			if printBackground != "" {
				printerOpts = append(printerOpts, service.WithPrintingBackground(printBackground == "true"))
			}

			pageRanges := r.URL.Query().Get("pdf.pageRanges")
			if pageRanges == "" {
				// FIXME: legacy support; remove in some future release.
				pageRanges = targetURL.Query().Get("pdf.pageRanges")
			}
			if pageRanges != "" {
				printerOpts = append(printerOpts, service.WithPageRanges(pageRanges))
			}

			var err error
			printer, err = service.NewPDFPrinter(printerOpts...)
			if err != nil {
				http.Error(w, fmt.Sprintf("invalid request: %v", err), http.StatusBadRequest)
				return
			}

			if pdfLandscape := r.URL.Query().Get("pdfLandscape"); pdfLandscape != "" {
				options = append(options, service.WithLandscape(pdfLandscape == "true"))
			}
		case "png":
			var printerOpts []service.PNGPrinterOption
			if height == -1 {
				printerOpts = append(printerOpts, service.WithFullHeight(true))
				options = append(options, service.WithViewport(width, 1080)) // add some height to make scrolling faster
			}

			var err error
			printer, err = service.NewPNGPrinter(printerOpts...)
			if err != nil {
				http.Error(w, fmt.Sprintf("invalid request: %v", err), http.StatusBadRequest)
				return
			}
		default:
			http.Error(w, fmt.Sprintf("invalid 'encoding' query parameter: %q", encoding), http.StatusBadRequest)
			return
		}
		if acceptLanguage := r.Header.Get("Accept-Language"); acceptLanguage != "" {
			options = append(options, service.WithHeader("Accept-Language", acceptLanguage))
		}

		start := time.Now()
		body, contentType, err := browser.Render(ctx, rawTargetURL, printer, options...)
		if err != nil {
			MetricRenderDuration.WithLabelValues("error").Observe(time.Since(start).Seconds())
			if errors.Is(err, context.DeadlineExceeded) ||
				errors.Is(err, context.Canceled) {
				http.Error(w, "Request timed out", http.StatusRequestTimeout)
				return
			} else if errors.Is(err, service.ErrInvalidBrowserOption) {
				http.Error(w, fmt.Sprintf("invalid request: %v", err), http.StatusBadRequest)
				return
			}
			slog.ErrorContext(ctx, "failed to render", "error", err)
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
