package acceptance

import (
	"net/http"
	"strings"
	"testing"

	"github.com/go-jose/go-jose/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/network"
)

// createRenderKey creates a JWT render key for authenticating with the image renderer
func createRenderKey(t *testing.T, authToken string) string {
	t.Helper()
	joseSigner, err := jose.NewSigner(jose.SigningKey{
		Algorithm: jose.HS512,
		Key:       []byte(authToken),
	}, nil)
	require.NoError(t, err, "could not create JWT signer")
	joseSignature, err := joseSigner.Sign([]byte(`{"renderUser": {"org_id": 1, "user_id": 1, "org_role": "Admin"}}`))
	require.NoError(t, err, "could not sign JWT")
	renderKey, err := joseSignature.CompactSerialize()
	require.NoError(t, err, "could not serialize JWT")
	return renderKey
}

func TestRequestConfigOverrides(t *testing.T) {
	LongTest(t)
	t.Parallel()

	rendererAuthToken := strings.Repeat("-", 512/8)
	renderKey := createRenderKey(t, rendererAuthToken)

	t.Run("min width and height overrides by URL pattern", func(t *testing.T) {
		t.Parallel()

		net, err := network.New(t.Context())
		require.NoError(t, err, "could not create Docker network")
		testcontainers.CleanupNetwork(t, net)

		StartPrometheus(t, WithNetwork(net, "prometheus"))

		// Start image renderer with:
		// - Default min width/height of 1000x1000
		// - Override for URLs matching "very-long-prometheus-dashboard" to use 500x500
		svc := StartImageRenderer(t,
			WithNetwork(net, "gir"),
			WithArgs(
				"server",
				"--browser.min-width=1000",
				"--browser.min-height=1000",
				// Override pattern: URLs containing "very-long-prometheus-dashboard" get 500x500 min dimensions
				"--browser.override=very-long-prometheus-dashboard=--browser.min-width=500 --browser.min-height=500",
			),
		)

		_ = StartGrafana(t,
			WithNetwork(net, "grafana"),
			WithEnv("GF_RENDERING_SERVER_URL", "http://gir:8081/render"),
			WithEnv("GF_RENDERING_CALLBACK_URL", "http://grafana:3000/"),
			WithEnv("GF_RENDERING_RENDERER_TOKEN", rendererAuthToken))

		t.Run("non-matching URL uses default min dimensions (1000x1000)", func(t *testing.T) {
			t.Parallel()

			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, svc.HTTPEndpoint+"/render", nil)
			require.NoError(t, err, "could not construct HTTP request")
			req.Header.Set("Accept", "image/png")
			req.Header.Set("X-Auth-Token", "-")
			query := req.URL.Query()
			// Use the prometheus dashboard which does NOT match the override pattern
			query.Set("url", "http://grafana:3000/d/provisioned-prom-testing?render=1&from=1699333200000&to=1699344000000&kiosk=true")
			query.Set("encoding", "png")
			query.Set("width", "1")  // Request tiny width, should be clamped to min
			query.Set("height", "1") // Request tiny height, should be clamped to min
			query.Set("renderKey", renderKey)
			query.Set("domain", "grafana")
			req.URL.RawQuery = query.Encode()

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err, "could not send HTTP request")
			require.Equal(t, http.StatusOK, resp.StatusCode, "unexpected HTTP status code")

			body := ReadBody(t, resp.Body)
			bodyImg := ReadRGBA(t, body)

			// Should use default min dimensions (1000x1000)
			bounds := bodyImg.Bounds()
			require.Equal(t, 1000, bounds.Dx(), "image width should be clamped to default min width of 1000")
			require.Equal(t, 1000, bounds.Dy(), "image height should be clamped to default min height of 1000")
		})

		t.Run("matching URL uses override min dimensions (500x500)", func(t *testing.T) {
			t.Parallel()

			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, svc.HTTPEndpoint+"/render", nil)
			require.NoError(t, err, "could not construct HTTP request")
			req.Header.Set("Accept", "image/png")
			req.Header.Set("X-Auth-Token", "-")
			query := req.URL.Query()
			// Use the very-long dashboard which DOES match the override pattern
			query.Set("url", "http://grafana:3000/d/very-long-prometheus-dashboard?render=1&from=1699333200000&to=1699344000000&kiosk=true")
			query.Set("encoding", "png")
			query.Set("width", "1")  // Request tiny width, should be clamped to override min
			query.Set("height", "1") // Request tiny height, should be clamped to override min
			query.Set("renderKey", renderKey)
			query.Set("domain", "grafana")
			req.URL.RawQuery = query.Encode()

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err, "could not send HTTP request")
			require.Equal(t, http.StatusOK, resp.StatusCode, "unexpected HTTP status code")

			body := ReadBody(t, resp.Body)
			bodyImg := ReadRGBA(t, body)

			// Should use override min dimensions (500x500)
			bounds := bodyImg.Bounds()
			require.Equal(t, 500, bounds.Dx(), "image width should be clamped to override min width of 500")
			require.Equal(t, 500, bounds.Dy(), "image height should be clamped to override min height of 500")
		})
	})
}

