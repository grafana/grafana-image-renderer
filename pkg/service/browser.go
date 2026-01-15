package service

import (
	"bytes"
	"context"
	"encoding"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"net/http"
	"net/netip"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/chromedp/cdproto/browser"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/grafana/chromedp"
	"github.com/grafana/grafana-image-renderer/pkg/config"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var (
	MetricBrowserGetVersionDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name: "browser_get_version_duration",
		ConstLabels: prometheus.Labels{
			"unit": "seconds",
		},
	})

	MetricBrowserInstancesActive = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "browser_instances_active",
		Help: "How many browser instances are currently launched at any given time?",
	})
	MetricBrowserRenderDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name: "browser_render_duration",
		ConstLabels: prometheus.Labels{
			"unit": "seconds",
		},
		Buckets: []float64{0.1, 0.5, 1, 3, 4, 5, 7, 9, 10, 11, 15, 19, 20, 21, 24, 27, 29, 30, 31, 35, 55, 95, 125, 305, 605},
	})
	MetricBrowserRenderCSVDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name: "browser_render_csv_duration",
		ConstLabels: prometheus.Labels{
			"unit": "seconds",
		},
		Buckets: []float64{0.1, 0.5, 1, 3, 4, 5, 7, 9, 10, 11, 15, 19, 20, 21, 24, 27, 29, 30, 31, 35, 55, 95, 125, 305, 605},
	})
	MetricBrowserActionDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "browser_action_duration",
		ConstLabels: prometheus.Labels{
			"unit": "seconds",
		},
		Buckets: []float64{0.01, 0.03, 0.05, 0.1, 0.3, 0.5, 1, 3, 5, 7, 10, 15, 20, 30, 50, 70, 100, 150, 300},
	}, []string{"action"})
	MetricBrowserRequestSize = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "browser_request_size",
		Help: "How large is the average response from a browser request? This is a best-effort measure, and may not be entirely accurate.",
		ConstLabels: prometheus.Labels{
			"unit": "bytes",
		},
		Buckets: []float64{1, 1024, 4 * 1024, 16 * 1024, 1024 * 1024, 4 * 1024 * 1024, 16 * 1024 * 1024, 64 * 1024 * 1024, 256 * 1024 * 1024},
	}, []string{"mime_type"})
)

var (
	ErrInvalidBrowserOption    = errors.New("invalid browser option")
	ErrBrowserReadinessTimeout = errors.New("timed out waiting for readiness")
)

type BrowserService struct {
	cfg       config.BrowserConfig
	processes *ProcessStatService

	// log is the base logger for the service.
	log *slog.Logger
}

// NewBrowserService creates a new browser service. It is used to launch browsers and control them.
//
// The options are not validated on creation, rather on request.
func NewBrowserService(cfg config.BrowserConfig, processStatService *ProcessStatService) *BrowserService {
	return &BrowserService{
		cfg:       cfg,
		processes: processStatService,
		log:       slog.With("service", "browser"),
	}
}

// GetVersion runs the binary with only a `--version` argument.
// Example output would be something like: `Brave Browser 139.1.81.131` or `Chromium 139.999.999.999`. Some browsers may include more details; do not try to parse this.
func (s *BrowserService) GetVersion(ctx context.Context) (string, error) {
	tracer := tracer(ctx)
	ctx, span := tracer.Start(ctx, "BrowserService.GetVersion")
	defer span.End()

	start := time.Now()
	version, err := exec.CommandContext(ctx, s.cfg.Path, "--version").Output()
	if err != nil {
		return "", fmt.Errorf("failed to get version of browser: %w", err)
	}
	MetricBrowserGetVersionDuration.Observe(time.Since(start).Seconds())
	return string(bytes.TrimSpace(version)), nil
}

type RenderingOption func(config.BrowserConfig) (config.BrowserConfig, error)

// WithTimeZone sets the time-zone of the browser for this request.
func WithTimeZone(loc *time.Location) RenderingOption {
	return func(cfg config.BrowserConfig) (config.BrowserConfig, error) {
		if loc == nil {
			return config.BrowserConfig{}, fmt.Errorf("%w: time-zone location was nil", ErrInvalidBrowserOption)
		}
		if loc.String() == "" {
			return config.BrowserConfig{}, fmt.Errorf("%w: time-zone name is empty", ErrInvalidBrowserOption)
		}
		cfg.TimeZone = loc
		return cfg, nil
	}
}

// WithCookie adds a new cookie to the browser's context.
func WithCookie(name, value, domain string) RenderingOption {
	// if it looks like an IPv6 address, but doesn't contain [] we need to add it when saving the Cookie otherwise Chromium will reject it.
	if strings.Contains(domain, ":") && (!strings.Contains(domain, "[") && !strings.Contains(domain, "]")) {
		ip, err := netip.ParseAddr(domain)
		if err == nil && ip.Is6() {
			domain = "[" + domain + "]"
		}
	}

	return func(cfg config.BrowserConfig) (config.BrowserConfig, error) {
		cfg.Cookies = append(cfg.Cookies, &network.SetCookieParams{
			Name:   name,
			Value:  value,
			Domain: domain,
		})
		return cfg, nil
	}
}

