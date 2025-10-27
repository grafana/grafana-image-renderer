package browser

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"net/http"
	"slices"
	"strings"
	"sync"

	"github.com/chromedp/cdproto/browser"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/grafana/grafana-image-renderer/pkg/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Spawn a new Chromium browser with the given configuration, and execute the provided function with it.
//
// TODO: Use a Chromium-specific configuration rather than a generic BrowserConfig. This will differ per browser.
func WithChromium(ctx context.Context, cfg config.BrowserConfig, do ...Action) error {
	tracer := tracer(ctx)
	ctx, span := tracer.Start(ctx, "WithChromium")
	defer span.End()

	allocatorOpts, err := createChromiumAllocatorOptions(cfg)
	if err != nil {
		return fmt.Errorf("failed to create Chromium allocator options: %w", err)
	}
	allocatorCtx, cancelAllocator := chromedp.NewExecAllocator(ctx, allocatorOpts...)
	defer cancelAllocator()
	browserCtx, cancelBrowser := chromedp.NewContext(allocatorCtx, chromiumLoggers(ctx))
	defer cancelBrowser()

	currentRequests := make(map[network.RequestID]trace.Span)
	currentRequestsLock := &sync.RWMutex{}
	requestsCtx, requestsSpan := tracer.Start(browserCtx, "request handler")
	defer requestsSpan.End()
	chromedp.ListenTarget(requestsCtx, func(ev any) {
		// We MUST NOT issue new actions within this goroutine. Spawn a new one, ALWAYS.
		// See the docs of ListenTarget for more.

		switch e := ev.(type) {
		case *fetch.EventRequestPaused:
			go func() {
				// We need to tell the browser to continue the request.
				// However, in order to also add a tracing header to the request without triggering CORS checks, we must do this work here.

				if sc := trace.SpanFromContext(requestsCtx); sc != nil && sc.IsRecording() {
					otel.GetTextMapPropagator().Inject(requestsCtx, chromedpNetworkHeadersCarrier(e.Request.Headers))
				}

				hdrs := make([]*fetch.HeaderEntry, 0, len(e.Request.Headers))
				for k, v := range e.Request.Headers {
					hdrs = append(hdrs, &fetch.HeaderEntry{Name: k, Value: fmt.Sprintf("%v", v)})
				}

				ctx, span := tracer.Start(requestsCtx, "fetch.ContinueRequest",
					trace.WithAttributes(
						attribute.String("requestID", string(e.RequestID)),
						attribute.String("url", e.Request.URL),
						attribute.String("method", e.Request.Method),
						attribute.Int("headers", len(e.Request.Headers)),
					))
				defer span.End()
				cdpCtx := chromedp.FromContext(requestsCtx)
				ctx = cdp.WithExecutor(ctx, cdpCtx.Target)

				if err := fetch.ContinueRequest(e.RequestID).WithHeaders(hdrs).Do(ctx); err != nil {
					span.SetStatus(codes.Error, err.Error())
				}
			}()

		case *network.EventRequestWillBeSent:
			_, span := tracer.Start(requestsCtx, "Browser HTTP request",
				trace.WithTimestamp(e.Timestamp.Time()),
				trace.WithAttributes(
					attribute.String("requestID", string(e.RequestID)),
					attribute.String("url", e.Request.URL),
					attribute.String("method", e.Request.Method),
					attribute.String("type", string(e.Type)),
				))

			currentRequestsLock.Lock() // lock at the end, so that we minimise how long we hold it
			currentRequests[e.RequestID] = span
			currentRequestsLock.Unlock()

		case *network.EventResponseReceived:
			currentRequestsLock.Lock()
			span, ok := currentRequests[e.RequestID]
			delete(currentRequests, e.RequestID) // no point keeping it around anymore.
			currentRequestsLock.Unlock()         // we want to minimise lock time
			if !ok {
				return
			}
			statusText := e.Response.StatusText
			if statusText == "" {
				statusText = http.StatusText(int(e.Response.Status))
			}
			span.SetAttributes(
				attribute.Int("status", int(e.Response.Status)),
				attribute.String("statusText", statusText),
				attribute.String("mimeType", e.Response.MimeType),
				attribute.String("protocol", e.Response.Protocol),
			)
			if e.Response.Status >= 400 {
				span.SetStatus(codes.Error, fmt.Sprintf("%d %s", e.Response.Status, statusText))
			} else {
				span.SetStatus(codes.Ok, fmt.Sprintf("%d %s", e.Response.Status, statusText))
			}
			span.End(trace.WithTimestamp(e.Timestamp.Time()))
		}
	})

	if err := chromedp.Run(browserCtx,
		network.Enable(),
		fetch.Enable(),
		chromedp.ActionFunc(func(ctx context.Context) error {
			ctx, span := tracer.Start(ctx, "Browser interaction")
			defer span.End()

			for _, do := range do {
				if err := do(ctx, chromium{
					currentRequests:     currentRequests,
					currentRequestsLock: currentRequestsLock,
				}); err != nil {
					span.SetStatus(codes.Error, "failed to execute browser function")
					span.RecordError(err)
					return fmt.Errorf("failed to execute browser function: %w", err)
				}
			}
			span.SetStatus(codes.Ok, "browser functions executed successfully")
			return nil
		})); err != nil {
		return fmt.Errorf("chromedp run failed: %w", err)
	}

	return nil
}

