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

func TestRegression935(t *testing.T) {
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

	t.Run("timezone query param is respected", func(t *testing.T) {
		t.Parallel()

		net, err := network.New(t.Context())
		require.NoError(t, err, "could not create Docker network")
		testcontainers.CleanupNetwork(t, net)

		StartPrometheus(t, WithNetwork(net, "prometheus"))
		svc := StartImageRenderer(t, WithNetwork(net, "gir"))
		_ = StartGrafana(t,
			WithNetwork(net, "grafana"),
			WithEnv("GF_RENDERING_SERVER_URL", "http://gir:8081/render"),
			WithEnv("GF_RENDERING_CALLBACK_URL", "http://grafana:3000/"),
			WithEnv("GF_RENDERING_RENDERER_TOKEN", rendererAuthToken))

		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, svc.HTTPEndpoint+"/render", nil)
		require.NoError(t, err, "could not construct HTTP request")
		req.Header.Set("Accept", "image/png")
		req.Header.Set("X-Auth-Token", "-")
		query := req.URL.Query()
		query.Set("url", "http://grafana:3000/d/provisioned-prom-testing?render=1&kiosk=true")
		query.Set("timezone", "Asia/Shanghai")
		query.Set("encoding", "png")
		query.Set("width", "1000")
		query.Set("height", "800")
		query.Set("renderKey", renderKey)
		query.Set("domain", "grafana")
		req.URL.RawQuery = query.Encode()

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err, "could not send HTTP request")
		require.Equal(t, http.StatusOK, resp.StatusCode, "unexpected HTTP status code")

		body := ReadBody(t, resp.Body)
		bodyImg := ReadRGBA(t, body)
		const fixture = "render-prometheus-timezone-asia-shanghai.png"
		fixtureImg := ReadFixtureRGBA(t, fixture)
		if !AssertPixelDifference(t, fixtureImg, bodyImg, 15_000) {
			UpdateFixtureIfEnabled(t, fixture, body)
		}
	})
}