// WithHeader adds a new header sent in _all_ requests from this browser.
//
// You should be careful about using this for authentication or other sensitive information; prefer cookies.
// If you do not use cookies, the user could embed a link to their own website somewhere, which means they'd get the auth tokens!
func WithHeader(name, value string) RenderingOption {
	return func(cfg config.BrowserConfig) (config.BrowserConfig, error) {
		if name == "" {
			return config.BrowserConfig{}, fmt.Errorf("%w: header name was empty", ErrInvalidBrowserOption)
		}
		if cfg.Headers == nil {
			cfg.Headers = make(network.Headers)
		}
		cfg.Headers[name] = value
		return cfg, nil
	}
}

// WithViewport sets the view of the browser: this is the size used by the actual webpage, not the browser window.
//
// A value of -1 is ignored.
// The width and height must be larger than 10 px each; usual values are 1000x500 and 1920x1080, although bigger & smaller work as well.
// You effectively set the aspect ratio with this as well: for 16:9, use a width that is 16px for every 9px it is high, or for 4:3, use a width that is 4px for every 3px it is high.
// For values below the Min values (from the config), we clamp it to these.
func WithViewport(width, height int) RenderingOption {
	clamped := func(v, minimum, maximum int) int {
		if v < minimum {
			return minimum
		} else if v > maximum && maximum > 0 {
			return maximum
		} else {
			return v
		}
	}

	return func(cfg config.BrowserConfig) (config.BrowserConfig, error) {
		cfg.ApplyAll(func(rc *config.RequestConfig) {
			if width != -1 {
				rc.MinWidth = clamped(width, rc.MinWidth, rc.MaxWidth)
			}
			if height != -1 {
				rc.MinHeight = clamped(height, rc.MinHeight, rc.MaxHeight)
			}
		})

		return cfg, nil
	}
}

// WithPageScaleFactor uses the given scale for all webpages visited by the browser.
func WithPageScaleFactor(factor float64) RenderingOption {
	return func(cfg config.BrowserConfig) (config.BrowserConfig, error) {
		if factor <= 0 {
			return cfg, fmt.Errorf("%w: page scale factor must be positive", ErrInvalidBrowserOption)
		}

		cfg.ApplyAll(func(rc *config.RequestConfig) {
			rc.PageScaleFactor = factor
		})

		return cfg, nil
	}
}

func WithLandscape(landscape bool) RenderingOption {
	return func(cfg config.BrowserConfig) (config.BrowserConfig, error) {
		cfg.ApplyAll(func(rc *config.RequestConfig) {
			rc.Landscape = landscape
		})

		return cfg, nil
	}
}

func (s *BrowserService) Render(ctx context.Context, url string, printer Printer, optionFuncs ...RenderingOption) ([]byte, string, error) {
	tracer := tracer(ctx)
	ctx, span := tracer.Start(ctx, "BrowserService.Render")
	defer span.End()
	start := time.Now()

	if url == "" {
		return nil, "text/plain", fmt.Errorf("url must not be empty")
	}

	cfg := s.cfg.DeepClone()
	for _, f := range optionFuncs {
		var err error
		cfg, err = f(cfg)
		if err != nil {
			return nil, "text/plain", fmt.Errorf("failed to apply rendering option: %w", err)
		}
	}
	span.AddEvent("options applied")

	chromiumCwd, err := os.MkdirTemp("", "")
	if err != nil {
		return nil, "text/plain", fmt.Errorf("failed to create temporary directory for browser CWD: %w", err)
	}
	defer func() { _ = os.RemoveAll(chromiumCwd) }()

	allocatorOptions, err := s.createAllocatorOptions(ctx, cfg, url, chromiumCwd)
	if err != nil {
		return nil, "text/plain", fmt.Errorf("failed to create allocator options: %w", err)
	}
	allocatorCtx, cancelAllocator := chromedp.NewExecAllocator(ctx, allocatorOptions...)
	defer cancelAllocator()
	browserCtx, cancelBrowser := chromedp.NewContext(allocatorCtx, browserLoggers(ctx, s.log))
	defer cancelBrowser()
	span.AddEvent("browser allocated")

	s.handleNetworkEvents(browserCtx)

	orientation := chromedp.EmulatePortrait

	requestConfig := cfg.LookupRequestConfig(span, url)
	if requestConfig.Landscape {
		orientation = chromedp.EmulateLandscape
	}

	fileChan := make(chan []byte, 1) // buffered: we don't want the browser to stick around while we try to export this value.
	actions := []chromedp.Action{
		observingAction("trackProcess", trackProcess(browserCtx, s.processes)),
		observingAction("network.Enable", network.Enable()), // required by waitForReady
		observingAction("fetch.Enable", fetch.Enable()),     // required by handleNetworkEvents
		observingAction("SetPageScaleFactor", emulation.SetPageScaleFactor(requestConfig.PageScaleFactor)),
		observingAction("EmulateViewport", chromedp.EmulateViewport(int64(requestConfig.MinWidth), int64(requestConfig.MinHeight), orientation, chromedp.EmulateScale(requestConfig.PageScaleFactor))),
		observingAction("setHeaders", setHeaders(browserCtx, cfg.Headers)),
		observingAction("setCookies", setCookies(cfg.Cookies)),
		observingAction("Navigate", chromedp.Navigate(url)),
		observingAction("WaitReady(body)", chromedp.WaitReady("body", chromedp.ByQuery)), // wait for a body to exist; this is when the page has started to actually render
		observingAction("scrollForElements", scrollForElements(requestConfig.TimeBetweenScrolls)),
		observingAction("waitForDuration", waitForDuration(requestConfig.ReadinessPriorWait)),
		observingAction("waitForReady", waitForReady(browserCtx, cfg, url)),
		observingAction("printer.prepare", printer.prepare(cfg, url)),
		observingAction("printer.action", printer.action(fileChan, cfg, url)),
	}
	span.AddEvent("actions created")
	MetricBrowserInstancesActive.Inc()
	defer MetricBrowserInstancesActive.Dec()
	if err := chromedp.Run(browserCtx, actions...); err != nil {
		return nil, "text/plain", fmt.Errorf("failed to run browser: %w", err)
	}
	span.AddEvent("actions completed")

	select {
	case fileContents := <-fileChan:
		MetricBrowserRenderDuration.Observe(time.Since(start).Seconds())
		return fileContents, printer.contentType(), nil
	default:
		span.AddEvent("no data received from printer")
		return nil, "text/plain", fmt.Errorf("failed to render: no data received after browser quit")
	}
}

