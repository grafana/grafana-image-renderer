package acceptance

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/network"
)

// TestBrowserLocale renders a table with locale-aware units as PDF and asserts on the extracted text.
// Text is exact, where a few changed characters could stay within a pixel-difference tolerance.
// The currency column uses the custom unit currency:financial:€:suffix, because Grafana's built-in
// currency units (such as currencyEUR) don't use the browser's locale.
//
// It doesn't run in parallel with other tests. When it did, its four containers started together with
// every other test's, and other renderers rejected more requests with 429 Too Many Requests, because
// the rate limiter checks the whole machine's free memory.
func TestBrowserLocale(t *testing.T) {
	LongTest(t)

	rendererAuthToken := strings.Repeat("-", 512/8)
	renderKey := createRenderKey(t, rendererAuthToken)

	net, err := network.New(t.Context())
	require.NoError(t, err, "could not create Docker network")
	testcontainers.CleanupNetwork(t, net)

	_ = StartGrafana(t,
		WithNetwork(net, "grafana"),
		WithEnv("GF_RENDERING_SERVER_URL", "http://gir:8081/render"),
		WithEnv("GF_RENDERING_CALLBACK_URL", "http://grafana:3000/"),
		WithEnv("GF_RENDERING_RENDERER_TOKEN", rendererAuthToken))

	renderPDFText := func(t *testing.T, svc *ImageRenderer) string {
		t.Helper()

		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, svc.HTTPEndpoint+"/render", nil)
		require.NoError(t, err, "could not construct HTTP request to Grafana")
		req.Header.Set("Accept", "application/pdf")
		req.Header.Set("X-Auth-Token", "-")
		query := req.URL.Query()
		query.Set("url", "http://grafana:3000/d/locale-panels?render=1&kiosk=true")
		query.Set("encoding", "pdf")
		query.Set("renderKey", renderKey)
		query.Set("domain", "grafana")
		req.URL.RawQuery = query.Encode()

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err, "could not send HTTP request to Grafana")
		require.Equal(t, http.StatusOK, resp.StatusCode, "unexpected HTTP status code from Grafana")

		// French groups digits with U+202F on screen, but PDF text extraction may return it as
		// U+00A0 or a plain space, so compare with plain spaces.
		text := PDFText(t, ReadBody(t, resp.Body))
		return strings.NewReplacer("\u202f", " ", "\u00a0", " ").Replace(text)
	}

	// The time column uses Grafana's explicit date format, so it must not change with the locale.
	const timeValue = "2023-11-07 05:00:00"

	t.Run("no locale uses the browser default", func(t *testing.T) {
		t.Parallel()

		svc := StartImageRenderer(t, WithNetwork(net, "gir"))

		text := renderPDFText(t, svc)
		assert.Contains(t, text, "639,127,392.47")
		assert.Contains(t, text, "1,234.50€")
		assert.Contains(t, text, timeValue)
	})

	t.Run("browser.locale formats numbers in that locale", func(t *testing.T) {
		t.Parallel()

		svc := StartImageRenderer(t,
			WithNetwork(net, "gir-fr"),
			WithEnv("BROWSER_LOCALE", "fr-FR"))

		text := renderPDFText(t, svc)
		assert.Contains(t, text, "639 127 392,47")
		assert.Contains(t, text, "1 234,50€")
		assert.Contains(t, text, timeValue)
	})

	t.Run("override pattern sets a different locale", func(t *testing.T) {
		t.Parallel()

		svc := StartImageRenderer(t,
			WithNetwork(net, "gir-de"),
			WithEnv("BROWSER_LOCALE", "fr-FR"),
			WithArgs("server", "--browser.override=locale-panels=--browser.locale=de-DE"))

		text := renderPDFText(t, svc)
		assert.Contains(t, text, "639.127.392,47")
	})
}
