package browser

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"

	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/cdproto/network"
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
func WithChromium(ctx context.Context, cfg config.BrowserConfig, do func(context.Context, Browser) error) error {
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
	requestsCtx, requestsSpan := tracer.Start(ctx, "request handler")
	defer requestsSpan.End()
	chromedp.ListenTarget(browserCtx, func(ev any) {
		// We MUST NOT issue new actions within this goroutine. Spawn a new one, ALWAYS.
		// See the docs of ListenTarget for more.

		switch e := ev.(type) {
		case *fetch.EventRequestPaused:
			go func() {
				// We need to tell the browser to continue the request.
				// However, in order to also add a tracing header to the request without triggering CORS checks, we must do this work here.

				if sc := trace.SpanFromContext(browserCtx); sc != nil && sc.IsRecording() {
					otel.GetTextMapPropagator().Inject(browserCtx, chromedpNetworkHeadersCarrier(e.Request.Headers))
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

			if err := do(ctx, &chromium{}); err != nil {
				return fmt.Errorf("failed to execute browser function: %w", err)
			}
			return nil
		})); err != nil {
		return fmt.Errorf("chromedp run failed: %w", err)
	}

	return nil
}

type chromium struct {
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