// RenderCSV visits a web page and downloads the CSV inside.
//
// You may be thinking: what the hell are we doing? Why are we using a browser for this?
// The CSV endpoint just returns HTML. The actual query is done by the browser, and then a script _in the webpage_ downloads it as a CSV file.
// This SHOULD be replaced at some point, such that the Grafana server does all the work; this is just not acceptable behaviour...
func (s *BrowserService) RenderCSV(ctx context.Context, url, renderKey, domain, acceptLanguage string) ([]byte, string, error) {
	tracer := tracer(ctx)
	ctx, span := tracer.Start(ctx, "BrowserService.RenderCSV")
	defer span.End()
	start := time.Now()

	if url == "" {
		return nil, "", fmt.Errorf("url must not be empty")
	}

	chromiumCwd, err := os.MkdirTemp("", "")
	if err != nil {
		return nil, "text/plain", fmt.Errorf("failed to create temporary directory for browser CWD: %w", err)
	}
	defer func() { _ = os.RemoveAll(chromiumCwd) }()

	chromiumDownloadDir := filepath.Join(chromiumCwd, "_gir_downloads")
	realDownloadDir := filepath.Join(chromiumCwd, "_gir_downloads")
	if s.cfg.Namespaced {
		chromiumDownloadDir = "/tmp/_gir_downloads"
	}
	if err := os.MkdirAll(realDownloadDir, 0o755); err != nil {
		return nil, "", fmt.Errorf("failed to create download directory at %q: %w", realDownloadDir, err)
	}

	allocatorOptions, err := s.createAllocatorOptions(ctx, s.cfg, url, chromiumCwd)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create allocator options: %w", err)
	}
	allocatorCtx, cancelAllocator := chromedp.NewExecAllocator(ctx, allocatorOptions...)
	defer cancelAllocator()
	browserCtx, cancelBrowser := chromedp.NewContext(allocatorCtx, browserLoggers(ctx, s.log))
	defer cancelBrowser()
	span.AddEvent("browser allocated")

	var headers network.Headers
	if acceptLanguage != "" {
		headers = network.Headers{
			"Accept-Language": acceptLanguage,
		}
	}

	s.handleNetworkEvents(browserCtx)

	actions := []chromedp.Action{
		observingAction("trackProcess", trackProcess(browserCtx, s.processes)),
		observingAction("network.Enable", network.Enable()),
		observingAction("setHeaders", setHeaders(browserCtx, headers)),
		observingAction("setCookies", setCookies([]*network.SetCookieParams{
			{
				Name:   "renderKey",
				Value:  renderKey,
				Domain: domain,
			},
		})),
		observingAction("SetDownloadBehavior", browser.SetDownloadBehavior(browser.SetDownloadBehaviorBehaviorAllow).WithDownloadPath(chromiumDownloadDir)),
		observingAction("Navigate", chromedp.Navigate(url)),
	}
	MetricBrowserInstancesActive.Inc()
	defer MetricBrowserInstancesActive.Dec()
	if err := chromedp.Run(browserCtx, actions...); err != nil {
		return nil, "", fmt.Errorf("failed to run browser: %w", err)
	}
	span.AddEvent("actions completed")

	// Wait for the file to be downloaded.
	filename := ""
	for {
		if err := ctx.Err(); err != nil {
			return nil, "", err
		}

		entries, err := os.ReadDir(realDownloadDir)
		if err != nil {
			return nil, "", fmt.Errorf("failed to read files in chromium's working directory: %w", err)
		}

		for _, entry := range entries {
			if strings.HasSuffix(strings.ToLower(entry.Name()), ".csv") {
				filename = filepath.Join(realDownloadDir, entry.Name())
				break
			}
		}
		if filename != "" {
			break
		}

		select {
		case <-ctx.Done():
			return nil, "", ctx.Err()
		case <-time.After(100 * time.Millisecond):
			// try again
		}
	}
	span.AddEvent("downloaded file located", trace.WithAttributes(attribute.String("path", filename)))

	fileContents, err := os.ReadFile(filename)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read temporary file %q: %w", filename, err)
	}

	MetricBrowserRenderCSVDuration.Observe(time.Since(start).Seconds())
	return fileContents, filepath.Base(filename), nil
}

