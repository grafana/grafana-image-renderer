package chromium

import (
	"bytes"
	"context"
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
	// Additional headers to write.
	Headers map[string]string
	// Cookies contains the various cookies we want to write in the browser.
	Cookies []Cookie
	// Format defines what type of file should be returned.
	Format RenderingFormat

	// Width is the width of the viewport in pixels.
	Width int
	// Height is the height of the viewport in pixels.
	Height int
	// MaxWidth is the upper limit to Width. If it is larger, Width is capped.
	MaxWidth int
	// MaxHeight is the upper limit to Height. If it is larger, Height is capped.
	MaxHeight int
	// DeviceScaleFactor is how much the web-page should be scaled in the request.
	DeviceScaleFactor float64
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
	// TimeBetweenScrolls is how long to wait on a page of the rendered website before scrolling further.
	TimeBetweenScrolls time.Duration
}

func (r *RenderingOptions) SetHeader(key, value string) {
	if r.Headers == nil {
		r.Headers = make(map[string]string)
	}
	r.Headers[key] = value
}

func (r *RenderingOptions) Normalise(prototype RenderingOptions) error {
	if strings.HasPrefix(r.URL, "socket://") {
		return fmt.Errorf("image rendering in socket mode is not supported")
	}

	if r.Timeout <= 0 {
		r.Timeout = prototype.Timeout
	}

	if r.Width < 10 {
		r.Width = prototype.Width
	}
	if r.Width > r.MaxWidth && r.MaxWidth > 0 {
		r.Width = r.MaxWidth
	}

	if r.Height == -1 {
		r.FullHeight = true
		r.Height = int(float64(r.Width) * 0.75)
	}

	if r.Height < 10 {
		r.Height = prototype.Height
	}
	if r.Height > r.MaxHeight && r.MaxHeight > 0 {
		r.Height = r.MaxHeight
	}

	// TODO: What are scaled thumbnails??
	if r.DeviceScaleFactor <= 0 {
		// TODO: Scale image accordingly instead of using deviceScaleFactor
		r.DeviceScaleFactor = 1
		/*
			JS:
			   if (options.deviceScaleFactor <= 0) {
			     options.scaleImage = options.deviceScaleFactor * -1;
			     options.deviceScaleFactor = 1;

			     if (options.scaleImage > 1) {
			       options.width *= options.scaleImage;
			       options.height *= options.scaleImage;
			     } else {
			       options.scaleImage = undefined;
			     }
			   } else if (options.deviceScaleFactor > this.config.maxDeviceScaleFactor) {
			     options.deviceScaleFactor = this.config.deviceScaleFactor;
			   }
		*/
	}

	return nil
}

// Render visits the website and waits for it to be fully rendered.
//
// The entire PDF is returned as a byte slice.
func (b *Browser) Render(ctx context.Context, renderingOptions RenderingOptions) ([]byte, error) {
	if renderingOptions.URL == "" {
		return nil, fmt.Errorf("rendering options must have a URL set")
	}
	if err := renderingOptions.Normalise(b.RenderingOptionsPrototype); err != nil {
		return nil, err
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
	var printer chromedp.Action
	switch renderingOptions.Format {
	case RenderingFormatPDF:
		printer = printPDF(renderingOptions, fileChan)
	case RenderingFormatPNG:
		printer = screenshotPNG(renderingOptions, fileChan)
	default:
		return nil, fmt.Errorf("unsupported rendering format: %s", renderingOptions.Format)
	}

	actions := []chromedp.Action{
		emulation.SetPageScaleFactor(renderingOptions.DeviceScaleFactor),
		setHeaders(renderingOptions.Headers),
		setCookies(renderingOptions.Cookies),
		chromedp.Navigate(renderingOptions.URL),
		scrollForElements(renderingOptions.TimeBetweenScrolls),
		waitForViz(renderingOptions),
		printer,
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
		return nil, fmt.Errorf("failed to render: no data received after browser quit")
	}
}

func setHeaders(headers map[string]string) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		if len(headers) == 0 {
			return nil
		}

		hdrs := make(network.Headers, len(headers))
		for k, v := range headers {
			hdrs[k] = v
		}
		return network.SetExtraHTTPHeaders(hdrs).Do(ctx)
	})
}

