package acceptance

import (
	"encoding/csv"
	"mime"
	"net/http"
	"strings"
	"testing"

	"github.com/go-jose/go-jose/v4"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/network"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
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

	t.Run("render all-panels dashboard as PNG", func(t *testing.T) {
		t.Parallel()

		net, err := network.New(t.Context())
		require.NoError(t, err, "could not create Docker network")
		testcontainers.CleanupNetwork(t, net)

		svc := StartImageRenderer(t, WithNetwork(net, "gir"))
		_ = StartGrafana(t,
			WithNetwork(net, "grafana"),
			WithEnv("GF_RENDERING_SERVER_URL", "http://gir:8081/render"),
			WithEnv("GF_RENDERING_CALLBACK_URL", "http://grafana:3000/"),
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

		body := ReadBody(t, resp.Body)
		bodyImg := ReadRGBA(t, body)
		AssertRGBASize(t, bodyImg, 1000, 800)
		const fixture = "render-all-panels.png"
		fixtureImg := ReadFixtureRGBA(t, fixture)
		// We happen to have a map on the image. It can mean significant change, because it renders differently for... some reason.
		if !AssertPixelDifference(t, fixtureImg, bodyImg, 150_000) {
			UpdateFixtureIfEnabled(t, fixture, body)
		}
	})

	t.Run("render prometheus dashboard as PNG", func(t *testing.T) {
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
		require.NoError(t, err, "could not construct HTTP request to Grafana")
		req.Header.Set("Accept", "image/png")
		req.Header.Set("X-Auth-Token", "-")
		query := req.URL.Query()
		query.Set("url", "http://grafana:3000/d/provisioned-prom-testing?render=1&from=1699333200000&to=1699344000000&kiosk=true")
		query.Set("encoding", "png")
		query.Set("width", "1400")
		query.Set("height", "800")
		query.Set("renderKey", renderKey)
		query.Set("domain", "grafana")
		req.URL.RawQuery = query.Encode()

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err, "could not send HTTP request to Grafana")
		require.Equal(t, http.StatusOK, resp.StatusCode, "unexpected HTTP status code from Grafana")

		body := ReadBody(t, resp.Body)
		bodyImg := ReadRGBA(t, body)
		AssertRGBASize(t, bodyImg, 1400, 800)
		const fixture = "render-prometheus.png"
		fixtureImg := ReadFixtureRGBA(t, fixture)
		if !AssertPixelDifference(t, fixtureImg, bodyImg, 17_000) {
			UpdateFixtureIfEnabled(t, fixture, body)
		}
	})

	t.Run("render prometheus dashboard as CSV", func(t *testing.T) {
		t.Parallel()
		OnlyEnterprise(t)

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

		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, svc.HTTPEndpoint+"/render/csv", nil)
		require.NoError(t, err, "could not construct HTTP request to Grafana")
		req.Header.Set("Accept", "text/csv, */*")
		req.Header.Set("X-Auth-Token", "-")
		query := req.URL.Query()
		query.Set("url", "http://grafana:3000/d-csv/provisioned-prom-testing?from=1699333200000&to=1699344000000&panelId=1&render=1&orgId=1&timezone=browser")
		query.Set("encoding", "csv")
		query.Set("renderKey", renderKey)
		query.Set("domain", "grafana")
		req.URL.RawQuery = query.Encode()

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err, "could not send HTTP request to Grafana")
		require.Equal(t, http.StatusOK, resp.StatusCode, "unexpected HTTP status code from Grafana")

		// Grafana requires this to save the file somewhere.
		contentDisposition := resp.Header.Get("Content-Disposition")
		_, params, err := mime.ParseMediaType(contentDisposition)
		require.NoError(t, err, "could not parse Content-Disposition header")
		require.NotEmpty(t, params["filename"], "no filename in Content-Disposition header")

		reader := csv.NewReader(transform.NewReader(resp.Body, unicode.BOMOverride(encoding.Nop.NewDecoder())))
		reader.LazyQuotes = true
		records, err := reader.ReadAll()
		require.NoError(t, err, "could not parse CSV response from image-renderer")
		require.NotEmpty(t, records, "no records in CSV response from image-renderer")
		require.Equal(t, []string{"Time", "1"}, records[0])
	})

	t.Run("render prometheus dashboard as PDF", func(t *testing.T) {
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
		require.NoError(t, err, "could not construct HTTP request to Grafana")
		req.Header.Set("Accept", "application/pdf")
		req.Header.Set("X-Auth-Token", "-")
		query := req.URL.Query()
		query.Set("url", "http://grafana:3000/d/provisioned-prom-testing?render=1&from=1699333200000&to=1699344000000&kiosk=true")
		query.Set("encoding", "pdf")
		query.Set("width", "1400")
		query.Set("height", "800")
		query.Set("renderKey", renderKey)
		query.Set("domain", "grafana")
		req.URL.RawQuery = query.Encode()

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err, "could not send HTTP request to Grafana")
		require.Equal(t, http.StatusOK, resp.StatusCode, "unexpected HTTP status code from Grafana")

		pdfBody := ReadBody(t, resp.Body)
		image := PDFtoImage(t, pdfBody)
		const fixture = "render-prometheus-pdf.png"
		fixtureImg := ReadFixtureRGBA(t, fixture)
		if !AssertPixelDifference(t, fixtureImg, image, 17_000) {
			UpdateFixtureIfEnabled(t, fixture+".pdf", pdfBody)
			UpdateFixtureIfEnabled(t, fixture, EncodePNG(t, image))
		}
	})
}