func (s *BrowserService) createAllocatorOptions(ctx context.Context, cfg config.BrowserConfig, url string, cwd string) ([]chromedp.ExecAllocatorOption, error) {
	tracer := tracer(ctx)
	ctx, span := tracer.Start(ctx, "BrowserService.createAllocatorOptions")
	defer span.End()

	opts := chromedp.DefaultExecAllocatorOptions[:]
	opts = append(opts, chromedp.NoFirstRun, chromedp.NoDefaultBrowserCheck)
	if !cfg.GPU {
		opts = append(opts, chromedp.DisableGPU)
	}
	if !cfg.Sandbox {
		opts = append(opts, chromedp.NoSandbox)
	}
	if cfg.Namespaced {
		var traceID string
		if sc := trace.SpanContextFromContext(ctx); sc.IsValid() && sc.HasTraceID() {
			traceID = sc.TraceID().String()
		}

		opts = append(opts, chromedp.ExecPath("/proc/self/exe"))
		// TODO: Add additional flags for necessary mounts for the browser if it is not Chromium?
		opts = append(opts, chromedp.InitialArgs("_internal_sandbox", "bootstrap", "--tmp", cwd, "--cwd", "/tmp", "--trace", traceID, "--", cfg.Path))
		opts = append(opts, chromedp.UserDataDir("/tmp"))
	} else {
		opts = append(opts, chromedp.ExecPath(cfg.Path))
		opts = append(opts, chromedp.UserDataDir(cwd))
	}

	if _, exists := os.LookupEnv("XDG_CONFIG_HOME"); !exists {
		opts = append(opts, chromedp.Env("XDG_CONFIG_HOME="+cwd))
	}
	if _, exists := os.LookupEnv("XDG_CACHE_HOME"); !exists {
		opts = append(opts, chromedp.Env("XDG_CACHE_HOME="+cwd))
	}

	requestConfig := cfg.LookupRequestConfig(span, url)
	opts = append(opts, chromedp.WindowSize(requestConfig.MinWidth, requestConfig.MinHeight))
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

func (s *BrowserService) handleNetworkEvents(browserCtx context.Context) {
	requests := make(map[network.RequestID]trace.Span)
	requestsMutex := &sync.Mutex{}

	requestMimes := make(map[network.RequestID]string)
	requestMimesMutex := &sync.Mutex{}

	tracer := tracer(browserCtx)

	chromedp.ListenTarget(browserCtx, func(ev any) {
		// We MUST NOT issue new actions within this goroutine. Spawn a new one, ALWAYS.
		// See the docs of ListenTarget for more.

		switch e := ev.(type) {
		case *fetch.EventRequestPaused:
			go func() {
				if sc := trace.SpanFromContext(browserCtx); sc != nil && sc.IsRecording() {
					otel.GetTextMapPropagator().Inject(browserCtx, networkHeadersCarrier(e.Request.Headers))
				}

				hdrs := make([]*fetch.HeaderEntry, 0, len(e.Request.Headers))
				for k, v := range e.Request.Headers {
					hdrs = append(hdrs, &fetch.HeaderEntry{Name: k, Value: fmt.Sprintf("%v", v)})
				}

				ctx, span := tracer.Start(browserCtx, "fetch.ContinueRequest",
					trace.WithAttributes(
						attribute.String("requestID", string(e.RequestID)),
						attribute.String("url", e.Request.URL),
						attribute.String("method", e.Request.Method),
						attribute.Int("headers", len(e.Request.Headers)),
					))
				defer span.End()
				cdpCtx := chromedp.FromContext(browserCtx)
				ctx = cdp.WithExecutor(ctx, cdpCtx.Target)

				if err := fetch.ContinueRequest(e.RequestID).WithHeaders(hdrs).Do(ctx); err != nil {
					span.SetStatus(codes.Error, err.Error())
					slog.DebugContext(ctx, "failed to continue request", "requestID", e.RequestID, "error", err)
				}
			}()

		case *network.EventRequestWillBeSent:
			_, span := tracer.Start(browserCtx, "Browser HTTP request",
				trace.WithTimestamp(e.Timestamp.Time()),
				trace.WithAttributes(
					attribute.String("requestID", string(e.RequestID)),
					attribute.String("url", e.Request.URL),
					attribute.String("method", e.Request.Method),
					attribute.String("type", string(e.Type)),
				))

			requestsMutex.Lock()
			requests[e.RequestID] = span
			requestsMutex.Unlock()

		case *network.EventResponseReceived:
			requestMimesMutex.Lock()
			requestMimes[e.RequestID] = e.Response.MimeType
			requestMimesMutex.Unlock()

			requestsMutex.Lock()
			span, ok := requests[e.RequestID]
			delete(requests, e.RequestID) // no point keeping it around anymore.
			requestsMutex.Unlock()
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

		case *network.EventLoadingFinished:
			requestMimesMutex.Lock()
			mime, ok := requestMimes[e.RequestID]
			delete(requestMimes, e.RequestID)
			requestMimesMutex.Unlock()
			if !ok {
				return
			}
			MetricBrowserRequestSize.WithLabelValues(mime).Observe(float64(e.EncodedDataLength))
		}
	})
}

func browserLoggers(ctx context.Context, log *slog.Logger) chromedp.ContextOption {
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

var (
	_ encoding.TextUnmarshaler = (*PaperSize)(nil)
	_ json.Unmarshaler         = (*PaperSize)(nil)
)

// PaperSize is the size of various paper formats.
//
// Ref: https://pptr.dev/api/puppeteer.paperformat#remarks
type PaperSize string

// The valid paper sizes.
// These are all lower-case (even though it is not standard for e.g. A4), as that's easier to match against with case-insensitivity.
const (
	PaperLetter  PaperSize = "letter"
	PaperLegal   PaperSize = "legal"
	PaperTabloid PaperSize = "tabloid"
	PaperLedger  PaperSize = "ledger"
	PaperA0      PaperSize = "a0"
	PaperA1      PaperSize = "a1"
	PaperA2      PaperSize = "a2"
	PaperA3      PaperSize = "a3"
	PaperA4      PaperSize = "a4"
	PaperA5      PaperSize = "a5"
	PaperA6      PaperSize = "a6"
)

func (p *PaperSize) UnmarshalText(text []byte) error {
	text = bytes.ToLower(text)
	switch PaperSize(text) {
	case PaperLetter, PaperLegal, PaperTabloid, PaperLedger,
		PaperA0, PaperA1, PaperA2, PaperA3, PaperA4, PaperA5, PaperA6:
		*p = PaperSize(text)
		return nil
	default:
		return fmt.Errorf("invalid paper size name: %q", text)
	}
}

func (p *PaperSize) UnmarshalJSON(data []byte) error {
	var text string
	if err := json.Unmarshal(data, &text); err != nil {
		return fmt.Errorf("failed to unmarshal paper size: %w", err)
	}
	return p.UnmarshalText([]byte(text))
}

// FormatInches returns the dimensions of the paper size in inches.
// If the paper size is unknown, (-1, -1) is returned along with an error.
func (p PaperSize) FormatInches() (width float64, height float64, err error) {
	// BUG: The puppeteer code have differences with what works in practice; where they exist, it's _always_ 4 pixels.
	//      I haven't figured out why, but this makes a _tiny_ difference in practice. Best guess, it's some JS rounding error(???).
	// https://github.com/puppeteer/puppeteer/blob/e09d56b6559460bc98d8a2811b32852d79135f7b/packages/puppeteer-core/src/common/PDFOptions.ts#L226-L274
	switch p {
	case PaperLetter:
		return 8.5, 11, nil
	case PaperLegal:
		return 8.5, 14, nil
	case PaperTabloid:
		return 11, 17, nil
	case PaperLedger:
		return 17, 11, nil
	case PaperA0:
		// BUG: The puppeteer code says 33.1102 x 46.811, but in practice, it becomes 46.80 high.
		return 33.1102, 46.80, nil
	case PaperA1:
		// BUG: The puppeteer code says 23.3858 x 33.1102, but in practice, it becomes 23.39 wide. As opposed to height in A0. WTF?
		return 23.39, 33.1102, nil
	case PaperA2:
		// BUG: The puppeteer code says 16.5354 x 23.3858, but in practice, it becomes 23.39 high.
		return 16.5354, 23.39, nil
	case PaperA3:
		// BUG: The puppeteer code says 11.6929 x 16.5354, but in practice, it becomes 11.70 wide.
		return 11.70, 16.5354, nil
	case PaperA4:
		// BUG: The puppeteer code says 8.2677 x 11.6929, but in practice, it becomes 11.70 high... which is the opposite way of A0. WTF?
		return 8.2677, 11.70, nil
	case PaperA5:
		return 5.8268, 8.2677, nil
	case PaperA6:
		return 4.1339, 5.8268, nil
	default:
		return -1, -1, fmt.Errorf("unknown paper size: %q", p)
	}
}

type Printer interface {
	prepare(cfg config.BrowserConfig, url string) chromedp.Action
	action(output chan []byte, cfg config.BrowserConfig, url string) chromedp.Action
	contentType() string
}

type pdfPrinter struct {
	paperSize       PaperSize
	printBackground bool
	pageRanges      string // empty string is all pages
}

func (p *pdfPrinter) prepare(_ config.BrowserConfig, _ string) chromedp.Action {
	return chromedp.ActionFunc(func(context.Context) error {
		return nil
	})
}

func (p *pdfPrinter) action(dst chan []byte, cfg config.BrowserConfig, url string) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		tracer := tracer(ctx)
		ctx, span := tracer.Start(ctx, "pdfPrinter.action")
		defer span.End()

		requestConfig := cfg.LookupRequestConfig(span, url)
		span.SetAttributes(
			attribute.String("paperSize", string(p.paperSize)),
			attribute.Bool("printBackground", p.printBackground),
			attribute.Bool("landscape", requestConfig.Landscape),
			attribute.Float64("pageScaleFactor", requestConfig.PageScaleFactor),
		)

		width, height, err := p.paperSize.FormatInches()
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			return fmt.Errorf("failed to get paper size dimensions: %w", err)
		}

		scale := 1.0
		if requestConfig.PageScaleFactor != 0 {
			scale = 1.0 / requestConfig.PageScaleFactor
		}

		// We don't need the stream return value; we don't ask for a stream.
		output, _, err := page.PrintToPDF().
			WithPrintBackground(p.printBackground).
			WithMarginBottom(0).
			WithMarginLeft(0).
			WithMarginRight(0).
			WithMarginTop(0).
			WithLandscape(requestConfig.Landscape).
			WithPaperWidth(width).
			WithPaperHeight(height).
			WithScale(scale).
			WithPageRanges(p.pageRanges).
			Do(ctx)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			return fmt.Errorf("failed to print to PDF: %w", err)
		}
		dst <- output
		span.SetStatus(codes.Ok, "PDF printed successfully")
		return nil
	})
}

