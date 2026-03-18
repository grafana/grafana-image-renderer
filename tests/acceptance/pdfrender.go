package acceptance

import (
	"bytes"
	"image"
	"image/draw"
	"image/png"
	"math"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/klippa-app/go-pdfium"
	"github.com/klippa-app/go-pdfium/requests"
	"github.com/klippa-app/go-pdfium/webassembly"
	"github.com/stretchr/testify/require"
)

var pdfiumPoolOnce = sync.OnceValues(func() (pdfium.Pool, error) {
	workers := max(1, runtime.NumCPU())
	return webassembly.Init(webassembly.Config{
		MinIdle:      1,
		MaxIdle:      workers,
		MaxTotal:     workers,
		ReuseWorkers: true,
	})
})

func PDFtoImage(tb testing.TB, data []byte) *image.RGBA {
	tb.Helper()

	pool, err := pdfiumPoolOnce()
	require.NoError(tb, err)

	instance, err := pool.GetInstance(30 * time.Second)
	require.NoError(tb, err)

	tb.Cleanup(func() { require.NoError(tb, instance.Close()) })

	doc, err := instance.OpenDocument(&requests.OpenDocument{File: &data})
	require.NoError(tb, err)

	tb.Cleanup(func() {
		_, err := instance.FPDF_CloseDocument(&requests.FPDF_CloseDocument{
			Document: doc.Document,
		})
		require.NoError(tb, err)
	})

	pageCount, err := instance.FPDF_GetPageCount(&requests.FPDF_GetPageCount{Document: doc.Document})
	require.NoError(tb, err)

	var imgs []*image.RGBA
	for i := range pageCount.PageCount {
		page := requests.Page{
			ByIndex: &requests.PageByIndex{
				Document: doc.Document,
				Index:    i,
			},
		}

		pageSize, err := instance.GetPageSize(&requests.GetPageSize{Page: page})
		require.NoError(tb, err)

		rendered, err := instance.RenderPageInPixels(&requests.RenderPageInPixels{
			Width:  pointsToPixels(pageSize.Width),
			Height: pointsToPixels(pageSize.Height),
			Page:   page,
		})
		require.NoError(tb, err)

		imgs = append(imgs, rendered.Result.Image)
	}

	return mergeImages(imgs...)
}

func EncodePNG(tb testing.TB, img *image.RGBA) []byte {
	tb.Helper()
	buf := &bytes.Buffer{}
	require.NoError(tb, png.Encode(buf, img), "failed to encode image to PNG")
	return buf.Bytes()
}

func mergeImages(imgs ...*image.RGBA) *image.RGBA {
	x, y := 0, 0
	for _, img := range imgs {
		if img.Bounds().Dx() > x {
			x = img.Bounds().Dx()
		}
		y += img.Bounds().Dy()
	}

	merged := image.NewRGBA(image.Rect(0, 0, x, y))
	y = 0
	for _, img := range imgs {
		draw.Draw(merged, image.Rect(0, y, img.Bounds().Dx(), y+img.Bounds().Dy()), img, image.Point{0, 0}, draw.Over)
		y += img.Bounds().Dy()
	}
	return merged
}

func pointsToPixels(points float64) int {
	const dpi = 300.
	normalizedPoints := math.Round(points*100) / 100
	return max(1, int(math.Round(normalizedPoints*dpi/72.0)))
}
