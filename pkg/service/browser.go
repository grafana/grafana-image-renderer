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
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/chromedp/cdproto/browser"
	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
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
)

var ErrInvalidBrowserOption = errors.New("invalid browser option")

type BrowserService struct {
	// binary is the path to the browser's binary on the file-system.
	// It will be resolved against the `PATH`.
	binary string
	// args are the arguments passed to the browser; the binary path should not be included, that is implicitly handled.
	// The args are not passed through any kind of interpretation, so do not include quotes or similar like if run from an interactive shell.
	args []string

	// defaultRenderingOptions acts as a prototype for the options that are possible to pass in.
	defaultRenderingOptions []RenderingOption

	// log is the base logger for the service.
	log *slog.Logger
}

// NewBrowserService creates a new browser service. It is used to launch browsers and control them.
//
// The options are not validated on creation, rather on request.
func NewBrowserService(binary string, args []string, defaultRenderingOptions ...RenderingOption) *BrowserService {
	return &BrowserService{
		binary:                  binary,
		args:                    args,
		defaultRenderingOptions: defaultRenderingOptions,
		log:                     slog.With("service", "browser"),
	}
}

// GetVersion runs the binary with only a `--version` argument.
// Example output would be something like: `Brave Browser 139.1.81.131` or `Chromium 139.999.999.999`. Some browsers may include more details; do not try to parse this.
func (s *BrowserService) GetVersion(ctx context.Context) (string, error) {
	tracer := tracer(ctx)
	ctx, span := tracer.Start(ctx, "BrowserService.GetVersion")
	defer span.End()

	start := time.Now()
	version, err := exec.CommandContext(ctx, s.binary, "--version").Output()
	if err != nil {
		return "", fmt.Errorf("failed to get version of browser: %w", err)
	}
	MetricBrowserGetVersionDuration.Observe(time.Since(start).Seconds())
	return string(bytes.TrimSpace(version)), nil
}

type RenderingOption func(*renderingOptions) error

// WithGPU changes whether a GPU is used.
//
// Enabling this with no GPU installed in the system is a no-op.
// When the GPU is enabled, the GPU must be accessible from the user. This potentially includes extra infra configuration.
// When the GPU is disabled, the rendering is done by the CPU; this may require Swiftshader or similar to be installed.
func WithGPU(enabled bool) RenderingOption {
	return func(ro *renderingOptions) error {
		ro.gpu = enabled
		return nil
	}
}

// WithSandbox changes whether the Chromium sandbox is used; <https://chromium.googlesource.com/chromium/src/+/refs/heads/lkgr/docs/design/sandbox.md>.
//
// Long term, this option will be removed entirely, instead being replaced with automatic detection of capabilities.
// See also: <https://github.com/grafana/grafana-operator-experience-squad/issues/1460>
func WithSandbox(enabled bool) RenderingOption {
	return func(ro *renderingOptions) error {
		ro.sandbox = enabled
		return nil
	}
}

// WithTimeZone sets the time-zone of the browser for this request.
func WithTimeZone(loc *time.Location) RenderingOption {
	return func(ro *renderingOptions) error {
		if loc == nil {
			return fmt.Errorf("%w: time-zone location was nil", ErrInvalidBrowserOption)
		}
		if loc.String() == "" {
			return fmt.Errorf("%w: time-zone name is empty", ErrInvalidBrowserOption)
		}
		ro.timezone = loc
		return nil
	}
}

// WithCookie adds a new cookie to the browser's context.
func WithCookie(name, value, domain string) RenderingOption {
	return func(ro *renderingOptions) error {
		ro.cookies = append(ro.cookies, &network.SetCookieParams{
			Name:   name,
			Value:  value,
			Domain: domain,
		})
		return nil
	}
}

// WithHeader adds a new header sent in _all_ requests from this browser.
//
// You should be careful about using this for authentication or other sensitive information; prefer cookies.
// If you do not use cookies, the user could embed a link to their own website somewhere, which means they'd get the auth tokens!
func WithHeader(name, value string) RenderingOption {
	return func(ro *renderingOptions) error {
		if name == "" {
			return fmt.Errorf("%w: header name was empty", ErrInvalidBrowserOption)
		}
		if ro.headers == nil {
			ro.headers = make(network.Headers)
		}
		ro.headers[name] = value
		return nil
	}
}