func (p *pdfPrinter) contentType() string {
	return "application/pdf"
}

type PDFPrinterOption func(*pdfPrinter) error

func WithPaperSize(size PaperSize) PDFPrinterOption {
	return func(pp *pdfPrinter) error {
		_, _, err := size.FormatInches()
		if err != nil {
			return fmt.Errorf("%w: could not get paper size in inches: %v", ErrInvalidBrowserOption, err)
		}
		pp.paperSize = size
		return nil
	}
}

func WithPrintingBackground(printBackground bool) PDFPrinterOption {
	return func(pp *pdfPrinter) error {
		pp.printBackground = printBackground
		return nil
	}
}

func WithPageRanges(ranges string) PDFPrinterOption {
	return func(pp *pdfPrinter) error {
		pp.pageRanges = ranges
		return nil
	}
}

func NewPDFPrinter(opts ...PDFPrinterOption) (*pdfPrinter, error) {
	printer := &pdfPrinter{
		paperSize:       PaperA4,
		printBackground: true,
	}
	for _, f := range opts {
		if err := f(printer); err != nil {
			return nil, fmt.Errorf("failed to apply PDF printer option: %w", err)
		}
	}
	return printer, nil
}

type pngPrinter struct {
	fullHeight bool
}

func (p *pngPrinter) prepare(cfg config.BrowserConfig, url string) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		if !p.fullHeight {
			return nil
		}

		tracer := tracer(ctx)
		ctx, span := tracer.Start(ctx, "pngPrinter.prepare")
		defer span.End()

		requestConfig := cfg.LookupRequestConfig(span, url)
		span.SetAttributes(
			attribute.Int("currentViewportWidth", requestConfig.MinWidth),
			attribute.Int("currentViewportHeight", requestConfig.MinHeight),
			attribute.Bool("landscape", requestConfig.Landscape),
			attribute.Float64("pageScaleFactor", requestConfig.PageScaleFactor),
		)

		var scrollHeight int
		err := chromedp.Evaluate("document.body.scrollHeight", &scrollHeight).Do(ctx)
		if err != nil {
			span.SetStatus(codes.Error, "failed to get scroll height: "+err.Error())
			return fmt.Errorf("failed to get scroll height: %w", err)
		}
		span.AddEvent("obtained scroll height", trace.WithAttributes(attribute.Int("scrollHeight", scrollHeight)))

		if scrollHeight > requestConfig.MinHeight {
			span.AddEvent("resizing viewport for full height capture",
				trace.WithAttributes(
					attribute.Int("originalHeight", requestConfig.MinHeight),
					attribute.Int("newHeight", scrollHeight),
				))

			orientation := chromedp.EmulatePortrait
			if requestConfig.Landscape {
				orientation = chromedp.EmulateLandscape
			}

			width := int64(requestConfig.MinWidth)
			height := int64(scrollHeight)

			err = chromedp.EmulateViewport(width, height, orientation, chromedp.EmulateScale(requestConfig.PageScaleFactor)).Do(ctx)
			if err != nil {
				span.SetStatus(codes.Error, "failed to resize viewport: "+err.Error())
				return fmt.Errorf("failed to resize viewport for full height: %w", err)
			}

			span.SetStatus(codes.Ok, "viewport resized successfully")
			if err := waitForReady(ctx, cfg, url).Do(ctx); err != nil {
				return fmt.Errorf("failed to wait for readiness after resizing viewport: %w", err)
			}
		} else {
			span.AddEvent("no viewport resize needed")
		}

		return nil
	})
}

