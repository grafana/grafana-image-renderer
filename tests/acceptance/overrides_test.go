package acceptance

import (
	"net/http"
	"strings"
	"testing"

	"github.com/go-jose/go-jose/v4"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/network"
)

func TestRequestConfigOverrides(t *testing.T) {
	LongTest(t)
	// t.Parallel()

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

	t.Run("min width and height overrides by URL pattern", func(t *testing.T) {
		// t.Parallel()

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
			// t.Parallel()

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
			// t.Parallel()

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
