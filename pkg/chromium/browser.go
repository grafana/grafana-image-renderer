package chromium

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/google/uuid"
)

type Browser struct {
	// Binary is a path to the browser's binary on the file-system.
	Binary string
	// Args is the arguments to give to the browser.
	Args []string

	// Width is the width of the viewport in pixels.
	Width int
	// Height is the height of the viewport in pixels.
	Height int
	// TimeZone is the IANA time-zone name, e.g. `Etc/UTC`.
	TimeZone string

	// PaperSize is how large a PDF should be per page.
	PaperSize PaperSize
	// Landscape defines which orientation PDFs should be in: vertical (default) or landscape (aka horizontal).
	Landscape bool
	// PrintBackground defines whether to include background graphics, such as div backgrounds or dark page backgrounds.
	// You generally want this to preserve contrasts, but it is possibly wasteful if you intend on printing the document to physical paper later on.
	PrintBackground bool

	// RenderTimeout is the duration of time we'll wait on render requests to complete.
	RenderTimeout time.Duration
}

type browserOption func(*Browser) error

func WithViewport(width, height int) browserOption {
	return func(b *Browser) error {
		b.Width = width
		b.Height = height
		return nil
	}
}

func WithTimeZone(timeZone string) browserOption {
	return func(b *Browser) error {
		b.TimeZone = timeZone
		return nil
	}
}

func NewBrowser(
	binary string,
	args []string,
	options ...browserOption,
) (*Browser, error) {
	browser := &Browser{
		Binary: binary,
		Args:   args,

		Width:    1920,
		Height:   1080,
		TimeZone: "Etc/UTC",

		PaperSize:       PaperA4,
		Landscape:       false,
		PrintBackground: true,

		RenderTimeout: time.Second * 30,
	}
	for _, opt := range options {
		if err := opt(browser); err != nil {
			return nil, err
		}
	}
	return browser, nil
}

// GetVersion finds the version of the browser.
func (b *Browser) GetVersion(ctx context.Context) (string, error) {
	version, err := exec.CommandContext(ctx, b.Binary, "--version").Output()
	if err != nil {
		return "", fmt.Errorf("failed to get version of browser: %w", err)
	}
	return string(bytes.TrimSpace(version)), nil
}

type requestOptions struct {
	timeZone        string
	width, height   int
	fullHeight      bool
	paperSize       PaperSize
	landscape       bool
	printBackground bool
	timeout         time.Duration
}

type requestOption func(*requestOptions) error

// OverridingTimeZone sets the time zone for the request.
//
// This overrides the time-zone used by the application-level browser configuration.
// The time-zone is expected to be in the format of e.g. `Etc/UTC`; if it is not, it might just not apply or break entirely.
func OverridingTimeZone(timeZone string) requestOption {
	return func(opts *requestOptions) error {
		opts.timeZone = timeZone
		return nil
	}
}

// OverridingViewport changes the viewport of the browser itself.
//
// If either value is less than or equal to zero, it will not be changed.
func OverridingViewport(width, height int) requestOption {
	return func(opts *requestOptions) error {
		if width > 0 {
			opts.width = width
		}
		if height > 0 {
			opts.height = height
		}
		return nil
	}
}

// OverridingFullHeight changes whether the _entire_ web-page will be included in the render.
//
// When used, it will override the viewport height to be 75% of the width.
// This should only be used for image renders; PDFs don't have a concept of this.
func OverridingFullHeight(fullHeight bool) requestOption {
	return func(ro *requestOptions) error {
		ro.fullHeight = fullHeight
		if fullHeight {
			ro.height = int(float64(ro.width) * 0.75)
		}
		return nil
	}
}

// OverridingPaperSize changes the size of paper intended in the PDF export.
//
// This is a no-op on image renders.
func OverridingPaperSize(paperSize PaperSize) requestOption {
	return func(ro *requestOptions) error {
		ro.paperSize = paperSize
		return nil
	}
}

// OverridingPrintBackground defines whether to include background graphics, such as div backgrounds or dark page backgrounds.
// You generally want this to preserve contrasts, but it is possibly wasteful if you intend on printing the document to physical paper later on.
func OverridingPrintBackground(printBackground bool) requestOption {
	return func(ro *requestOptions) error {
		ro.printBackground = printBackground
		return nil
	}
}

