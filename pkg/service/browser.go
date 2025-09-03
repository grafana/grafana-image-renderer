package service

import (
	"bytes"
	"context"
	"encoding"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"time"

	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/google/uuid"
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
	version, err := exec.CommandContext(ctx, s.binary, "--version").Output()
	if err != nil {
		return "", fmt.Errorf("failed to get version of browser: %w", err)
	}
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

func (s *BrowserService) Render(ctx context.Context, url string, optionFuncs ...RenderingOption) ([]byte, string, error) {
	if url == "" {
		return nil, "text/plain", fmt.Errorf("url must not be empty")
	}

	opts := &renderingOptions{ // set sensible defaults here; we want all values filled in to show explicit intent
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

	traceID, err := getTraceID(ctx)
	if err != nil {
		return nil, "text/plain", fmt.Errorf("failed to get trace ID: %w", err)
	}
	log := s.log.With("trace_id", traceID)

	allocatorOptions, err := s.createAllocatorOptions(opts)
	if err != nil {
		return nil, "text/plain", fmt.Errorf("failed to create allocator options: %w", err)
	}
	timeoutCtx, cancelTimeout := context.WithTimeout(ctx, opts.timeout)
	defer cancelTimeout()
	allocatorCtx, cancelAllocator := chromedp.NewExecAllocator(timeoutCtx, allocatorOptions...)
	defer cancelAllocator()
	browserCtx, cancelBrowser := chromedp.NewContext(allocatorCtx, browserLoggers(ctx, log))
	defer cancelBrowser()

	orientation := chromedp.EmulatePortrait
	if opts.landscape {
		orientation = chromedp.EmulateLandscape
	}
	fileChan := make(chan []byte, 1) // buffered: we don't want the browser to stick around while we try to export this value.
	actions := []chromedp.Action{
		emulation.SetPageScaleFactor(opts.pageScaleFactor),
		chromedp.EmulateViewport(int64(opts.viewportWidth), int64(opts.viewportHeight), orientation),
		setHeaders(opts.headers),
		setCookies(opts.cookies),
		chromedp.Navigate(url),
		chromedp.WaitReady("body", chromedp.ByQuery), // wait for a body to exist; this is when the page has started to actually render
		scrollForElements(opts.timeBetweenScrolls),
		waitForViz(),
		waitForDuration(time.Second),
		opts.printer.action(fileChan, opts),
	}
	if err := chromedp.Run(browserCtx, actions...); err != nil {
		return nil, "text/plain", fmt.Errorf("failed to run browser: %w", err)
	}

	select {
	case fileContents := <-fileChan:
		return fileContents, opts.printer.contentType(), nil
	default:
		return nil, "text/plain", fmt.Errorf("failed to render: no data received after browser quit")
	}
}

func getTraceID(context.Context) (string, error) {
	// TODO: Use OTEL trace ID from context
	id, err := uuid.NewRandom()
	if err != nil {
		return "", fmt.Errorf("failed to generate new UUID: %w", err)
	}
	return id.String(), nil
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
		equals := strings.Index(arg, "=")
		if equals == -1 {
			opts = append(opts, chromedp.Flag(arg, ""))
		} else {
			opts = append(opts, chromedp.Flag(arg[:equals], arg[equals+1:]))
		}
	}

	opts = append(opts, chromedp.Env("TZ="+renderingOptions.timezone.String()))

	return opts, nil
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
		return 33.1102, 46.811, nil
	case PaperA1:
		return 23.3858, 33.1102, nil
	case PaperA2:
		return 16.5354, 23.3858, nil
	case PaperA3:
		return 11.6929, 16.5354, nil
	case PaperA4:
		return 8.2677, 11.6929, nil
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
}

func (p *pdfPrinter) action(dst chan []byte, req *renderingOptions) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		width, height, err := p.paperSize.FormatInches()
		if err != nil {
			return fmt.Errorf("failed to get paper size dimensions: %w", err)
		}

		scale := 1.0
		if req.pageScaleFactor != 0 {
			scale = 1.0 / req.pageScaleFactor
		}

		// We don't need the stream return value; we don't ask for a stream.
		output, _, err := page.PrintToPDF().
			WithPrintBackground(p.printBackground).
			WithLandscape(req.landscape).
			WithPaperWidth(width).
			WithPaperHeight(height).
			WithScale(scale).
			Do(ctx)
		if err != nil {
			return fmt.Errorf("failed to print to PDF: %w", err)
		}
		dst <- output
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

func (p *pngPrinter) action(dst chan []byte, _ *renderingOptions) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		output, err := page.CaptureScreenshot().
			WithFormat(page.CaptureScreenshotFormatPng).
			WithCaptureBeyondViewport(p.fullHeight).
			Do(ctx)
		if err != nil {
			return fmt.Errorf("failed to capture screenshot: %w", err)
		}
		dst <- output
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

func setHeaders(headers network.Headers) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		if len(headers) == 0 {
			return nil
		}
		return network.SetExtraHTTPHeaders(headers).Do(ctx)
	})
}

func setCookies(cookies []*network.SetCookieParams) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		for _, cookie := range cookies {
			if err := cookie.Do(ctx); err != nil {
				return fmt.Errorf("failed to set cookie %q: %w", cookie.Name, err)
			}
		}
		return nil
	})
}

func scrollForElements(timeBetweenScrolls time.Duration) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		var scrolls int
		err := chromedp.Evaluate(`Math.floor(document.body.scrollHeight / window.innerHeight)`, &scrolls).Do(ctx)
		if err != nil {
			return fmt.Errorf("failed to calculate scrolls required: %w", err)
		}

		select {
		case <-time.After(timeBetweenScrolls):
		case <-ctx.Done():
			return ctx.Err()
		}
		for range scrolls {
			err := chromedp.Evaluate(`window.scrollBy(0, window.innerHeight, { behavior: 'instant' })`, nil).Do(ctx)
			if err != nil {
				return fmt.Errorf("failed to scroll: %w", err)
			}
			select {
			case <-time.After(timeBetweenScrolls):
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		err = chromedp.Evaluate(`window.scrollTo(0, 0, { behavior: 'instant' })`, nil).Do(ctx)
		if err != nil {
			return fmt.Errorf("failed to scroll to top: %w", err)
		}

		return nil
	})
}

func waitForViz() chromedp.Action {
	const script = `(() => {
		return !window.__grafanaSceneContext || window.__grafanaRunningQueryCount === 0;
	})()`
	return chromedp.Poll(script, nil, chromedp.WithPollingMutation(), chromedp.WithPollingTimeout(0))
}

func waitForDuration(d time.Duration) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		select {
		case <-time.After(d):
		case <-ctx.Done():
			return ctx.Err()
		}
		return nil
	})
}
