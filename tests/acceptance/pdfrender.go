package acceptance

import (
	"bytes"
	"image"
	"image/draw"
	"image/png"
	"testing"

	"github.com/gen2brain/go-fitz"
	"github.com/stretchr/testify/require"
)

func PDFtoImage(tb testing.TB, pdf []byte) *image.RGBA {
	tb.Helper()

	doc, err := fitz.NewFromMemory(pdf)
	require.NoError(tb, err, "failed to open PDF from memory")
	defer func() {
		if err := doc.Close(); err != nil {
			tb.Logf("failed to close PDF document: %v", err)
		}
	}()

	var imgs []*image.RGBA
	for page := range doc.NumPage() {
		img, err := doc.Image(page)
		require.NoError(tb, err, "failed to render page %d of PDF", page)
		imgs = append(imgs, img)
	}
	merged := mergeImages(imgs...)

	return merged
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
