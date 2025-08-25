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

	// RenderingOptionsPrototype is a clonable instance of the options we want to apply to rendering requests.
	RenderingOptionsPrototype RenderingOptions
}

func NewBrowser(
	binary string,
	args []string,
	prototype RenderingOptions,
) (*Browser, error) {
	return &Browser{
		Binary:                    binary,
		Args:                      args,
		RenderingOptionsPrototype: prototype,
	}, nil
}

// GetVersion finds the version of the browser.
func (b *Browser) GetVersion(ctx context.Context) (string, error) {
	version, err := exec.CommandContext(ctx, b.Binary, "--version").Output()
	if err != nil {
		return "", fmt.Errorf("failed to get version of browser: %w", err)
	}
	return string(bytes.TrimSpace(version)), nil
}

type RenderingOptions struct {
	// URL is the location to visit. This is required.
	URL string

	// Width is the width of the viewport in pixels.
	Width int
	// Height is the height of the viewport in pixels.
	Height int
	// TimeZone is the IANA time-zone name, e.g. `Etc/UTC`.
	TimeZone string

	// FullHeight defines whether an image render should include the entire height of the webpage.
	FullHeight bool

	// PaperSize is how large a PDF should be per page.
	PaperSize PaperSize
	// Landscape defines which orientation PDFs should be in: vertical (default) or landscape (aka horizontal).
	Landscape bool
	// PrintBackground defines whether to include background graphics, such as div backgrounds or dark page backgrounds.
	// You generally want this to preserve contrasts, but it is possibly wasteful if you intend on printing the document to physical paper later on.
	PrintBackground bool

	// Timeout is the duration of time we'll wait on render requests to complete.
	Timeout time.Duration
}

// RenderPDF visits the website and waits for it to be fully rendered.
//
// The entire PDF is returned as a byte slice.
func (b *Browser) RenderPDF(ctx context.Context, renderingOptions RenderingOptions) ([]byte, error) {
	if renderingOptions.URL == "" {
		return nil, fmt.Errorf("rendering options must have a URL set")
	}

	browserID, err := uuid.NewRandom() // TODO: Use traceID if exists in context
	if err != nil {
		return nil, fmt.Errorf("failed to generate browser ID: %w", err)
	}
	log := slog.With("browser_id", browserID.String())

	allocatorOptions, err := b.createAllocatorOptions(renderingOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to create allocator options: %w", err)
	}
	allocatorCtx, cancelAllocator := chromedp.NewExecAllocator(ctx, allocatorOptions...)
	defer cancelAllocator()
	browserCtx, cancelBrowser := chromedp.NewContext(allocatorCtx, browserLoggers(ctx, log))
	defer cancelBrowser()

	fileChan := make(chan []byte, 1) // buffered: we don't want the browser to stick around while we try to export this value.
	actions := []chromedp.Action{
		chromedp.Navigate(renderingOptions.URL),
		printPDF(renderingOptions, fileChan),
	}

	timeoutCtx, cancelTimeout := context.WithTimeout(browserCtx, renderingOptions.Timeout)
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

func (b *Browser) createAllocatorOptions(renderingOptions RenderingOptions) ([]chromedp.ExecAllocatorOption, error) {
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

	opts = append(opts, chromedp.Env("TZ="+renderingOptions.TimeZone))
	opts = append(opts, chromedp.WindowSize(renderingOptions.Width, renderingOptions.Height))

	return opts, nil
}

func printPDF(requestOptions RenderingOptions, dst chan []byte) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		// We don't need the stream return value; we don't ask for a stream.
		width, height, err := requestOptions.PaperSize.FormatInches()
		if err != nil {
			return fmt.Errorf("failed to get paper size dimensions: %w", err)
		}

		output, _, err := page.PrintToPDF().
			WithPrintBackground(requestOptions.PrintBackground).
			WithLandscape(requestOptions.Landscape).
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