// WithTimeout sets how long a request can take in total, from the point of starting up the browser.
//
// If a request takes longer, it is cancelled. We will not use this to render an incomplete result or similar.
// The timeout must be positive: we do not support waiting forever, but you can set a _very_ high timeout.
func WithTimeout(timeout time.Duration) RenderingOption {
	return func(ro *renderingOptions) error {
		if timeout <= 0 {
			return fmt.Errorf("%w: timeout must be positive", ErrInvalidBrowserOption)
		}
		ro.timeout = timeout
		return nil
	}
}

// WithTimeBetweenScrolls changes how long we wait for a scroll event to complete before starting a new one.
//
// We will scroll the entire web-page by the entire viewport over and over until we have seen everything.
// That means for a viewport that is 500px high, and a webpage that is 2500px high, we will scroll 5 times, meaning a total wait duration of 6 * duration (as we have to wait on the first & last scrolls as well).
// This means that for very, very large web-pages, the [WithTimeout] may also need to be changed.
func WithTimeBetweenScrolls(duration time.Duration) RenderingOption {
	return func(ro *renderingOptions) error {
		if duration <= 0 {
			return fmt.Errorf("%w: time between scrolls must be positive", ErrInvalidBrowserOption)
		}
		ro.timeBetweenScrolls = duration
		return nil
	}
}

// WithViewport sets the view of the browser: this is the size used by the actual webpage, not the browser window.
//
// A value of -1 is ignored.
// The width and height must be larger than 10 px each; usual values are 1000x500 and 1920x1080, although bigger & smaller work as well.
// You effectively set the aspect ratio with this as well: for 16:9, use a width that is 16px for every 9px it is high, or for 4:3, use a width that is 4px for every 3px it is high.
func WithViewport(width, height int) RenderingOption {
	return func(ro *renderingOptions) error {
		if width != -1 {
			if width < 10 {
				return fmt.Errorf("%w: viewport width must be at least 10px", ErrInvalidBrowserOption)
			}
			ro.viewportWidth = width
		}
		if height != -1 {
			if height < 10 {
				return fmt.Errorf("%w: viewport height must be at least 10px", ErrInvalidBrowserOption)
			}
			ro.viewportHeight = height
		}
		return nil
	}
}

// WithPageScaleFactor uses the given scale for all webpages visited by the browser.
func WithPageScaleFactor(factor float64) RenderingOption {
	return func(ro *renderingOptions) error {
		ro.pageScaleFactor = factor
		return nil
	}
}

func WithLandscape(landscape bool) RenderingOption {
	return func(ro *renderingOptions) error {
		ro.landscape = landscape
		return nil
	}
}

type renderingOptions struct {
	gpu                bool
	sandbox            bool
	timezone           *time.Location
	cookies            []*network.SetCookieParams
	headers            network.Headers
	timeout            time.Duration
	timeBetweenScrolls time.Duration
	viewportWidth      int
	viewportHeight     int
	pageScaleFactor    float64
	printer            printer
	landscape          bool
}

func defaultRenderingOptions() *renderingOptions {
	return &renderingOptions{ // set sensible defaults here; we want all values filled in to show explicit intent
		gpu:                false,                 // assume no GPU: this can be heavy, and if it exists, it likely exists for AI/ML/transcoding/... purposes, not for us
		sandbox:            false,                 // FIXME: enable this; <https://github.com/grafana/grafana-operator-experience-squad/issues/1460>
		timezone:           time.UTC,              // UTC ensures consistency when it is not configured but the users' servers are in multiple locations
		cookies:            nil,                   // no cookies by default; append handles nil fine
		headers:            nil,                   // no headers by default; WithHeader ensures nil is handled
		timeout:            30 * time.Second,      // we don't want to wait forever, so 30 seconds should be enough to get a website to be viewable
		timeBetweenScrolls: time.Millisecond * 50, // we want to wait long enough for the JS to pick up we see panels and start querying
		viewportWidth:      1000,                  // this makes a 2:1 aspect ratio
		viewportHeight:     500,
		pageScaleFactor:    1.0,                 // no scaling by default
		printer:            defaultPDFPrinter(), // print as PDF if no other format is requested
		landscape:          true,
	}
}