func (p *pngPrinter) action(dst chan []byte, cfg config.BrowserConfig, url string) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		tracer := tracer(ctx)
		ctx, span := tracer.Start(ctx, "pngPrinter.action",
			trace.WithAttributes(
				attribute.Bool("fullHeight", p.fullHeight),
			))
		defer span.End()

		output, err := page.CaptureScreenshot().
			WithFormat(page.CaptureScreenshotFormatPng).
			// We don't want to use this option: it doesn't take a full window screenshot,
			//   rather it takes a screenshot including content that bleeds outside the viewport (e.g. something 110vh tall).
			// Instead, we change the viewport height to match the content height.
			WithCaptureBeyondViewport(false).
			Do(ctx)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			return fmt.Errorf("failed to capture screenshot: %w", err)
		}
		dst <- output
		span.SetStatus(codes.Ok, "screenshot captured")
		return nil
	})
}

func (p *pngPrinter) contentType() string {
	return "image/png"
}

type PNGPrinterOption func(*pngPrinter) error

func WithFullHeight(fullHeight bool) PNGPrinterOption {
	return func(pp *pngPrinter) error {
		pp.fullHeight = fullHeight
		return nil
	}
}

func NewPNGPrinter(opts ...PNGPrinterOption) (*pngPrinter, error) {
	printer := &pngPrinter{fullHeight: false}
	for _, f := range opts {
		if err := f(printer); err != nil {
			return nil, fmt.Errorf("failed to apply PNG printer option: %w", err)
		}
	}
	return printer, nil
}

func setHeaders(browserCtx context.Context, headers network.Headers) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		tracer := tracer(ctx)
		ctx, span := tracer.Start(ctx, "setHeaders",
			trace.WithAttributes(attribute.Int("count", len(headers))))
		defer span.End()

		if len(headers) == 0 {
			return nil
		}
		return network.SetExtraHTTPHeaders(headers).Do(ctx)
	})
}

