package acceptance

import (
	"fmt"
	"mime"
	"net/http"
	"net/url"
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

		t.Run("with set width and height", func(t *testing.T) {
			t.Parallel()

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
			const fixture = "render-prometheus-set-width-height.png"
			fixtureImg := ReadFixtureRGBA(t, fixture)
			if !AssertPixelDifference(t, fixtureImg, bodyImg, 17_000) {
				UpdateFixtureIfEnabled(t, fixture, body)
			}
		})
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

		t.Run("with defaults", func(t *testing.T) {
			t.Parallel()

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

			body := ReadBody(t, resp.Body)
			const fixture = "render-prometheus-defaults.csv"
			if !assert.Equal(t, string(ReadFixture(t, fixture)), string(body), "fixture and actual CSV responses from renderer differ") {
				UpdateFixtureIfEnabled(t, fixture, body)
			}
		})
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

		t.Run("with defaults", func(t *testing.T) {
			t.Parallel()

			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, svc.HTTPEndpoint+"/render", nil)
			require.NoError(t, err, "could not construct HTTP request to Grafana")
			req.Header.Set("Accept", "application/pdf")
			req.Header.Set("X-Auth-Token", "-")
			query := req.URL.Query()
			query.Set("url", "http://grafana:3000/d/provisioned-prom-testing?render=1&from=1699333200000&to=1699344000000&kiosk=true")
			query.Set("encoding", "pdf")
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

		t.Run("with US English language", func(t *testing.T) {
			t.Parallel()

			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, svc.HTTPEndpoint+"/render", nil)
			require.NoError(t, err, "could not construct HTTP request to Grafana")
			req.Header.Set("Accept", "application/pdf")
			req.Header.Set("X-Auth-Token", "-")
			query := req.URL.Query()
			query.Set("url", "http://grafana:3000/d/provisioned-prom-testing?render=1&from=1704063600000&to=1704236400000&kiosk=true")
			query.Set("encoding", "pdf")
			query.Set("renderKey", renderKey)
			query.Set("domain", "grafana")
			query.Set("width", "2000")
			query.Set("height", "800")
			req.URL.RawQuery = query.Encode()
			req.Header.Set("Accept-Language", "en-US,en;q=0.9")

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err, "could not send HTTP request to Grafana")
			require.Equal(t, http.StatusOK, resp.StatusCode, "unexpected HTTP status code from Grafana")

			pdfBody := ReadBody(t, resp.Body)
			image := PDFtoImage(t, pdfBody)
			const fixture = "render-prometheus-pdf-us-lang.png"
			fixtureImg := ReadFixtureRGBA(t, fixture)
			if !AssertPixelDifference(t, fixtureImg, image, 17_000) {
				UpdateFixtureIfEnabled(t, fixture+".pdf", pdfBody)
				UpdateFixtureIfEnabled(t, fixture, EncodePNG(t, image))
			}
		})

		t.Run("with German language", func(t *testing.T) {
			t.Parallel()

			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, svc.HTTPEndpoint+"/render", nil)
			require.NoError(t, err, "could not construct HTTP request to Grafana")
			req.Header.Set("Accept", "application/pdf")
			req.Header.Set("X-Auth-Token", "-")
			query := req.URL.Query()
			query.Set("url", "http://grafana:3000/d/provisioned-prom-testing?render=1&from=1704063600000&to=1704236400000&kiosk=true")
			query.Set("encoding", "pdf")
			query.Set("renderKey", renderKey)
			query.Set("domain", "grafana")
			query.Set("width", "2000")
			query.Set("height", "800")
			req.URL.RawQuery = query.Encode()
			req.Header.Set("Accept-Language", "de-DE,de;q=0.9")

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err, "could not send HTTP request to Grafana")
			require.Equal(t, http.StatusOK, resp.StatusCode, "unexpected HTTP status code from Grafana")

			pdfBody := ReadBody(t, resp.Body)
			image := PDFtoImage(t, pdfBody)
			const fixture = "render-prometheus-pdf-de-lang.png"
			fixtureImg := ReadFixtureRGBA(t, fixture)
			if !AssertPixelDifference(t, fixtureImg, image, 17_000) {
				UpdateFixtureIfEnabled(t, fixture+".pdf", pdfBody)
				UpdateFixtureIfEnabled(t, fixture, EncodePNG(t, image))
			}
		})

		for _, paper := range []string{"letter", "legal", "tabloid", "ledger", "a0", "a1", "a2", "a3", "a4", "a5", "a6"} {
			t.Run("print with paper="+paper, func(t *testing.T) {
				t.Parallel()

				req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, svc.HTTPEndpoint+"/render", nil)
				require.NoError(t, err, "could not construct HTTP request to Grafana")
				req.Header.Set("Accept", "application/pdf")
				req.Header.Set("X-Auth-Token", "-")
				query := req.URL.Query()
				query.Set("url", "http://grafana:3000/d/provisioned-prom-testing?render=1&from=1699333200000&to=1699344000000&kiosk=true&pdf.format="+paper)
				query.Set("encoding", "pdf")
				query.Set("renderKey", renderKey)
				query.Set("domain", "grafana")
				req.URL.RawQuery = query.Encode()

				resp, err := http.DefaultClient.Do(req)
				require.NoError(t, err, "could not send HTTP request to Grafana")
				require.Equal(t, http.StatusOK, resp.StatusCode, "unexpected HTTP status code from Grafana")

				pdfBody := ReadBody(t, resp.Body)
				image := PDFtoImage(t, pdfBody)
				fixture := fmt.Sprintf("render-prometheus-pdf-%s.png", paper)
				fixtureImg := ReadFixtureRGBA(t, fixture)
				if !AssertPixelDifference(t, fixtureImg, image, 17_000) {
					UpdateFixtureIfEnabled(t, fmt.Sprintf("render-prometheus-pdf-%s.pdf", paper), pdfBody)
					UpdateFixtureIfEnabled(t, fixture, EncodePNG(t, image))
				}
			})
		}

		for _, printBackground := range []bool{true, false} {
			t.Run(fmt.Sprintf("print with background=%v", printBackground), func(t *testing.T) {
				t.Parallel()

				req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, svc.HTTPEndpoint+"/render", nil)
				require.NoError(t, err, "could not construct HTTP request to Grafana")
				req.Header.Set("Accept", "application/pdf")
				req.Header.Set("X-Auth-Token", "-")
				query := req.URL.Query()
				query.Set("url", fmt.Sprintf("http://grafana:3000/d/provisioned-prom-testing?render=1&from=1699333200000&to=1699344000000&kiosk=true&pdf.printBackground=%v", printBackground))
				query.Set("encoding", "pdf")
				query.Set("renderKey", renderKey)
				query.Set("domain", "grafana")
				req.URL.RawQuery = query.Encode()

				resp, err := http.DefaultClient.Do(req)
				require.NoError(t, err, "could not send HTTP request to Grafana")
				require.Equal(t, http.StatusOK, resp.StatusCode, "unexpected HTTP status code from Grafana")

				pdfBody := ReadBody(t, resp.Body)
				image := PDFtoImage(t, pdfBody)
				fixture := fmt.Sprintf("render-prometheus-pdf-printBackground-%v.png", printBackground)
				fixtureImg := ReadFixtureRGBA(t, fixture)
				if !AssertPixelDifference(t, fixtureImg, image, 17_000) {
					UpdateFixtureIfEnabled(t, fmt.Sprintf("render-prometheus-pdf-printBackground-%v.pdf", printBackground), pdfBody)
					UpdateFixtureIfEnabled(t, fixture, EncodePNG(t, image))
				}
			})
		}
	})

	t.Run("render very long prometheus dashboard as PDF", func(t *testing.T) {
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

		t.Run("render many pages", func(t *testing.T) {
			t.Parallel()

			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, svc.HTTPEndpoint+"/render", nil)
			require.NoError(t, err, "could not construct HTTP request to Grafana")
			req.Header.Set("Accept", "application/pdf")
			req.Header.Set("X-Auth-Token", "-")
			query := req.URL.Query()
			query.Set("url", "http://grafana:3000/d/very-long-prometheus-dashboard?render=1&from=1699333200000&to=1699344000000&kiosk=true")
			query.Set("encoding", "pdf")
			query.Set("renderKey", renderKey)
			query.Set("domain", "grafana")
			req.URL.RawQuery = query.Encode()

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err, "could not send HTTP request to Grafana")
			require.Equal(t, http.StatusOK, resp.StatusCode, "unexpected HTTP status code from Grafana")

			pdfBody := ReadBody(t, resp.Body)
			image := PDFtoImage(t, pdfBody)
			const fixture = "render-very-long-prometheus-dashboard.png"
			fixtureImg := ReadFixtureRGBA(t, fixture)
			if !AssertPixelDifference(t, fixtureImg, image, 17_000) {
				UpdateFixtureIfEnabled(t, fixture+".pdf", pdfBody)
				UpdateFixtureIfEnabled(t, fixture, EncodePNG(t, image))
			}
		})

		for name, pageRange := range map[string]string{
			"single-page":     "4",
			"all pages":       "",
			"range in middle": "2-4",
			"first 3":         "1-3",
			"1 and 3":         "1, 3",
		} {
			t.Run("print with pageRanges="+name, func(t *testing.T) {
				t.Parallel()

				req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, svc.HTTPEndpoint+"/render", nil)
				require.NoError(t, err, "could not construct HTTP request to Grafana")
				req.Header.Set("Accept", "application/pdf")
				req.Header.Set("X-Auth-Token", "-")
				query := req.URL.Query()
				query.Set("url", "http://grafana:3000/d/very-long-prometheus-dashboard?render=1&from=1699333200000&to=1699344000000&kiosk=true&pdf.pageRanges="+url.QueryEscape(pageRange))
				query.Set("encoding", "pdf")
				query.Set("renderKey", renderKey)
				query.Set("domain", "grafana")
				req.URL.RawQuery = query.Encode()

				resp, err := http.DefaultClient.Do(req)
				require.NoError(t, err, "could not send HTTP request to Grafana")
				require.Equal(t, http.StatusOK, resp.StatusCode, "unexpected HTTP status code from Grafana")

				pdfBody := ReadBody(t, resp.Body)
				image := PDFtoImage(t, pdfBody)
				fixture := fmt.Sprintf("render-very-long-prometheus-dashboard-pageranges-%s.png", strings.ReplaceAll(name, " ", "-"))
				fixtureImg := ReadFixtureRGBA(t, fixture)
				if !AssertPixelDifference(t, fixtureImg, image, 17_000) {
					UpdateFixtureIfEnabled(t, fixture+".pdf", pdfBody)
					UpdateFixtureIfEnabled(t, fixture, EncodePNG(t, image))
				}
			})
		}
	})

	t.Run("render panel dashboards as PNG", func(t *testing.T) {
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

		requestDashboard := func(tb testing.TB, id string) []byte {
			req, err := http.NewRequestWithContext(tb.Context(), http.MethodGet, svc.HTTPEndpoint+"/render", nil)
			require.NoError(t, err, "could not construct HTTP request to Grafana")
			req.Header.Set("Accept", "image/png")
			req.Header.Set("X-Auth-Token", "-")
			query := req.URL.Query()
			query.Set("url", fmt.Sprintf("http://grafana:3000/d/%s?render=1&from=1699333200000&to=1699344000000&kiosk=true", id))
			query.Set("encoding", "png")
			query.Set("width", "1400")
			query.Set("height", "800")
			query.Set("renderKey", renderKey)
			query.Set("domain", "grafana")
			req.URL.RawQuery = query.Encode()

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err, "could not send HTTP request to Grafana")
			require.Equal(t, http.StatusOK, resp.StatusCode, "unexpected HTTP status code from Grafana")

			return ReadBody(tb, resp.Body)
		}

		t.Run("geomap", func(t *testing.T) {
			t.Parallel()

			t.Run("with default settings", func(t *testing.T) {
				t.Parallel()

				body := requestDashboard(t, "default-geomap")
				bodyImg := ReadRGBA(t, body)
				const fixture = "render-panel-geomap-default-settings.png"
				fixtureImg := ReadFixtureRGBA(t, fixture)
				if !AssertPixelDifference(t, fixtureImg, bodyImg, 17_000) {
					UpdateFixtureIfEnabled(t, fixture, body)
				}
			})

			t.Run("with USA states flight info", func(t *testing.T) {
				t.Parallel()

				body := requestDashboard(t, "geomap-with-usa-flights")
				bodyImg := ReadRGBA(t, body)
				const fixture = "render-panel-geomap-with-usa-flights.png"
				fixtureImg := ReadFixtureRGBA(t, fixture)
				if !AssertPixelDifference(t, fixtureImg, bodyImg, 17_000) {
					UpdateFixtureIfEnabled(t, fixture, body)
				}
			})
		})
	})
}
