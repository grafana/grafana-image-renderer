package acceptance

import (
	"errors"
	"fmt"
	"image"
	"testing"

	"github.com/stretchr/testify/assert"
)

func CountPixelDifferences(img1, img2 image.Image) (uint64, error) {
	return CountPixelDifferencesWithTolerance(img1, img2, 0)
}

// Two independent browser renders are never bit-for-bit identical: anti-aliasing
// and colour rounding leave a scattering of pixels off by ±1 in a colour channel.
// A small channelDelta (1-2) absorbs that rasterisation jitter so the count
// reflects meaningful differences rather than noise. Pass 0 for an exact comparison.
func CountPixelDifferencesWithTolerance(img1, img2 image.Image, channelDelta uint8) (uint64, error) {
	if img1.Bounds() != img2.Bounds() {
		return 0, fmt.Errorf("the images are different sizes (%v != %v)", img1.Bounds(), img2.Bounds())
	}

	rgba1, ok := img1.(*image.RGBA)
	if !ok {
		return 0, errors.New("img1 is not an RGBA image")
	}
	rgba2, ok := img2.(*image.RGBA)
	if !ok {
		return 0, errors.New("img2 is not an RGBA image")
	}

	var diff uint64
	for i := range len(rgba1.Pix) / 4 {
		if channelAbsDiff(rgba1.Pix[i*4], rgba2.Pix[i*4]) > channelDelta || // R
			channelAbsDiff(rgba1.Pix[i*4+1], rgba2.Pix[i*4+1]) > channelDelta || // G
			channelAbsDiff(rgba1.Pix[i*4+2], rgba2.Pix[i*4+2]) > channelDelta || // B
			channelAbsDiff(rgba1.Pix[i*4+3], rgba2.Pix[i*4+3]) > channelDelta { // A
			diff++
		}
	}
	return diff, nil
}

func channelAbsDiff(a, b uint8) uint8 {
	if a > b {
		return a - b
	}
	return b - a
}

func AssertPixelDifference(tb testing.TB, img1, img2 image.Image, maxDiff uint64) bool {
	tb.Helper()

	diff, err := CountPixelDifferences(img1, img2)
	return assert.NoError(tb, err, "could not compare images") && assert.LessOrEqual(tb, diff, maxDiff, "images differ in too many pixels (%d > %d)", diff, maxDiff)
}
