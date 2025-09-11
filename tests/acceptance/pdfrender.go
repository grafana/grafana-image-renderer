package acceptance

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/stretchr/testify/require"
)

func PDFtoPNG(tb testing.TB, pdf []byte) []byte {
	tb.Helper()

	dir := tb.TempDir()
	err := os.WriteFile(dir+"/in.pdf", pdf, 0644)
	require.NoError(tb, err, "failed to write input PDF to a file")
	return PDFFiletoPNG(tb, dir+"/in.pdf")
}

func PDFFiletoPNG(tb testing.TB, pdfFile string) []byte {
	tb.Helper()

	opts := chromedp.DefaultExecAllocatorOptions[:]
	opts = append(opts, chromedp.WindowSize(2000, 1200))
	opts = append(opts, chromedp.Env("TZ=Etc/UTC"))
	if browser := os.Getenv("BROWSER"); browser != "" {
		opts = append(opts, chromedp.ExecPath(browser))
	}

	allocatorCtx, cancelAllocator := chromedp.NewExecAllocator(tb.Context(), opts...)
	defer cancelAllocator()
	browserCtx, cancelBrowser := chromedp.NewContext(allocatorCtx)
	defer cancelBrowser()

	var buf []byte
	err := chromedp.Run(browserCtx,
		chromedp.EmulateViewport(1920, 1080, chromedp.EmulateLandscape),
		chromedp.Navigate("file://"+pdfFile),
		chromedp.ActionFunc(func(ctx context.Context) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Millisecond * 500):
				return nil
			}
		}),
		chromedp.ActionFunc(func(ctx context.Context) error {
			output, err := page.CaptureScreenshot().
				WithFormat(page.CaptureScreenshotFormatPng).
				WithCaptureBeyondViewport(true).
				Do(ctx)
			if err != nil {
				return fmt.Errorf("screenshot failed: %w", err)
			}
			buf = output
			return nil
		}))
	require.NoError(tb, err, "failed to render PDF to PNG")
	return buf
}