// OverridingRenderTimeout sets the timeout for the render request; after this time, the browser will be killed and an error returned.
func OverridingRenderTimeout(timeout time.Duration) requestOption {
	return func(ro *requestOptions) error {
		if timeout > 0 {
			ro.timeout = timeout
		}
		return nil
	}
}

// RenderPDF visits the website and waits for it to be fully rendered.
//
// The entire PDF is returned as a byte slice.
func (b *Browser) RenderPDF(ctx context.Context, url string, reqOpts ...requestOption) ([]byte, error) {
	browserID, err := uuid.NewRandom() // TODO: Use traceID if exists in context
	if err != nil {
		return nil, fmt.Errorf("failed to generate browser ID: %w", err)
	}
	log := slog.With("browser_id", browserID.String())

	allocatorOptions, requestOptions, err := b.createAllocatorOptions(reqOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to create allocator options: %w", err)
	}
	allocatorCtx, cancelAllocator := chromedp.NewExecAllocator(ctx, allocatorOptions...)
	defer cancelAllocator()
	browserCtx, cancelBrowser := chromedp.NewContext(allocatorCtx, browserLoggers(ctx, log))
	defer cancelBrowser()

	fileChan := make(chan []byte, 1)
	actions := []chromedp.Action{
		chromedp.Navigate(url),
		printPDF(requestOptions, fileChan),
	}

	timeoutCtx, cancelTimeout := context.WithTimeout(browserCtx, requestOptions.timeout)
	defer cancelTimeout()

	if err := chromedp.Run(timeoutCtx, actions...); err != nil {
		return nil, fmt.Errorf("failed to run browser: %w", err)
	}

	select {
	case fileContents := <-fileChan:
		return fileContents, nil
	default:
		return nil, fmt.Errorf("failed to render PDF: no data received after browser quit")
	}
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

func (b *Browser) createAllocatorOptions(reqOpts []requestOption) ([]chromedp.ExecAllocatorOption, *requestOptions, error) {
	opts := chromedp.DefaultExecAllocatorOptions[:]
	opts = append(opts, chromedp.Headless, chromedp.DisableGPU)              // TODO: make configurable?
	opts = append(opts, chromedp.NoFirstRun, chromedp.NoDefaultBrowserCheck) // TODO: make configurable?
	opts = append(opts, chromedp.NoSandbox)                                  // TODO: Make this configurable, so we can slowly phase it back in
	opts = append(opts, chromedp.ExecPath(b.Binary))
	for _, arg := range b.Args {
		arg = strings.TrimPrefix(arg, "--")
		equals := strings.Index(arg, "=")
		if equals == -1 {
			opts = append(opts, chromedp.Flag(arg, ""))
		} else {
			opts = append(opts, chromedp.Flag(arg[:equals], arg[equals+1:]))
		}
	}

	requestOptions := &requestOptions{
		timeZone:        b.TimeZone,
		width:           b.Width,
		height:          b.Height,
		paperSize:       b.PaperSize,
		landscape:       b.Landscape,
		printBackground: b.PrintBackground,
		timeout:         b.RenderTimeout,
	}
	for _, opt := range reqOpts {
		if err := opt(requestOptions); err != nil {
			return nil, nil, err
		}
	}
	opts = append(opts, chromedp.Env("TZ="+requestOptions.timeZone))
	opts = append(opts, chromedp.WindowSize(requestOptions.width, requestOptions.height))

	return opts, requestOptions, nil
}

func printPDF(requestOptions *requestOptions, dst chan []byte) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		// We don't need the stream return value; we don't ask for a stream.
		width, height := requestOptions.paperSize.FormatInches()
		output, _, err := page.PrintToPDF().
			WithPrintBackground(requestOptions.printBackground).
			WithLandscape(requestOptions.landscape).
			WithPaperWidth(width).
			WithPaperHeight(height).
			Do(ctx)
		if err != nil {
			return fmt.Errorf("failed to print to PDF: %w", err)
		}
		dst <- output
		return nil
	})
}
