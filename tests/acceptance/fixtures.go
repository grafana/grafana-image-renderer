package acceptance

import (
	"bytes"
	"image"
	"image/draw"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ShouldUpdateFixtures() bool {
	return os.Getenv("UPDATE_FIXTURES") == "true"
}

func UpdateFixture(tb testing.TB, name string, data []byte) {
	tb.Helper()

	err := os.WriteFile(filepath.Join("fixtures", name), data, 0644)
	require.NoError(tb, err, "could not update fixture %q", name)
}

func UpdateFixtureIfEnabled(tb testing.TB, name string, data []byte) {
	tb.Helper()

	if ShouldUpdateFixtures() {
		UpdateFixture(tb, name, data)
	}
}

func ReadFixture(tb testing.TB, name string) []byte {
	tb.Helper()

	data, err := os.ReadFile(filepath.Join("fixtures", name))
	require.NoError(tb, err, "could not read fixture %q", name)
	return data
}

func ReadFixtureRGBA(tb testing.TB, name string) *image.RGBA {
	tb.Helper()

	data := ReadFixture(tb, name)
	img, err := png.Decode(bytes.NewReader(data))
	require.NoError(tb, err, "could not decode fixture image %q", name)
	if nrgba, ok := img.(*image.NRGBA); ok {
		// Convert NRGBA to RGBA
		rgba := image.NewRGBA(nrgba.Bounds())
		draw.Draw(rgba, nrgba.Rect, nrgba, image.Point{}, draw.Src)
		return rgba
	}
	rgba, ok := img.(*image.RGBA)
	require.True(tb, ok, "fixture image %q is not in RGBA format (got %T)", name, img)
	return rgba
}

func ReadBody(tb testing.TB, r io.Reader) []byte {
	tb.Helper()

	data, err := io.ReadAll(r)
	require.NoError(tb, err, "could not read body")
	return data
}

func ReadRGBA(tb testing.TB, b []byte) *image.RGBA {
	tb.Helper()

	img, err := png.Decode(bytes.NewReader(b))
	require.NoError(tb, err, "could not decode image body")
	rgba, ok := img.(*image.RGBA)
	require.True(tb, ok, "image body is not in RGBA format")
	return rgba
}

func AssertRGBASize(tb testing.TB, img *image.RGBA, width, height int) bool {
	tb.Helper()

	// We want both errors to be reported, so we use variables
	a := assert.Equal(tb, width, img.Bounds().Max.X, "image has wrong width")
	b := assert.Equal(tb, height, img.Bounds().Max.Y, "image has wrong height")
	return a && b
}