func setCookies(cookies []Cookie) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		for _, cookie := range cookies {
			params := network.SetCookie(cookie.Name, cookie.Value)
			params.Domain = cookie.Domain
			if err := params.Do(ctx); err != nil {
				return fmt.Errorf("failed to set cookie %q: %w", cookie.Name, err)
			}
		}
		return nil
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

	return opts, nil
}

func printPDF(requestOptions RenderingOptions, dst chan []byte) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		width, height, err := requestOptions.PaperSize.FormatInches()
		if err != nil {
			return fmt.Errorf("failed to get paper size dimensions: %w", err)
		}

		scale := 1.0
		if requestOptions.DeviceScaleFactor != 0 {
			scale = 1.0 / requestOptions.DeviceScaleFactor
		}

		// We don't need the stream return value; we don't ask for a stream.
		output, _, err := page.PrintToPDF().
			WithPrintBackground(requestOptions.PrintBackground).
			WithLandscape(requestOptions.Landscape).
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

func screenshotPNG(requestOptions RenderingOptions, dst chan []byte) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		output, err := page.CaptureScreenshot().
			WithFormat(page.CaptureScreenshotFormatPng).
			WithCaptureBeyondViewport(requestOptions.FullHeight).
			Do(ctx)
		if err != nil {
			return fmt.Errorf("failed to capture screenshot: %w", err)
		}
		dst <- output
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

		time.Sleep(timeBetweenScrolls)
		for range scrolls {
			err := chromedp.Evaluate(`window.scrollBy(0, window.innerHeight, { behavior: 'instant' })`, nil).Do(ctx)
			if err != nil {
				return fmt.Errorf("failed to scroll: %w", err)
			}
			time.Sleep(timeBetweenScrolls)
		}

		err = chromedp.Evaluate(`window.scrollTo(0, 0, { behavior: 'instant' })`, nil).Do(ctx)
		if err != nil {
			return fmt.Errorf("failed to scroll to top: %w", err)
		}

		return nil
	})
}

func waitForViz(renderingOptions RenderingOptions) chromedp.Action {
	script := fmt.Sprintf(`(() => {
		if (window.__grafanaSceneContext) {
			return window.__grafanaRunningQueryCount === 0;
		}

		const isFullPage = true;
		if (isFullPage) {
			const panelCount = document.querySelectorAll('[data-panelId]').length;
			const panelsRendered = document.querySelectorAll("[class$='panel-count']");
			let panelsRenderedCount = 0;
			panelsRendered.forEach(value => {
				if (value.childElementCount > 0) {
					panelsRenderedCount++;
				}
			});

			const rowCount = document.querySelectorAll('.dashboard-row').length || document.querySelectorAll("[data-testid='dashboard-row-container']").length;
			return (panelsRenderedCount + rowCount) >= panelCount;
		}

		const panelCount = document.querySelectorAll('.panel-solo').length || document.querySelectorAll("[class$='panel-container']").length;
		return window.panelsRendered >= panelCount || panelCount === 0;
	})(%v)`, renderingOptions.FullHeight)
	return chromedp.Poll(script, nil, chromedp.WithPollingMutation(), chromedp.WithPollingTimeout(0))
}

type Cookie struct {
	Name   string
	Value  string
	Domain string
}

type RenderingFormat string

const (
	RenderingFormatPDF RenderingFormat = "pdf"
	RenderingFormatPNG RenderingFormat = "png"
)

func (f *RenderingFormat) UnmarshalText(text []byte) error {
	text = bytes.ToLower(text)
	switch RenderingFormat(text) {
	case RenderingFormatPDF, RenderingFormatPNG:
		*f = RenderingFormat(text)
		return nil
	default:
		return fmt.Errorf("invalid rendering format %q", text)
	}
}

func (f RenderingFormat) ContentType() string {
	switch f {
	case RenderingFormatPDF:
		return "application/pdf"
	case RenderingFormatPNG:
		return "image/png"
	default:
		return "application/octet-stream" // fallback
	}
}
