package api

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata" // fallback where we have no tzdata on the distro; used in LoadLocation

	"github.com/grafana/grafana-image-renderer/pkg/config"
	"github.com/grafana/grafana-image-renderer/pkg/service"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
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

func HandleGetRender(browser *service.BrowserService, apiConfig config.APIConfig) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tracer := tracer(r.Context())
		ctx, span := tracer.Start(r.Context(), "HandleGetRender")
		defer span.End()
		r = r.WithContext(ctx)

		rawTargetURL := r.URL.Query().Get("url")
		if rawTargetURL == "" {
			span.SetStatus(codes.Error, "url query param empty")
			http.Error(w, "missing 'url' query parameter", http.StatusBadRequest)
			return
		}
		targetURL, err := url.Parse(rawTargetURL)
		if err != nil {
			span.SetStatus(codes.Error, "url query param was unparseable")
			span.RecordError(err, trace.WithAttributes(attribute.String("url", rawTargetURL)))
			http.Error(w, fmt.Sprintf("invalid 'url' query parameter: %v", err), http.StatusBadRequest)
			return
		}
		span.SetAttributes(attribute.String("url", targetURL.String()))

		var options []service.RenderingOption

		width, height := -1, -1
		if widthStr := r.URL.Query().Get("width"); widthStr != "" {
			var err error
			width, err = strconv.Atoi(widthStr)
			if err != nil {
				span.SetStatus(codes.Error, "invalid width query param")
				span.RecordError(err, trace.WithAttributes(attribute.String("width", widthStr)))
				http.Error(w, fmt.Sprintf("invalid 'width' query parameter: %v", err), http.StatusBadRequest)
				return
			}
			span.SetAttributes(attribute.Int("width", width))
		}
		if heightStr := r.URL.Query().Get("height"); heightStr != "" {
			var err error
			height, err = strconv.Atoi(heightStr)
			if err != nil {
				span.SetStatus(codes.Error, "invalid height query param")
				span.RecordError(err, trace.WithAttributes(attribute.String("height", heightStr)))
				http.Error(w, fmt.Sprintf("invalid 'height' query parameter: %v", err), http.StatusBadRequest)
				return
			}
			span.SetAttributes(attribute.Int("height", height))
		}
		options = append(options, service.WithViewport(width, height))
		if timeout := r.URL.Query().Get("timeout"); timeout != "" {
			dur, err := parseTimeout(timeout)
			if err != nil {
				span.SetStatus(codes.Error, "invalid timeout query param")
				span.RecordError(err, trace.WithAttributes(attribute.String("timeout", timeout)))
				http.Error(w, fmt.Sprintf("invalid 'timeout' query parameter: %v", err), http.StatusBadRequest)
				return
			}
			if dur > 0 {
				span.SetAttributes(attribute.String("timeout", dur.String()))
				timeoutCtx, cancelTimeout := context.WithTimeout(ctx, dur)
				defer cancelTimeout()
				ctx = timeoutCtx
			}
		}
		if scaleFactor := r.URL.Query().Get("deviceScaleFactor"); scaleFactor != "" {
			pageScaleFactor, err := strconv.ParseFloat(scaleFactor, 64)
			if err != nil {
				span.SetStatus(codes.Error, "invalid deviceScaleFactor query param")
				span.RecordError(err, trace.WithAttributes(attribute.String("scaleFactor", scaleFactor)))
				http.Error(w, fmt.Sprintf("invalid 'deviceScaleFactor' query parameter: %v", err), http.StatusBadRequest)
				return
			}
			options = append(options, service.WithPageScaleFactor(pageScaleFactor))
			span.SetAttributes(attribute.Float64("deviceScaleFactor", pageScaleFactor))
		}
		if timeZone := r.URL.Query().Get("timeZone"); timeZone != "" {
			timeLocation, err := time.LoadLocation(timeZone)
			if err != nil {
				span.SetStatus(codes.Error, "invalid timeZone query param")
				span.RecordError(err, trace.WithAttributes(attribute.String("timeZone", timeZone)))
				http.Error(w, fmt.Sprintf("invalid 'timeZone' query parameter: %v", err), http.StatusBadRequest)
				return
			}
			options = append(options, service.WithTimeZone(timeLocation))
			span.SetAttributes(attribute.String("timeZone", timeZone))
		}
		if landscape := r.URL.Query().Get("landscape"); landscape != "" {
			options = append(options, service.WithLandscape(landscape == "true"))
			span.SetAttributes(attribute.Bool("landscape", landscape == "true"))
		}
		renderKey := r.URL.Query().Get("renderKey")
		domain := r.URL.Query().Get("domain")
		if renderKey != "" && domain != "" {
			options = append(options, service.WithCookie("renderKey", renderKey, domain))
			span.AddEvent("added renderKey cookie", trace.WithAttributes(attribute.String("domain", domain)))
		}
		encoding := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("encoding")))
		if encoding == "" {
			encoding = string(apiConfig.DefaultEncoding)
		}
		var printer service.Printer
		switch encoding {
		case "pdf":
			var printerOpts []service.PDFPrinterOption

			paper := r.URL.Query().Get("pdf.format")
			if paper == "" {
				// FIXME: legacy support; remove in some future release.
				paper = targetURL.Query().Get("pdf.format")
			}
			if paper != "" {
				var psz service.PaperSize
				if err := psz.UnmarshalText([]byte(paper)); err != nil {
					span.SetStatus(codes.Error, "invalid pdf.format query param")
					span.RecordError(err, trace.WithAttributes(attribute.String("pdf.format", paper)))
					http.Error(w, fmt.Sprintf("invalid 'pdf.format' query parameter: %v", err), http.StatusBadRequest)
					return
				}
				printerOpts = append(printerOpts, service.WithPaperSize(psz))
				span.SetAttributes(attribute.String("pdf.format", paper))
			}

			printBackground := r.URL.Query().Get("pdf.printBackground")
			if printBackground == "" {
				// FIXME: legacy support; remove in some future release.
				printBackground = targetURL.Query().Get("pdf.printBackground")
			}
			if printBackground != "" {
				printerOpts = append(printerOpts, service.WithPrintingBackground(printBackground == "true"))
				span.SetAttributes(attribute.Bool("pdf.printBackground", printBackground == "true"))
			}

			pageRanges := r.URL.Query().Get("pdf.pageRanges")
			if pageRanges == "" {
				// FIXME: legacy support; remove in some future release.
				pageRanges = targetURL.Query().Get("pdf.pageRanges")
			}
			if pageRanges != "" {
				printerOpts = append(printerOpts, service.WithPageRanges(pageRanges))
				span.SetAttributes(attribute.String("pdf.pageRanges", pageRanges))
			}

			var err error
			printer, err = service.NewPDFPrinter(printerOpts...)
			if err != nil {
				span.SetStatus(codes.Error, "invalid pdf printer option")
				span.RecordError(err)
				http.Error(w, fmt.Sprintf("invalid request: %v", err), http.StatusBadRequest)
				return
			}
			span.SetAttributes(attribute.String("encoding", "pdf"))

			pdfLandscape := r.URL.Query().Get("pdf.landscape")
			if pdfLandscape == "" {
				// FIXME: legacy support; remove in some future release.
				pdfLandscape = targetURL.Query().Get("pdf.landscape")
			}
			if pdfLandscape != "" {
				options = append(options, service.WithLandscape(pdfLandscape == "true"))
				span.SetAttributes(attribute.Bool("pdf.landscape", pdfLandscape == "true"))
			}
		case "png":
			var printerOpts []service.PNGPrinterOption
			if height == -1 {
				printerOpts = append(printerOpts, service.WithFullHeight(true))
				options = append(options, service.WithViewport(width, int(math.Floor(0.75*float64(width))))) // add some height to make scrolling faster
				span.SetAttributes(attribute.Bool("fullHeight", true))
			}

			var err error
			printer, err = service.NewPNGPrinter(printerOpts...)
			if err != nil {
				span.SetStatus(codes.Error, "invalid png printer option")
				span.RecordError(err)
				http.Error(w, fmt.Sprintf("invalid request: %v", err), http.StatusBadRequest)
				return
			}
			span.SetAttributes(attribute.String("encoding", "png"))
		default:
			span.SetStatus(codes.Error, "invalid encoding query param")
			span.RecordError(errors.New("invalid encoding"), trace.WithAttributes(attribute.String("encoding", encoding)))
			http.Error(w, fmt.Sprintf("invalid 'encoding' query parameter: %q", encoding), http.StatusBadRequest)
			return
		}
		if acceptLanguage := r.Header.Get("Accept-Language"); acceptLanguage != "" {
			options = append(options, service.WithHeader("Accept-Language", acceptLanguage))
			span.SetAttributes(attribute.String("Accept-Language", acceptLanguage))
		}

		start := time.Now()
		body, contentType, err := browser.Render(ctx, rawTargetURL, printer, options...)
		if err != nil {
			MetricRenderDuration.WithLabelValues("error").Observe(time.Since(start).Seconds())
			span.SetStatus(codes.Error, "rendering failed")
			span.RecordError(err)
			if errors.Is(err, context.DeadlineExceeded) ||
				errors.Is(err, context.Canceled) ||
				errors.Is(err, service.ErrBrowserReadinessTimeout) {
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
		span.SetStatus(codes.Ok, "rendered successfully")

		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(body)))
		w.WriteHeader(http.StatusOK)
		// TODO(perf): we could stream the bytes through from the browser to the response...
		_, _ = w.Write(body)
	})
}