func (s *BrowserService) Render(ctx context.Context, url string, optionFuncs ...RenderingOption) ([]byte, string, error) {
	tracer := tracer(ctx)
	ctx, span := tracer.Start(ctx, "BrowserService.Render")
	defer span.End()
	start := time.Now()

	if url == "" {
		return nil, "text/plain", fmt.Errorf("url must not be empty")
	}

	opts := defaultRenderingOptions()
	for _, f := range s.defaultRenderingOptions {
		if err := f(opts); err != nil {
			return nil, "text/plain", fmt.Errorf("failed to apply default rendering option: %w", err)
		}
	}
	for _, f := range optionFuncs {
		if err := f(opts); err != nil {
			return nil, "text/plain", fmt.Errorf("failed to apply rendering option: %w", err)
		}
	}
	span.AddEvent("options applied")

	allocatorOptions, err := s.createAllocatorOptions(opts)
	if err != nil {
		return nil, "text/plain", fmt.Errorf("failed to create allocator options: %w", err)
	}
	timeoutCtx, cancelTimeout := context.WithTimeout(ctx, opts.timeout)
	defer cancelTimeout()
	allocatorCtx, cancelAllocator := chromedp.NewExecAllocator(timeoutCtx, allocatorOptions...)
	defer cancelAllocator()
	browserCtx, cancelBrowser := chromedp.NewContext(allocatorCtx, browserLoggers(ctx, s.log))
	defer cancelBrowser()
	span.AddEvent("browser allocated")

	s.handleNetworkEvents(browserCtx)

	orientation := chromedp.EmulatePortrait
	if opts.landscape {
		orientation = chromedp.EmulateLandscape
	}
	fileChan := make(chan []byte, 1) // buffered: we don't want the browser to stick around while we try to export this value.
	actions := []chromedp.Action{
		tracingAction("network.Enable", network.Enable()),
		tracingAction("SetPageScaleFactor", emulation.SetPageScaleFactor(opts.pageScaleFactor)),
		tracingAction("EmulateViewport", chromedp.EmulateViewport(int64(opts.viewportWidth), int64(opts.viewportHeight), orientation)),
		setHeaders(browserCtx, opts.headers),
		setCookies(opts.cookies),
		tracingAction("Navigate", chromedp.Navigate(url)),
		tracingAction("WaitReady(body)", chromedp.WaitReady("body", chromedp.ByQuery)), // wait for a body to exist; this is when the page has started to actually render
		scrollForElements(opts.timeBetweenScrolls),
		waitForDuration(time.Second),
		waitForReady(browserCtx, opts.timeout),
		resizeViewportForFullHeight(opts),      // Resize after all content is loaded and ready
		waitForReady(browserCtx, opts.timeout), // Wait for readiness again after viewport resize
		opts.printer.action(fileChan, opts),
	}
	span.AddEvent("actions created")
	if err := chromedp.Run(browserCtx, actions...); err != nil {
		return nil, "text/plain", fmt.Errorf("failed to run browser: %w", err)
	}
	span.AddEvent("actions completed")

	select {
	case fileContents := <-fileChan:
		MetricBrowserRenderDuration.Observe(time.Since(start).Seconds())
		return fileContents, opts.printer.contentType(), nil
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
func (s *BrowserService) RenderCSV(ctx context.Context, url, renderKey, domain, acceptLanguage string) ([]byte, error) {
	tracer := tracer(ctx)
	ctx, span := tracer.Start(ctx, "BrowserService.RenderCSV")
	defer span.End()
	start := time.Now()

	if url == "" {
		return nil, fmt.Errorf("url must not be empty")
	}

	allocatorOptions, err := s.createAllocatorOptions(defaultRenderingOptions())
	if err != nil {
		return nil, fmt.Errorf("failed to create allocator options: %w", err)
	}
	allocatorCtx, cancelAllocator := chromedp.NewExecAllocator(ctx, allocatorOptions...)
	defer cancelAllocator()
	browserCtx, cancelBrowser := chromedp.NewContext(allocatorCtx, browserLoggers(ctx, s.log))
	defer cancelBrowser()
	span.AddEvent("browser allocated")

	tmpDir, err := os.MkdirTemp("", "gir-csv-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary directory: %w", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			s.log.WarnContext(ctx, "failed to remove temporary directory", "path", tmpDir, "error", err)
			span.AddEvent("temporary directory removed", trace.WithAttributes(attribute.String("path", tmpDir)))
		}
	}()
	span.AddEvent("temporary directory created", trace.WithAttributes(attribute.String("path", tmpDir)))

	var headers network.Headers
	if acceptLanguage != "" {
		headers = network.Headers{
			"Accept-Language": acceptLanguage,
		}
	}

	s.handleNetworkEvents(browserCtx)

	actions := []chromedp.Action{
		tracingAction("network.Enable", network.Enable()),
		setHeaders(browserCtx, headers),
		setCookies([]*network.SetCookieParams{
			{
				Name:   "renderKey",
				Value:  renderKey,
				Domain: domain,
			},
		}),
		tracingAction("SetDownloadBehavior", browser.SetDownloadBehavior(browser.SetDownloadBehaviorBehaviorAllow).WithDownloadPath(tmpDir)),
		tracingAction("Navigate", chromedp.Navigate(url)),
	}
	if err := chromedp.Run(browserCtx, actions...); err != nil {
		return nil, fmt.Errorf("failed to run browser: %w", err)
	}
	span.AddEvent("actions completed")

	// Wait for the file to be downloaded.
	var entries []os.DirEntry
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		entries, err = os.ReadDir(tmpDir)
		if err == nil && len(entries) > 0 {
			break // file exists now
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(100 * time.Millisecond):
			// try again
		}
	}
	span.AddEvent("downloaded file located", trace.WithAttributes(attribute.String("path", filepath.Join(tmpDir, entries[0].Name()))))

	fileContents, err := os.ReadFile(filepath.Join(tmpDir, entries[0].Name()))
	if err != nil {
		return nil, fmt.Errorf("failed to read temporary file: %w", err)
	}

	MetricBrowserRenderCSVDuration.Observe(time.Since(start).Seconds())
	return fileContents, nil
}