type chromium struct {
	currentRequests     map[network.RequestID]trace.Span
	currentRequestsLock *sync.RWMutex
}

func (c chromium) GetPID(ctx context.Context) (int32, error) {
	tracer := tracer(ctx)
	ctx, span := tracer.Start(ctx, "(*chromium).GetPID")
	defer span.End()

	cdpCtx := chromedp.FromContext(ctx)
	proc := cdpCtx.Browser.Process()
	if proc == nil {
		return -1, fmt.Errorf("assertion failure: browser process is nil for a locally allocated browser?")
	}
	return int32(proc.Pid), nil
}

func (c chromium) GetCurrentNetworkRequests(ctx context.Context) (int, error) {
	tracer := tracer(ctx)
	_, span := tracer.Start(ctx, "(*chromium).GetCurrentNetworkRequests")
	defer span.End()

	c.currentRequestsLock.RLock()
	defer c.currentRequestsLock.RUnlock()
	return len(c.currentRequests), nil
}

func (c chromium) SetPageScale(ctx context.Context, scale float64) error {
	tracer := tracer(ctx)
	ctx, span := tracer.Start(ctx, "(*chromium).SetPageScale",
		trace.WithAttributes(attribute.Float64("scale", scale)))
	defer span.End()

	return emulation.SetPageScaleFactor(scale).Do(ctx)
}

func (c chromium) SetViewPort(ctx context.Context, width, height int, orientation Orientation) error {
	tracer := tracer(ctx)
	ctx, span := tracer.Start(ctx, "(*chromium).SetViewPort",
		trace.WithAttributes(
			attribute.Int("width", width),
			attribute.Int("height", height),
			attribute.String("orientation", orientation.String())))
	defer span.End()

	if !orientation.IsValid() {
		return fmt.Errorf("invalid orientation: %s", orientation)
	}

	orientationOption := chromedp.EmulatePortrait
	if orientation == OrientationLandscape {
		orientationOption = chromedp.EmulateLandscape
	}

	return chromedp.EmulateViewport(int64(width), int64(height), orientationOption).Do(ctx)
}

func (c chromium) SetExtraHeaders(ctx context.Context, headers map[string]string) error {
	tracer := tracer(ctx)
	ctx, span := tracer.Start(ctx, "(*chromium).SetExtraHeaders",
		trace.WithAttributes(attribute.Int("headerCount", len(headers))))
	defer span.End()

	if len(headers) == 0 {
		return nil
	}
	// network.Headers is a map[string]any :(
	hdrs := make(network.Headers, len(headers))
	for k, v := range headers {
		hdrs[k] = v
	}
	return network.SetExtraHTTPHeaders(hdrs).Do(ctx)
}

func (c chromium) SetCookie(ctx context.Context, cookie config.Cookie) error {
	tracer := tracer(ctx)
	ctx, span := tracer.Start(ctx, "(*chromium).SetCookie",
		trace.WithAttributes(
			attribute.String("name", cookie.Name),
			attribute.String("domain", cookie.Domain),
			attribute.Bool("httpOnly", cookie.HTTPOnly),
			attribute.Bool("secure", cookie.Secure)))
	defer span.End()

	return network.SetCookie(cookie.Name, cookie.Value).
		WithDomain(cookie.Domain).
		WithHTTPOnly(cookie.HTTPOnly).
		WithSecure(cookie.Secure).
		Do(ctx)
}

func (c chromium) NavigateAndWait(ctx context.Context, url string) error {
	tracer := tracer(ctx)
	ctx, span := tracer.Start(ctx, "(*chromium).NavigateAndWait",
		trace.WithAttributes(attribute.String("url", url)))
	defer span.End()

	return chromedp.Navigate(url).Do(ctx)
}

func (c chromium) Evaluate(ctx context.Context, js string) error {
	tracer := tracer(ctx)
	ctx, span := tracer.Start(ctx, "(*chromium).Evaluate")
	defer span.End()

	return chromedp.Evaluate(js, nil).Do(ctx)
}

func (c chromium) EvaluateToInt(ctx context.Context, js string) (int, error) {
	tracer := tracer(ctx)
	ctx, span := tracer.Start(ctx, "(*chromium).EvaluateToInt")
	defer span.End()

	var result int
	if err := chromedp.Evaluate(js, &result).Do(ctx); err != nil {
		span.SetStatus(codes.Error, "evaluation failed")
		span.RecordError(err)
		return 0, err
	}
	span.SetStatus(codes.Ok, "evaluation succeeded")
	span.SetAttributes(attribute.Int("result", result))
	return result, nil
}