// TestOverridesOnlyAffectSpecifiedValues is a regression test that verifies that override settings for a regex url do not affect the other render settings for that url.
// It starts two Grafana instances and an image renderer with an override set for one and not the other.
// We want the images to be identical, so the override value is still the default. The intent of the override is to trigger the reconstructFlags logic.
func TestOverridesOnlyAffectSpecifiedValues(t *testing.T) {
	LongTest(t)

	rendererAuthToken := strings.Repeat("-", 512/8)
	renderKey := createRenderKey(t, rendererAuthToken)

	t.Run("override with default page-scale-factor produces identical images", func(t *testing.T) {
		net, err := network.New(t.Context())
		require.NoError(t, err, "could not create Docker network")
		testcontainers.CleanupNetwork(t, net)

		StartPrometheus(t, WithNetwork(net, "prometheus"))

		// Start two Grafana instances with different network names
		_ = StartGrafana(t,
			WithNetwork(net, "grafana-no-override"),
			WithEnv("GF_RENDERING_SERVER_URL", "http://gir:8081/render"),
			WithEnv("GF_RENDERING_CALLBACK_URL", "http://grafana-no-override:3000/"),
			WithEnv("GF_RENDERING_RENDERER_TOKEN", rendererAuthToken))

		_ = StartGrafana(t,
			WithNetwork(net, "grafana-with-override"),
			WithEnv("GF_RENDERING_SERVER_URL", "http://gir:8081/render"),
			WithEnv("GF_RENDERING_CALLBACK_URL", "http://grafana-with-override:3000/"),
			WithEnv("GF_RENDERING_RENDERER_TOKEN", rendererAuthToken))

		// Start image renderer with:
		// - Default page-scale-factor is 1.0
		// - Override for "grafana-with-override" that explicitly sets page-scale-factor to 1.0 (same as default)
		// If the override system works correctly, renders from both Grafanas should be identical.
		svc := StartImageRenderer(t,
			WithNetwork(net, "gir"),
			WithArgs(
				"server",
				"--browser.page-scale-factor=1.0",
				"--browser.override=grafana-with-override=--browser.page-scale-factor=1.0",
			),
		)

		// Render from Grafana without override
		reqNoOverride, err := http.NewRequestWithContext(t.Context(), http.MethodGet, svc.HTTPEndpoint+"/render", nil)
		require.NoError(t, err, "could not construct HTTP request for no-override")
		reqNoOverride.Header.Set("Accept", "image/png")
		reqNoOverride.Header.Set("X-Auth-Token", "-")
		queryNoOverride := reqNoOverride.URL.Query()
		queryNoOverride.Set("url", "http://grafana-no-override:3000/d/provisioned-prom-testing?render=1&from=1699333200000&to=1699344000000&kiosk=true")
		queryNoOverride.Set("encoding", "png")
		queryNoOverride.Set("width", "800")
		queryNoOverride.Set("height", "600")
		queryNoOverride.Set("renderKey", renderKey)
		queryNoOverride.Set("domain", "grafana-no-override")
		reqNoOverride.URL.RawQuery = queryNoOverride.Encode()

		respNoOverride, err := http.DefaultClient.Do(reqNoOverride)
		require.NoError(t, err, "could not send HTTP request for no-override")
		require.Equal(t, http.StatusOK, respNoOverride.StatusCode, "unexpected HTTP status code for no-override")

		bodyNoOverride := ReadBody(t, respNoOverride.Body)
		imgNoOverride := ReadRGBA(t, bodyNoOverride)

		// Render from Grafana with override (page-scale-factor=1.0, same as default)
		reqWithOverride, err := http.NewRequestWithContext(t.Context(), http.MethodGet, svc.HTTPEndpoint+"/render", nil)
		require.NoError(t, err, "could not construct HTTP request for with-override")
		reqWithOverride.Header.Set("Accept", "image/png")
		reqWithOverride.Header.Set("X-Auth-Token", "-")
		queryWithOverride := reqWithOverride.URL.Query()
		queryWithOverride.Set("url", "http://grafana-with-override:3000/d/provisioned-prom-testing?render=1&from=1699333200000&to=1699344000000&kiosk=true")
		queryWithOverride.Set("encoding", "png")
		queryWithOverride.Set("width", "800")
		queryWithOverride.Set("height", "600")
		queryWithOverride.Set("renderKey", renderKey)
		queryWithOverride.Set("domain", "grafana-with-override")
		reqWithOverride.URL.RawQuery = queryWithOverride.Encode()

		respWithOverride, err := http.DefaultClient.Do(reqWithOverride)
		require.NoError(t, err, "could not send HTTP request for with-override")
		require.Equal(t, http.StatusOK, respWithOverride.StatusCode, "unexpected HTTP status code for with-override")

		bodyWithOverride := ReadBody(t, respWithOverride.Body)
		imgWithOverride := ReadRGBA(t, bodyWithOverride)

		// Both images should have the same dimensions
		require.Equal(t, imgNoOverride.Bounds(), imgWithOverride.Bounds(),
			"images should have the same dimensions")

		// Both images should be identical (0 pixel differences)
		// This validates that the override with the same value as default doesn't
		// accidentally change any other rendering settings.
		diff, err := CountPixelDifferences(imgNoOverride, imgWithOverride)
		require.NoError(t, err, "could not compare images")
		assert.Equal(t, uint64(0), diff,
			"images should be identical when override uses same value as default; "+
				"this indicates the override system is incorrectly changing other config values")
	})
}