func (s *BrowserService) createAllocatorOptions(renderingOptions *renderingOptions) ([]chromedp.ExecAllocatorOption, error) {
	opts := chromedp.DefaultExecAllocatorOptions[:]
	opts = append(opts, chromedp.Headless, chromedp.DisableGPU)              // TODO: make configurable?
	opts = append(opts, chromedp.NoFirstRun, chromedp.NoDefaultBrowserCheck) // TODO: make configurable?
	opts = append(opts, chromedp.NoSandbox)                                  // TODO: Make this configurable, so we can slowly phase it back in
	opts = append(opts, chromedp.ExecPath(s.binary))
	opts = append(opts, chromedp.WindowSize(renderingOptions.viewportWidth, renderingOptions.viewportHeight))
	for _, arg := range s.args {
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

	opts = append(opts, chromedp.Env("TZ="+renderingOptions.timezone.String()))

	return opts, nil
}

func (s *BrowserService) handleNetworkEvents(browserCtx context.Context) {
	requests := make(map[network.RequestID]trace.Span)
	mu := &sync.Mutex{}
	tracer := tracer(browserCtx)

	chromedp.ListenTarget(browserCtx, func(ev any) {
		// We MUST NOT issue new actions within this goroutine. Spawn a new one, ALWAYS.
		// See the docs of ListenTarget for more.

		switch e := ev.(type) {
		case *network.EventRequestWillBeSent:
			mu.Lock()
			defer mu.Unlock()

			_, span := tracer.Start(browserCtx, "Browser HTTP request",
				trace.WithTimestamp(e.Timestamp.Time()),
				trace.WithAttributes(
					attribute.String("requestID", string(e.RequestID)),
					attribute.String("url", e.Request.URL),
					attribute.String("method", e.Request.Method),
					attribute.String("type", string(e.Type)),
				))
			requests[e.RequestID] = span

		case *network.EventResponseReceived:
			mu.Lock()
			defer mu.Unlock()

			span, ok := requests[e.RequestID]
			if !ok {
				return
			}
			span.SetAttributes(
				attribute.Int("status", int(e.Response.Status)),
				attribute.String("statusText", e.Response.StatusText),
				attribute.String("mimeType", e.Response.MimeType),
				attribute.String("protocol", e.Response.Protocol),
				attribute.String("contentType", fmt.Sprintf("%v", e.Response.Headers["Content-Type"])),
			)
			if e.Response.Status >= 400 {
				span.SetStatus(codes.Error, e.Response.StatusText)
			} else {
				span.SetStatus(codes.Ok, e.Response.StatusText)
			}
			span.End(trace.WithTimestamp(e.Timestamp.Time()))
			delete(requests, e.RequestID) // no point keeping it around anymore.
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

type printer interface {
	action(output chan []byte, req *renderingOptions) chromedp.Action
	contentType() string
}

type pdfPrinter struct {
	paperSize       PaperSize
	printBackground bool
	pageRanges      string // empty string is all pages
}

func (p *pdfPrinter) action(dst chan []byte, req *renderingOptions) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		tracer := tracer(ctx)
		ctx, span := tracer.Start(ctx, "pdfPrinter.action",
			trace.WithAttributes(
				attribute.String("paperSize", string(p.paperSize)),
				attribute.Bool("printBackground", p.printBackground),
				attribute.Bool("landscape", req.landscape),
				attribute.Float64("pageScaleFactor", req.pageScaleFactor),
			))
		defer span.End()

		width, height, err := p.paperSize.FormatInches()
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			return fmt.Errorf("failed to get paper size dimensions: %w", err)
		}

		scale := 1.0
		if req.pageScaleFactor != 0 {
			scale = 1.0 / req.pageScaleFactor
		}

		// We don't need the stream return value; we don't ask for a stream.
		output, _, err := page.PrintToPDF().
			WithPrintBackground(p.printBackground).
			WithMarginBottom(0).
			WithMarginLeft(0).
			WithMarginRight(0).
			WithMarginTop(0).
			WithLandscape(req.landscape).
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

func defaultPDFPrinter() *pdfPrinter {
	return &pdfPrinter{
		paperSize:       PaperA4,
		printBackground: true,
	}
}

func WithPDFPrinter(opts ...PDFPrinterOption) RenderingOption {
	return func(ro *renderingOptions) error {
		pp := defaultPDFPrinter()
		for _, f := range opts {
			if err := f(pp); err != nil {
				return fmt.Errorf("failed to apply PDF printer option: %w", err)
			}
		}
		ro.printer = pp
		return nil
	}
}

type pngPrinter struct {
	fullHeight bool
}

func (p *pngPrinter) action(dst chan []byte, opts *renderingOptions) chromedp.Action {
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

func WithPNGPrinter(opts ...PNGPrinterOption) RenderingOption {
	return func(ro *renderingOptions) error {
		pp := &pngPrinter{
			fullHeight: false,
		}
		for _, f := range opts {
			if err := f(pp); err != nil {
				return fmt.Errorf("failed to apply PNG printer option: %w", err)
			}
		}
		ro.printer = pp
		return nil
	}
}

func setHeaders(browserCtx context.Context, headers network.Headers) chromedp.Action {
	if sc := trace.SpanFromContext(browserCtx); sc != nil && sc.IsRecording() {
		if headers == nil {
			headers = make(network.Headers)
		}
		otel.GetTextMapPropagator().Inject(browserCtx, networkHeadersCarrier(headers))
	}

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

func resizeViewportForFullHeight(opts *renderingOptions) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		// Only resize for PNG printers with fullHeight enabled
		pngPrinter, ok := opts.printer.(*pngPrinter)
		if !ok || !pngPrinter.fullHeight {
			return nil // Skip for non-PNG or non-fullHeight screenshots
		}

		tracer := tracer(ctx)
		ctx, span := tracer.Start(ctx, "resizeViewportForFullHeight")
		defer span.End()

		var scrollHeight int
		err := chromedp.Evaluate(`document.body.scrollHeight`, &scrollHeight).Do(ctx)
		if err != nil {
			span.SetStatus(codes.Error, "failed to get scroll height: "+err.Error())
			return fmt.Errorf("failed to get scroll height: %w", err)
		}

		// Only resize if the page is actually taller than the current viewport
		if scrollHeight > opts.viewportHeight {
			span.AddEvent("resizing viewport for full height capture",
				trace.WithAttributes(
					attribute.Int("originalHeight", opts.viewportHeight),
					attribute.Int("newHeight", scrollHeight),
				))

			// Determine orientation from options
			orientation := chromedp.EmulatePortrait
			if opts.landscape {
				orientation = chromedp.EmulateLandscape
			}

			err = chromedp.EmulateViewport(int64(opts.viewportWidth), int64(scrollHeight), orientation).Do(ctx)
			if err != nil {
				span.SetStatus(codes.Error, "failed to resize viewport: "+err.Error())
				return fmt.Errorf("failed to resize viewport for full height: %w", err)
			}

			span.SetStatus(codes.Ok, "viewport resized successfully")
		} else {
			span.AddEvent("no viewport resize needed", trace.WithAttributes(attribute.Int("pageHeight", scrollHeight)))
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

func waitForReady(browserCtx context.Context, timeout time.Duration) chromedp.Action {
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
	chromedp.ListenTarget(browserCtx, func(ev any) {
		switch ev.(type) {
		case *network.EventRequestWillBeSent:
			requests.Add(1)
			lastRequest.Store(time.Now())
		case *network.EventLoadingFinished, *network.EventLoadingFailed:
			requests.Add(-1)
		}
	})

	return chromedp.ActionFunc(func(ctx context.Context) error {
		tracer := tracer(ctx)
		ctx, span := tracer.Start(ctx, "waitForReady",
			trace.WithAttributes(attribute.Float64("timeout_seconds", timeout.Seconds())))
		defer span.End()

		timeout := time.After(timeout)

		hasHadQueries := false
		giveUpFirstQuery := time.Now().Add(time.Second * 3)

		var domHashCode int
		initialDOMPass := true

		for {
			select {
			case <-ctx.Done():
				span.SetStatus(codes.Error, "context completed before readiness detected")
				return ctx.Err()
			case <-timeout:
				span.SetStatus(codes.Error, "timed out waiting for readiness")
				return fmt.Errorf("timed out waiting for readiness")
			case <-time.After(100 * time.Millisecond):
			}

			if requests.Load() > 0 {
				initialDOMPass = true
				span.AddEvent("network requests still ongoing", trace.WithAttributes(attribute.Int64("inflightRequests", requests.Load())))
				continue // still waiting on network requests to complete
			}

			running, err := getRunningQueries(ctx)
			if err != nil {
				span.SetStatus(codes.Error, err.Error())
				return fmt.Errorf("failed to get running queries: %w", err)
			}
			span.AddEvent("queried running queries", trace.WithAttributes(attribute.Bool("running", running)))
			if running {
				initialDOMPass = true
				hasHadQueries = true
				continue // still waiting on queries to complete
			} else if !hasHadQueries && time.Now().Before(giveUpFirstQuery) {
				span.AddEvent("no first query detected yet; giving it more time")
				continue
			}

			if initialDOMPass {
				domHashCode, err = getDOMHashCode(ctx)
				if err != nil {
					span.SetStatus(codes.Error, err.Error())
					return fmt.Errorf("failed to get DOM hash code: %w", err)
				}
				span.AddEvent("initial DOM hash code recorded", trace.WithAttributes(attribute.Int("hashCode", domHashCode)))
				initialDOMPass = false
				continue // not stable yet
			}

			newHashCode, err := getDOMHashCode(ctx)
			if err != nil {
				span.SetStatus(codes.Error, err.Error())
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

func tracingAction(name string, action chromedp.Action) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		tracer := tracer(ctx)
		ctx, span := tracer.Start(ctx, name)
		defer span.End()
		start := time.Now()

		err := action.Do(ctx)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
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