func setCookies(cookies []*network.SetCookieParams) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		tracer := tracer(ctx)
		ctx, span := tracer.Start(ctx, "setCookies",
			trace.WithAttributes(attribute.Int("count", len(cookies))))
		defer span.End()

		for _, cookie := range cookies {
			ctx, span := tracer.Start(ctx, "setCookie",
				trace.WithAttributes(
					attribute.String("name", cookie.Name),
					attribute.String("domain", cookie.Domain),
					attribute.Bool("httpOnly", cookie.HTTPOnly),
					attribute.Bool("secure", cookie.Secure)))
			if err := cookie.Do(ctx); err != nil {
				span.SetStatus(codes.Error, err.Error())
				span.End()
				return fmt.Errorf("failed to set cookie %q: %w", cookie.Name, err)
			}
			span.SetStatus(codes.Ok, "cookie set successfully")
			span.End()
		}
		return nil
	})
}

func scrollForElements(timeBetweenScrolls time.Duration) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		tracer := tracer(ctx)
		ctx, span := tracer.Start(ctx, "scrollForElements")
		defer span.End()

		var scrolls int
		err := chromedp.Evaluate(`Math.floor(document.body.scrollHeight / window.innerHeight)`, &scrolls).Do(ctx)
		if err != nil {
			return fmt.Errorf("failed to calculate scrolls required: %w", err)
		}
		span.AddEvent("calculated scrolls", trace.WithAttributes(attribute.Int("scrolls", scrolls)))

		select {
		case <-time.After(timeBetweenScrolls):
			span.AddEvent("initial wait complete")
		case <-ctx.Done():
			span.AddEvent("context completed before finishing initial wait")
			return ctx.Err()
		}
		for range scrolls {
			err := chromedp.Evaluate(`window.scrollBy(0, window.innerHeight, { behavior: 'instant' })`, nil).Do(ctx)
			span.AddEvent("scrolled one viewport")
			if err != nil {
				return fmt.Errorf("failed to scroll: %w", err)
			}
			select {
			case <-time.After(timeBetweenScrolls):
				span.AddEvent("wait after scroll complete")
			case <-ctx.Done():
				span.AddEvent("context completed before finishing scroll wait")
				return ctx.Err()
			}
		}

		err = chromedp.Evaluate(`window.scrollTo(0, 0, { behavior: 'instant' })`, nil).Do(ctx)
		if err != nil {
			return fmt.Errorf("failed to scroll to top: %w", err)
		}
		span.AddEvent("scrolled to top")

		return nil
	})
}

