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
	"strings"
	"testing"

	"github.com/go-jose/go-jose/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/network"
)

func TestRenderingGrafana(t *testing.T) {
	LongTest(t)
	t.Parallel()

	rendererAuthToken := strings.Repeat("-", 512/8)
	joseSigner, err := jose.NewSigner(jose.SigningKey{
		Algorithm: jose.HS512,
		Key:       []byte(rendererAuthToken),
	}, nil)
	require.NoError(t, err, "could not create JWT signer")
	joseSignature, err := joseSigner.Sign([]byte(`{"renderUser": {"org_id": 1, "user_id": 1, "org_role": "Admin"}}`))
	require.NoError(t, err, "could not sign JWT")
	renderKey, err := joseSignature.CompactSerialize()
	require.NoError(t, err, "could not serialize JWT")

	t.Run("render all-panels dashboard", func(t *testing.T) {
		t.Parallel()

		net, err := network.New(t.Context())
		require.NoError(t, err, "could not create Docker network")
		testcontainers.CleanupNetwork(t, net)

		svc := StartImageRenderer(t, WithNetwork(net, "gir"))
		_ = StartGrafana(t,
			WithNetwork(net, "grafana"),
			WithEnv("GF_FEATURE_TOGGLES_ENABLE", "renderAuthJWT"),
			WithEnv("GF_RENDERING_SERVER_URL", "http://gir:8081/render"),
			WithEnv("GF_RENDERING_CALLBACK_URL", "http://grafana:3000/"),
			WithEnv("GF_LOG_FILTERS", "debug"),
			WithEnv("GF_RENDERING_RENDERER_TOKEN", rendererAuthToken))

		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, svc.HTTPEndpoint+"/render", nil)
		require.NoError(t, err, "could not construct HTTP request to Grafana")
		req.Header.Set("Accept", "image/png")
		req.Header.Set("X-Auth-Token", "-")
		query := req.URL.Query()
		query.Set("url", "http://grafana:3000/d/n1jR8vnnz?render=1&from=1699333200000&to=1699344000000&kiosk=true")
		query.Set("encoding", "png")
		query.Set("width", "1000")
		query.Set("height", "800")
		query.Set("renderKey", renderKey)
		query.Set("domain", "grafana")
		req.URL.RawQuery = query.Encode()

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err, "could not send HTTP request to Grafana")
		require.Equal(t, http.StatusOK, resp.StatusCode, "unexpected HTTP status code from Grafana")

		const fixturePath = "fixtures/render-all-panels.png"
		fixture, err := os.Open(fixturePath)
		require.NoError(t, err, "could not read fixture file")
		fixtureImg, err := png.Decode(fixture)
		require.NoError(t, err, "could not decode fixture PNG image")
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err, "could not read response body")
		bodyImg, err := png.Decode(bytes.NewReader(body))
		require.NoError(t, err, "could not decode response PNG image")

		assert.Equal(t, bodyImg.Bounds().Max.X, 1000, "rendered image has wrong width")
		assert.Equal(t, bodyImg.Bounds().Max.Y, 800, "rendered image has wrong height")

		diff, err := imgDiff(fixtureImg, bodyImg)
		ok := assert.NoError(t, err, "could not diff images") && assert.LessOrEqual(t, diff, uint64(3000), "rendered image has changed significantly")
		if !ok && os.Getenv("UPDATE_FIXTURES") == "true" {
			err := os.WriteFile(fixturePath, body, 0o644)
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
