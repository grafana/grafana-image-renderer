package service

import (
	"bytes"
	"context"
	"encoding"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/grafana/grafana-image-renderer/pkg/config"
	"github.com/grafana/grafana-image-renderer/pkg/service/browser"
	"github.com/prometheus/client_golang/prometheus"
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
)

var ErrInvalidBrowserOption = errors.New("invalid browser option")

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
	return func(cfg config.BrowserConfig) (config.BrowserConfig, error) {
		cfg.Cookies = append(cfg.Cookies, config.Cookie{
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
			cfg.Headers = make(map[string]string)
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
		if width != -1 {
			if width < 100 {
				return cfg, fmt.Errorf("%w: viewport width must be at least 100px", ErrInvalidBrowserOption)
			}
			cfg.MinWidth = clamped(width, cfg.MinWidth, cfg.MaxWidth)
		}
		if height != -1 {
			if height < 100 {
				return cfg, fmt.Errorf("%w: viewport height must be at least 100px", ErrInvalidBrowserOption)
			}
			cfg.MinHeight = clamped(height, cfg.MinHeight, cfg.MaxHeight)
		}
		return cfg, nil
	}
}

// WithPageScaleFactor uses the given scale for all webpages visited by the browser.
func WithPageScaleFactor(factor float64) RenderingOption {
	return func(cfg config.BrowserConfig) (config.BrowserConfig, error) {
		if factor <= 0 {
			return cfg, fmt.Errorf("%w: page scale factor must be positive", ErrInvalidBrowserOption)
		}
		cfg.PageScaleFactor = factor
		return cfg, nil
	}
}

func WithLandscape(landscape bool) RenderingOption {
	return func(cfg config.BrowserConfig) (config.BrowserConfig, error) {
		cfg.Landscape = landscape
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

	fileChan := make(chan []byte, 1) // buffered: we don't want the browser to stick around while we try to export this value.
	defer close(fileChan)
	MetricBrowserInstancesActive.Inc()
	defer MetricBrowserInstancesActive.Dec()
	err := browser.WithChromium(ctx, cfg,
		observingAction("trackProcess", trackProcess(s.processes)),
		observingAction("setPageScaleFactor", setPageScaleFactor(cfg.PageScaleFactor)),
		observingAction("setViewPort", setViewPort(cfg.MinWidth, cfg.MinHeight, cfg.Landscape)),
		observingAction("setHeaders", setHeaders(cfg.Headers)),
		observingAction("setCookies", setCookies(cfg.Cookies)),
		observingAction("navigate", navigate(url)),
		observingAction("scrollForElements", scrollForElements(cfg.TimeBetweenScrolls)),
		observingAction("waitForDuration", waitForDuration(cfg.ReadinessPriorWait)),
		observingAction("waitForReady", waitForReady(cfg)),
		observingAction("printer.prepare", printer.prepare(cfg)),
		observingAction("printer.print", printer.action(fileChan, cfg)),
	)
	if err != nil {
		return nil, "text/plain", fmt.Errorf("browser failure: %w", err)
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

	tmpDir, err := os.MkdirTemp("", "gir-csv-*")
	if err != nil {
		return nil, "", fmt.Errorf("failed to create temporary directory: %w", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			s.log.WarnContext(ctx, "failed to remove temporary directory", "path", tmpDir, "error", err)
			span.AddEvent("temporary directory removed", trace.WithAttributes(attribute.String("path", tmpDir)))
		}
	}()
	span.AddEvent("temporary directory created", trace.WithAttributes(attribute.String("path", tmpDir)))

	var headers map[string]string
	if acceptLanguage != "" {
		headers = map[string]string{
			"Accept-Language": acceptLanguage,
		}
	}

	csvPath := make(chan string, 1)
	defer close(csvPath)
	err = browser.WithChromium(ctx, s.cfg,
		observingAction("trackProcess", trackProcess(s.processes)),
		observingAction("setHeaders", setHeaders(headers)),
		observingAction("setCookies", setCookies([]config.Cookie{
			{
				Name:   "renderKey",
				Value:  renderKey,
				Domain: domain,
			},
		})),
		observingAction("setDownloadsDir", setDownloadsDir(tmpDir)),
		observingAction("navigate", navigate(url)),
		observingAction("awaitDownloadedCSV", awaitDownloadedCSV(tmpDir, csvPath)))
	if err != nil {
		return nil, "", fmt.Errorf("browser failure: %w", err)
	}
	span.AddEvent("actions completed")

	select {
	case fp := <-csvPath:
		fileContents, err := os.ReadFile(fp)
		if err != nil {
			return nil, "", fmt.Errorf("failed to read temporary file %q: %w", fp, err)
		}

		MetricBrowserRenderCSVDuration.Observe(time.Since(start).Seconds())
		return fileContents, filepath.Base(fp), nil

	default:
		// invariant violated: we should have gotten a value from the awaitDownloadedCSV action, OR returned an error from WithChromium.
		return nil, "", fmt.Errorf("no CSV file was downloaded")
	}
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
	prepare(cfg config.BrowserConfig) browser.Action
	action(output chan<- []byte, cfg config.BrowserConfig) browser.Action
	contentType() string
}

type pdfPrinter struct {
	paperSize       PaperSize
	printBackground bool
	pageRanges      string // empty string is all pages
}

func (p *pdfPrinter) prepare(_ config.BrowserConfig) browser.Action {
	return func(context.Context, browser.Browser) error {
		return nil
	}
}

func (p *pdfPrinter) action(dst chan<- []byte, cfg config.BrowserConfig) browser.Action {
	return func(ctx context.Context, b browser.Browser) error {
		width, height, err := p.paperSize.FormatInches()
		if err != nil {
			return fmt.Errorf("failed to get paper size dimensions: %w", err)
		}

		scale := 1.0
		if cfg.PageScaleFactor != 0 {
			scale = 1.0 / cfg.PageScaleFactor
		}

		data, err := b.PrintPDF(ctx, browser.PDFOptions{
			IncludeBackground: p.printBackground,
			Landscape:         cfg.Landscape,
			PaperWidth:        width,
			PaperHeight:       height,
			Scale:             scale,
			PageRanges:        p.pageRanges,
		})
		if err != nil {
			return fmt.Errorf("failed to print page to PDF: %w", err)
		}
		dst <- data
		return nil
	}
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

func (p *pngPrinter) prepare(cfg config.BrowserConfig) browser.Action {
	return func(ctx context.Context, b browser.Browser) error {
		if !p.fullHeight {
			return nil
		}

		tracer := tracer(ctx)
		ctx, span := tracer.Start(ctx, "pngPrinter.prepare",
			trace.WithAttributes(
				attribute.Int("currentViewportWidth", cfg.MinWidth),
				attribute.Int("currentViewportHeight", cfg.MinHeight),
				attribute.Bool("landscape", cfg.Landscape),
			))
		defer span.End()

		scrollHeight, err := b.EvaluateToInt(ctx, "document.body.scrollHeight")
		if err != nil {
			span.SetStatus(codes.Error, "failed to get scroll height: "+err.Error())
			return fmt.Errorf("failed to get scroll height: %w", err)
		}
		span.AddEvent("obtained scroll height", trace.WithAttributes(attribute.Int("scrollHeight", scrollHeight)))

		if scrollHeight > cfg.MinHeight {
			span.AddEvent("resizing viewport for full height capture",
				trace.WithAttributes(
					attribute.Int("originalHeight", cfg.MinHeight),
					attribute.Int("newHeight", scrollHeight),
				))

			orientation := browser.OrientationPortrait
			if cfg.Landscape {
				orientation = browser.OrientationLandscape
			}

			err := b.SetViewPort(ctx, cfg.MinWidth, scrollHeight, orientation)
			if err != nil {
				return fmt.Errorf("failed to resize viewport for full height: %w", err)
			}

			start := time.Now()
			if err := waitForStableDOM(cfg, start, ctx, b); err != nil {
				return fmt.Errorf("failed to wait for stable DOM after resizing viewport: %w", err)
			}
		} else {
			span.AddEvent("no viewport resize needed")
		}

		return nil
	}
}

func (p *pngPrinter) action(dst chan<- []byte, cfg config.BrowserConfig) browser.Action {
	return func(ctx context.Context, b browser.Browser) error {
		data, err := b.PrintPNG(ctx)
		if err != nil {
			return fmt.Errorf("failed to print page to PNG: %w", err)
		}
		dst <- data
		return nil
	}
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

func observingAction(name string, action browser.Action) browser.Action {
	return func(ctx context.Context, browser browser.Browser) error {
		tracer := tracer(ctx)
		ctx, span := tracer.Start(ctx, name)
		defer span.End()
		start := time.Now()

		err := action(ctx, browser)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			span.RecordError(err)
			return err
		}
		span.SetStatus(codes.Ok, "action completed successfully")

		MetricBrowserActionDuration.WithLabelValues(name).Observe(time.Since(start).Seconds())
		return nil
	}
}

func trackProcess(processes *ProcessStatService) browser.Action {
	return func(ctx context.Context, browser browser.Browser) error {
		pid, err := browser.GetPID(ctx)
		if err != nil {
			return fmt.Errorf("failed to get browser process PID: %w", err)
		}

		processes.TrackProcess(ctx, pid)
		return nil
	}
}

func setPageScaleFactor(factor float64) browser.Action {
	return func(ctx context.Context, b browser.Browser) error {
		return b.SetPageScale(ctx, factor)
	}
}

func setViewPort(width, height int, landscape bool) browser.Action {
	return func(ctx context.Context, b browser.Browser) error {
		orientation := browser.OrientationPortrait
		if landscape {
			orientation = browser.OrientationLandscape
		}
		return b.SetViewPort(ctx, width, height, orientation)
	}
}

func setHeaders(headers map[string]string) browser.Action {
	return func(ctx context.Context, b browser.Browser) error {
		return b.SetExtraHeaders(ctx, headers)
	}
}

func setCookies(cookies []config.Cookie) browser.Action {
	return func(ctx context.Context, b browser.Browser) error {
		for _, cookie := range cookies {
			if err := b.SetCookie(ctx, cookie); err != nil {
				return fmt.Errorf("failed to set cookie %q: %w", cookie.Name, err)
			}
		}
		return nil
	}
}

func navigate(url string) browser.Action {
	return func(ctx context.Context, b browser.Browser) error {
		return b.NavigateAndWait(ctx, url)
	}
}

func scrollForElements(timeBetweenScrolls time.Duration) browser.Action {
	return func(ctx context.Context, b browser.Browser) error {
		tracer := tracer(ctx)
		ctx, span := tracer.Start(ctx, "scrollForElements")
		defer span.End()

		scrolls, err := b.EvaluateToInt(ctx, `Math.floor(document.body.scrollHeight / window.innerHeight)`)
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
			err := b.Evaluate(ctx, `window.scrollBy(0, window.innerHeight, { behavior: 'instant' })`)
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

		err = b.Evaluate(ctx, `window.scrollTo(0, 0, { behavior: 'instant' })`)
		if err != nil {
			return fmt.Errorf("failed to scroll to top: %w", err)
		}
		span.AddEvent("scrolled to top")
		return nil
	}
}

func waitForDuration(d time.Duration) browser.Action {
	return func(ctx context.Context, b browser.Browser) error {
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
	}
}

func waitForNetworkIdle(cfg config.BrowserConfig, start time.Time, ctx context.Context, b browser.Browser) (sawRequests bool, err error) {
	tracer := tracer(ctx)
	ctx, span := tracer.Start(ctx, "waitForNetworkIdle",
		trace.WithAttributes(
			attribute.String("interval", cfg.ReadinessIterationInterval.String()),
			attribute.String("start_time", start.String()),
			attribute.Bool("disabled", cfg.ReadinessDisableNetworkWait),
			attribute.String("timeout", cfg.ReadinessNetworkIdleTimeout.String())))
	defer span.End()

	if cfg.ReadinessDisableNetworkWait {
		span.AddEvent("network wait disabled; skipping")
		return false, nil
	}

	sawRequests = false
	for {
		if cfg.ReadinessNetworkIdleTimeout > 0 && time.Since(start) >= cfg.ReadinessNetworkIdleTimeout {
			span.AddEvent("network idle wait timed out")
			break
		}
		select {
		case <-ctx.Done():
			return sawRequests, ctx.Err()
		case <-time.After(cfg.ReadinessIterationInterval):
		}

		currentRequests, err := b.GetCurrentNetworkRequests(ctx)
		if err != nil {
			return sawRequests, fmt.Errorf("failed to get current network requests: %w", err)
		}

		if currentRequests > 0 {
			span.AddEvent("network requests still ongoing", trace.WithAttributes(attribute.Int("inflight_requests", currentRequests)))
			sawRequests = true
			continue // still waiting on network requests to complete
		}

		break
	}

	return sawRequests, nil
}

func waitForQueries(cfg config.BrowserConfig, start time.Time, ctx context.Context, b browser.Browser) (sawQueries bool, err error) {
	tracer := tracer(ctx)
	ctx, span := tracer.Start(ctx, "waitForQueries",
		trace.WithAttributes(
			attribute.String("interval", cfg.ReadinessIterationInterval.String()),
			attribute.String("start_time", start.String()),
			attribute.Bool("disabled", cfg.ReadinessDisableQueryWait),
			attribute.String("timeout", cfg.ReadinessQueriesTimeout.String()),
			attribute.String("first_query_timeout", cfg.ReadinessFirstQueryTimeout.String())))
	defer span.End()

	if cfg.ReadinessDisableQueryWait {
		span.AddEvent("query wait disabled; skipping")
		return false, nil
	}

	sawQueries = false
	for {
		if cfg.ReadinessQueriesTimeout > 0 && time.Since(start) >= cfg.ReadinessQueriesTimeout {
			span.AddEvent("query wait timed out")
			break
		}
		if !sawQueries && cfg.ReadinessFirstQueryTimeout > 0 && time.Since(start) >= cfg.ReadinessFirstQueryTimeout {
			span.AddEvent("first query wait timed out")
			break
		}
		select {
		case <-ctx.Done():
			return sawQueries, ctx.Err()
		case <-time.After(cfg.ReadinessIterationInterval):
		}

		isRunningQueries := false
		{
			// hide the ugly int in a new scope
			running, err := b.EvaluateToInt(ctx, `(!!(window.__grafanaSceneContext && window.__grafanaRunningQueryCount > 0)) ? 1 : 0`)
			if err != nil {
				return sawQueries, fmt.Errorf("failed to get running queries: %w", err)
			}
			isRunningQueries = running != 0
		}

		if !isRunningQueries {
			span.AddEvent("no running queries detected")
			if sawQueries {
				break
			}
		} else {
			sawQueries = true
		}
	}

	return sawQueries, nil
}

func waitForStableDOM(cfg config.BrowserConfig, start time.Time, ctx context.Context, b browser.Browser) error {
	tracer := tracer(ctx)
	ctx, span := tracer.Start(ctx, "waitForStableDOM",
		trace.WithAttributes(
			attribute.String("interval", cfg.ReadinessIterationInterval.String()),
			attribute.String("start_time", start.String()),
			attribute.Bool("disabled", cfg.ReadinessDisableDOMHashCodeWait),
			attribute.String("timeout", cfg.ReadinessDOMHashCodeTimeout.String())))
	defer span.End()

	if cfg.ReadinessDisableDOMHashCodeWait {
		span.AddEvent("DOM hash code wait disabled; skipping")
		return nil
	}

	initialDOMPass := true
	previousHashCode := 0
	for {
		if cfg.ReadinessDOMHashCodeTimeout > 0 && time.Since(start) >= cfg.ReadinessDOMHashCodeTimeout {
			span.AddEvent("DOM hash code wait timed out")
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(cfg.ReadinessIterationInterval):
		}

		newHashCode, err := b.EvaluateToInt(ctx, `((x) => {
			let h = 0;
			for (let i = 0; i < x.length; i++) {
				h = (Math.imul(31, h) + x.charCodeAt(i)) | 0;
			}
			return h;
		})(document.body.toString())`)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			span.RecordError(err)
			return fmt.Errorf("failed to get DOM hash code: %w", err)
		}

		if initialDOMPass {
			span.AddEvent("initial DOM hash code recorded", trace.WithAttributes(attribute.Int("hashCode", newHashCode)))
			initialDOMPass = false
			previousHashCode = newHashCode
			continue
		}

		span.AddEvent("subsequent DOM hash code recorded", trace.WithAttributes(attribute.Int("hashCode", newHashCode)))
		if newHashCode != previousHashCode {
			span.AddEvent("DOM hash code changed", trace.WithAttributes(
				attribute.Int("oldHashCode", previousHashCode),
				attribute.Int("newHashCode", newHashCode)))
			previousHashCode = newHashCode
			initialDOMPass = true
			continue // not stable yet
		}

		span.SetStatus(codes.Ok, "DOM hash code stable")
		span.AddEvent("DOM hash code stable", trace.WithAttributes(attribute.Int("hashCode", newHashCode)))
		return nil
	}
}

func waitForReady(cfg config.BrowserConfig) browser.Action {
	return func(ctx context.Context, b browser.Browser) error {
		start := time.Now()

		tracer := tracer(ctx)
		ctx, span := tracer.Start(ctx, "waitForReady",
			trace.WithAttributes(attribute.String("timeout", cfg.ReadinessTimeout.String())),
			trace.WithTimestamp(start))
		defer span.End()

		var readinessTimeout <-chan time.Time
		if cfg.ReadinessTimeout > 0 {
			readinessTimeout = time.After(cfg.ReadinessTimeout)
		}

		for {
			select {
			case <-ctx.Done():
				span.SetStatus(codes.Error, "context completed before readiness detected")
				return ctx.Err()
			case <-readinessTimeout:
				span.SetStatus(codes.Error, "timed out waiting for readiness")
				return fmt.Errorf("timed out waiting for readiness")

			case <-time.After(cfg.ReadinessIterationInterval):
				// Continue with the rest of the code; this is waiting for the next time we can do work.
			}

			sawRequests, err := waitForNetworkIdle(cfg, start, ctx, b)
			if err != nil {
				return fmt.Errorf("failed to wait for network idle: %w", err)
			}
			if sawRequests && (cfg.ReadinessNetworkIdleTimeout <= 0 || time.Since(start) < cfg.ReadinessNetworkIdleTimeout) {
				span.AddEvent("continuing wait after network activity detected")
				continue
			}

			sawQueries, err := waitForQueries(cfg, start, ctx, b)
			if err != nil {
				return fmt.Errorf("failed to wait for queries: %w", err)
			}
			if sawQueries && (cfg.ReadinessQueriesTimeout <= 0 || time.Since(start) < cfg.ReadinessQueriesTimeout) {
				span.AddEvent("continuing wait after queries detected")
				continue
			}

			if err := waitForStableDOM(cfg, start, ctx, b); err != nil {
				return fmt.Errorf("failed to wait for stable DOM: %w", err)
			}

			return nil
		}
	}
}

func setDownloadsDir(path string) browser.Action {
	return func(ctx context.Context, b browser.Browser) error {
		return b.PutDownloadsIn(ctx, path)
	}
}

func awaitDownloadedCSV(downloadsDir string, foundFilePath chan<- string) browser.Action {
	return func(ctx context.Context, b browser.Browser) error {
		for {
			if ctx.Err() != nil {
				return ctx.Err()
			}

			entries, err := os.ReadDir(downloadsDir)
			if err != nil {
				return fmt.Errorf("failed to read downloads directory: %w", err)
			}

			for _, entry := range entries {
				if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".csv") {
					continue // uninteresting to us
				}

				foundFilePath <- filepath.Join(downloadsDir, entry.Name())
				return nil
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(100 * time.Millisecond): // TODO: Make this configurable?
			}
		}
	}
}
