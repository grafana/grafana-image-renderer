package acceptance

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"io"
	"math"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/network"
)

func TestRenderingGrafana(t *testing.T) {
	LongTest(t)
	t.Parallel()

	t.Run("render login page", func(t *testing.T) {
		t.Parallel()

		net, err := network.New(t.Context())
		require.NoError(t, err, "could not create Docker network")
		testcontainers.CleanupNetwork(t, net)

		svc := StartImageRenderer(t, WithNetwork(net, "gir"))
		_ = StartGrafana(t,
			WithNetwork(net, "grafana"),
			WithEnv("GF_AUTH_ANONYMOUS_HIDE_VERSION", "true"), // hide the version number, which would otherwise often fail the test
			WithEnv("GF_RENDERING_SERVER_URL", "http://gir:8081/render"),
			WithEnv("GF_RENDERING_CALLBACK_URL", "http://grafana:3000/"),
			WithEnv("GF_LOG_FILTERS", "rendering:debug"))

		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, svc.HTTPEndpoint+"/render", nil)
		require.NoError(t, err, "could not construct HTTP request to Grafana")
		req.Header.Set("Accept", "image/png")
		req.Header.Set("X-Auth-Token", "-")
		query := req.URL.Query()
		query.Set("url", "http://grafana:3000/")
		query.Set("encoding", "png")
		query.Set("width", "1200")
		query.Set("height", "900")
		req.URL.RawQuery = query.Encode()

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err, "could not send HTTP request to Grafana")
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode, "unexpected HTTP status code from Grafana")

		fixture, err := os.Open("fixtures/grafana-login-page.png")
		require.NoError(t, err, "could not read fixture file")
		fixtureImg, err := png.Decode(fixture)
		require.NoError(t, err, "could not decode fixture PNG image")
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err, "could not read response body")
		bodyImg, err := png.Decode(bytes.NewReader(body))
		require.NoError(t, err, "could not decode response PNG image")

		assert.Equal(t, bodyImg.Bounds().Max.X, 1200, "rendered image has wrong width")
		assert.Equal(t, bodyImg.Bounds().Max.Y, 900, "rendered image has wrong height")

		diff, err := imgDiff(fixtureImg, bodyImg)
		ok := assert.NoError(t, err, "could not diff images") && assert.LessOrEqual(t, diff, uint64(3000), "rendered login page has changed significantly")
		if !ok && os.Getenv("UPDATE_FIXTURES") == "true" {
			err := os.WriteFile("fixtures/grafana-login-page.png", body, 0644)
			require.NoError(t, err, "could not update fixture file")
		}
	})
}

func imgDiff(a, b image.Image) (uint64, error) {
	castA, ok := a.(*image.RGBA)
	if !ok {
		return 0, fmt.Errorf("a is not RGBA")
	}
	castB, ok := b.(*image.RGBA)
	if !ok {
		return 0, fmt.Errorf("b is not RGBA")
	}
	return imgDiffRGBA(castA, castB)
}

func imgDiffRGBA(a, b *image.RGBA) (uint64, error) {
	if a.Bounds() != b.Bounds() {
		return 0, fmt.Errorf("images have different sizes")
	}

	diffs := int(0)
	for i := range a.Pix {
		d := int(a.Pix[i]) - int(b.Pix[i])
		diffs += d * d
	}
	return uint64(math.Sqrt(float64(diffs))), nil
}