func (c chromium) PrintPDF(ctx context.Context, options PDFOptions) ([]byte, error) {
	tracer := tracer(ctx)
	ctx, span := tracer.Start(ctx, "(*chromium).PrintPDF",
		trace.WithAttributes(
			attribute.Bool("includeBackground", options.IncludeBackground),
			attribute.Bool("landscape", options.Landscape),
			attribute.Float64("paperWidth", options.PaperWidth),
			attribute.Float64("paperHeight", options.PaperHeight),
			attribute.Float64("scale", options.Scale),
			attribute.String("pageRanges", options.PageRanges),
		))
	defer span.End()

	data, _, err := page.PrintToPDF().
		WithPrintBackground(options.IncludeBackground).
		WithLandscape(options.Landscape).
		WithPaperWidth(options.PaperWidth).
		WithPaperHeight(options.PaperHeight).
		WithScale(options.Scale).
		WithPageRanges(options.PageRanges).
		Do(ctx)
	if err != nil {
		span.SetStatus(codes.Error, "PrintToPDF failed")
		span.RecordError(err)
		return nil, err
	}
	span.SetStatus(codes.Ok, "PrintToPDF succeeded")
	span.SetAttributes(attribute.Int("dataLength", len(data)))
	return data, nil
}

func (c chromium) PrintPNG(ctx context.Context) ([]byte, error) {
	tracer := tracer(ctx)
	ctx, span := tracer.Start(ctx, "(*chromium).PrintPNG")
	defer span.End()

	return page.CaptureScreenshot().
		WithFormat("png").
		// We don't want to use this option: it doesn't take a full window screenshot,
		//   rather it takes a screenshot including content that bleeds outside the viewport (e.g. something 110vh tall).
		// Instead, we change the viewport height to match the content height.
		WithCaptureBeyondViewport(false).
		Do(ctx)
}

func (c chromium) PutDownloadsIn(ctx context.Context, dir string) error {
	tracer := tracer(ctx)
	ctx, span := tracer.Start(ctx, "(*chromium).PutDownloadsIn",
		trace.WithAttributes(attribute.String("directory", dir)))
	defer span.End()

	return browser.SetDownloadBehavior(browser.SetDownloadBehaviorBehaviorAllow).WithDownloadPath(dir).Do(ctx)
}

func createChromiumAllocatorOptions(cfg config.BrowserConfig) ([]chromedp.ExecAllocatorOption, error) {
	opts := chromedp.DefaultExecAllocatorOptions[:]
	opts = append(opts, chromedp.NoFirstRun, chromedp.NoDefaultBrowserCheck)
	if !cfg.GPU {
		opts = append(opts, chromedp.DisableGPU)
	}
	if !cfg.Sandbox {
		opts = append(opts, chromedp.NoSandbox)
	}
	opts = append(opts, chromedp.ExecPath(cfg.Path))
	opts = append(opts, chromedp.WindowSize(cfg.MinWidth, cfg.MinHeight))
	opts = append(opts, chromedp.Env("TZ="+cfg.TimeZone.String()))
	for _, arg := range cfg.Flags {
		arg = strings.TrimPrefix(arg, "--")
		key, value, hadEquals := strings.Cut(arg, "=")
		if !hadEquals || value == "true" {
			opts = append(opts, chromedp.Flag(key, true))
		} else if value == "false" {
			opts = append(opts, chromedp.Flag(key, false))
		} else {
			opts = append(opts, chromedp.Flag(key, value))
		}
	}
	return opts, nil
}

func chromiumLoggers(ctx context.Context) chromedp.ContextOption {
	log := slog.With("service", "chromium")
	return chromedp.WithBrowserOption(
		chromedp.WithBrowserLogf(func(s string, a ...any) {
			if log.Enabled(ctx, slog.LevelInfo) { // defer the Sprintf if possible
				log.InfoContext(ctx, "browser called logf", "message", fmt.Sprintf(s, a...))
			}
		}),
		chromedp.WithBrowserDebugf(func(s string, a ...any) {
			if log.Enabled(ctx, slog.LevelDebug) { // defer the Sprintf if possible
				log.DebugContext(ctx, "browser called debugf", "message", fmt.Sprintf(s, a...))
			}
		}),
		chromedp.WithBrowserErrorf(func(s string, a ...any) {
			// Assume that errors are always logged; this is fair in a production env.
			log.ErrorContext(ctx, "browser called errorf", "message", fmt.Sprintf(s, a...))
		}),
	)
}

type chromedpNetworkHeadersCarrier network.Headers

func (c chromedpNetworkHeadersCarrier) Get(key string) string {
	if len(c) == 0 { // nil-check
		return ""
	}
	v, ok := c[key]
	if !ok {
		return ""
	}
	if vs, ok := v.(string); ok {
		return vs
	}
	return fmt.Sprintf("%v", v)
}

func (c chromedpNetworkHeadersCarrier) Set(key string, value string) {
	c[key] = value
}

func (c chromedpNetworkHeadersCarrier) Keys() []string {
	return slices.Collect(maps.Keys(c))
}
