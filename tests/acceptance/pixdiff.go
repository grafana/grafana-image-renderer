package acceptance

import (
	"errors"
	"fmt"
	"image"
	"testing"

	"github.com/stretchr/testify/assert"
)

func CountPixelDifferences(img1, img2 image.Image) (uint64, error) {
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
		if rgba1.Pix[i*4] != rgba2.Pix[i*4] || // R
			rgba1.Pix[i*4+1] != rgba2.Pix[i*4+1] || // G
			rgba1.Pix[i*4+2] != rgba2.Pix[i*4+2] || // B
			rgba1.Pix[i*4+3] != rgba2.Pix[i*4+3] { // A
			diff++
		}
	}
	return diff, nil
}

func AssertPixelDifference(tb testing.TB, img1, img2 image.Image, maxDiff uint64) bool {
	tb.Helper()

	diff, err := CountPixelDifferences(img1, img2)
	return assert.NoError(tb, err, "could not compare images") && assert.LessOrEqual(tb, diff, maxDiff, "images differ in too many pixels (%d > %d)", diff, maxDiff)
}