func waitForReady(browserCtx context.Context, cfg config.BrowserConfig, url string) chromedp.Action {
	t := tracer(browserCtx)
	browserCtx, span := t.Start(browserCtx, "waitForReadySetup")
	defer span.End()

	requestConfig := cfg.LookupRequestConfig(span, url)
	span.SetAttributes(
		attribute.String("timeout", requestConfig.ReadinessTimeout.String()),
	)

	getRunningQueries := func(ctx context.Context) (bool, error) {
		var running bool
		err := chromedp.Evaluate(`!!(window.__grafanaSceneContext && window.__grafanaRunningQueryCount > 0)`, &running).Do(ctx)
		return running, err
	}
	getDOMHashCode := func(ctx context.Context) (int, error) {
		var hashCode int
		err := chromedp.Evaluate(`((x) => {
			let h = 0;
			for (let i = 0; i < x.length; i++) {
				h = (Math.imul(31, h) + x.charCodeAt(i)) | 0;
			}
			return h;
		})(document.body.toString())`, &hashCode).Do(ctx)
		return hashCode, err
	}

	requests := &atomic.Int64{}
	lastRequest := &atomicTime{} // TODO: use this to wait for network stabilisation.
	lastRequest.Store(time.Now())
	networkListenerCtx, cancelNetworkListener := context.WithCancel(browserCtx)
	if !requestConfig.ReadinessDisableNetworkWait {
		chromedp.ListenTarget(networkListenerCtx, func(ev any) {
			switch ev.(type) {
			case *network.EventRequestWillBeSent:
				requests.Add(1)
				lastRequest.Store(time.Now())
			case *network.EventLoadingFinished, *network.EventLoadingFailed:
				requests.Add(-1)
			}
		})
	}

	return chromedp.ActionFunc(func(ctx context.Context) error {
		defer cancelNetworkListener()

		tracer := tracer(ctx)
		ctx, span := tracer.Start(ctx, "waitForReady",
			trace.WithAttributes(attribute.String("timeout", requestConfig.ReadinessTimeout.String())))
		defer span.End()

		start := time.Now()

		var readinessTimeout <-chan time.Time
		if requestConfig.ReadinessTimeout > 0 {
			readinessTimeout = time.After(requestConfig.ReadinessTimeout)
		}

		hasSeenAnyQuery := false
		numSuccessfulCycles := 0

		var domHashCode int
		initialDOMPass := true

		for {
			select {
			case <-ctx.Done():
				span.SetStatus(codes.Error, "context completed before readiness detected")
				return ctx.Err()
			case <-readinessTimeout:
				span.SetStatus(codes.Error, ErrBrowserReadinessTimeout.Error())
				return ErrBrowserReadinessTimeout

			case <-time.After(requestConfig.ReadinessIterationInterval):
				// Continue with the rest of the code; this is waiting for the next time we can do work.
			}

			if !requestConfig.ReadinessDisableNetworkWait &&
				(requestConfig.ReadinessNetworkIdleTimeout <= 0 || time.Since(start) < requestConfig.ReadinessNetworkIdleTimeout) &&
				requests.Load() > 0 {
				initialDOMPass = true
				span.AddEvent("network requests still ongoing", trace.WithAttributes(attribute.Int64("inflight_requests", requests.Load())))
				continue // still waiting on network requests to complete
			}

			if !requestConfig.ReadinessDisableQueryWait && (requestConfig.ReadinessQueriesTimeout <= 0 || time.Since(start) < requestConfig.ReadinessQueriesTimeout) {
				running, err := getRunningQueries(ctx)
				if err != nil {
					span.SetStatus(codes.Error, err.Error())
					span.RecordError(err)
					return fmt.Errorf("failed to get running queries: %w", err)
				}
				span.AddEvent("queried running queries", trace.WithAttributes(attribute.Bool("running", running)))
				if running {
					initialDOMPass = true
					hasSeenAnyQuery = true
					numSuccessfulCycles = 0
					continue // still waiting on queries to complete
				} else if !hasSeenAnyQuery && (requestConfig.ReadinessFirstQueryTimeout <= 0 || time.Since(start) < requestConfig.ReadinessFirstQueryTimeout) {
					span.AddEvent("no first query detected yet; giving it more time")
					continue
				} else if numSuccessfulCycles+1 < requestConfig.ReadinessWaitForNQueryCycles {
					numSuccessfulCycles++
					span.AddEvent("waiting for more successful readiness cycles", trace.WithAttributes(attribute.Int("currentCycle", numSuccessfulCycles), attribute.Int("requiredCycles", requestConfig.ReadinessWaitForNQueryCycles)))
					continue // need more successful cycles
				}
			}

			if !requestConfig.ReadinessDisableDOMHashCodeWait && (requestConfig.ReadinessDOMHashCodeTimeout <= 0 || time.Since(start) < requestConfig.ReadinessDOMHashCodeTimeout) {
				if initialDOMPass {
					var err error
					domHashCode, err = getDOMHashCode(ctx)
					if err != nil {
						span.SetStatus(codes.Error, err.Error())
						span.RecordError(err)
						return fmt.Errorf("failed to get DOM hash code: %w", err)
					}
					span.AddEvent("initial DOM hash code recorded", trace.WithAttributes(attribute.Int("hashCode", domHashCode)))
					initialDOMPass = false
					continue // not stable yet
				}

				newHashCode, err := getDOMHashCode(ctx)
				if err != nil {
					span.SetStatus(codes.Error, err.Error())
					span.RecordError(err)
					return fmt.Errorf("failed to get DOM hash code: %w", err)
				}
				span.AddEvent("subsequent DOM hash code recorded", trace.WithAttributes(attribute.Int("hashCode", newHashCode)))
				if newHashCode != domHashCode {
					span.AddEvent("DOM hash code changed", trace.WithAttributes(attribute.Int("oldHashCode", domHashCode), attribute.Int("newHashCode", newHashCode)))
					domHashCode = newHashCode
					initialDOMPass = true
					continue // not stable yet
				}
				span.AddEvent("DOM hash code stable", trace.WithAttributes(attribute.Int("hashCode", domHashCode)))
			}

			break // we're done!!
		}

		return nil
	})
}

func waitForDuration(d time.Duration) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		tracer := tracer(ctx)
		ctx, span := tracer.Start(ctx, "waitForDuration",
			trace.WithAttributes(attribute.Float64("duration_seconds", d.Seconds())))
		defer span.End()

		select {
		case <-time.After(d):
			span.SetStatus(codes.Ok, "wait complete")
		case <-ctx.Done():
			span.SetStatus(codes.Error, "context completed before wait finished")
			return ctx.Err()
		}
		return nil
	})
}

// observingAction returns an augmented chromedp.Action which applies observability around the action:
//   - The action has a trace span around it, which will mark and record errors if any is returned.
//   - The action duration is recorded in the MetricBrowserActionDuration histogram, labelled by the action name.
//
// This is intended for use on both our own and external actions.
func observingAction(name string, action chromedp.Action) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		tracer := tracer(ctx)
		ctx, span := tracer.Start(ctx, name)
		defer span.End()
		start := time.Now()

		err := action.Do(ctx)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			span.RecordError(err)
			return err
		}
		span.SetStatus(codes.Ok, "action completed successfully")

		MetricBrowserActionDuration.WithLabelValues(name).Observe(time.Since(start).Seconds())
		return nil
	})
}

type networkHeadersCarrier network.Headers

func (c networkHeadersCarrier) Get(key string) string {
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

func (c networkHeadersCarrier) Set(key string, value string) {
	c[key] = value
}

func (c networkHeadersCarrier) Keys() []string {
	return slices.Collect(maps.Keys(c))
}

type atomicTime struct {
	atomic.Value
}

func (at *atomicTime) Load() time.Time {
	v := at.Value.Load()
	if v == nil {
		return time.Time{}
	}
	return v.(time.Time)
}

func (at *atomicTime) Store(t time.Time) {
	at.Value.Store(t)
}

func trackProcess(browserCtx context.Context, processes *ProcessStatService) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		cdpCtx := chromedp.FromContext(ctx)
		proc := cdpCtx.Browser.Process()
		if proc == nil {
			// no process to track.
			return nil
		}

		processes.TrackProcess(browserCtx, int32(proc.Pid))
		return nil
	})
}
