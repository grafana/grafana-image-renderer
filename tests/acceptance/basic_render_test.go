package acceptance

import (
	"fmt"
	"html"
	"image/color"
	"image/png"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/klippa-app/go-pdfium/requests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
)

func TestBasicRenders(t *testing.T) {
	LongTest(t)
	t.Parallel()

	t.Run("render PDF of service's root route", func(t *testing.T) {
		t.Parallel()

		svc := StartImageRenderer(t)

		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, svc.HTTPEndpoint+"/render?url=http://localhost:8081/&encoding=pdf", nil)
		require.NoError(t, err, "could not construct HTTP request to /render")
		req.Header.Set("Accept", "application/pdf")
		req.Header.Set("X-Auth-Token", "-")
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err, "could not make HTTP request to /render")
		require.Equal(t, http.StatusOK, resp.StatusCode, "expected HTTP 200 OK from /render")

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err, "could not read HTTP response body")

		pool, err := pdfiumPoolOnce()
		require.NoError(t, err)

		instance, err := pool.GetInstance(30 * time.Second)
		require.NoError(t, err)

		t.Cleanup(func() { require.NoError(t, instance.Close()) })

		doc, err := instance.OpenDocument(&requests.OpenDocument{File: &body})
		require.NoError(t, err)

		t.Cleanup(func() {
			_, err := instance.FPDF_CloseDocument(&requests.FPDF_CloseDocument{
				Document: doc.Document,
			})
			require.NoError(t, err)
		})

		pageCount, err := instance.FPDF_GetPageCount(&requests.FPDF_GetPageCount{Document: doc.Document})
		require.NoError(t, err)
		require.Greater(t, pageCount.PageCount, 0)
	})

	t.Run("sandboxed iframes use the requested density", func(t *testing.T) {
		t.Parallel()

		const inner = `<!doctype html><style>html,body{margin:0;width:100%;height:100%}</style><body><script>
requestAnimationFrame(() => requestAnimationFrame(() => parent.postMessage({density:devicePixelRatio,width:innerWidth,height:innerHeight}, '*')));
</script>`
		wrapper := `<!doctype html><style>html,body,iframe{margin:0;width:100%;height:100%;border:0}</style><body><iframe sandbox="allow-scripts" srcdoc="` + html.EscapeString(inner) + `"></iframe><script>
addEventListener('message', event => parent.postMessage(event.data, '*'));
</script>`
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			_, err := fmt.Fprintf(w, `<!doctype html><style>html,body,iframe{margin:0;width:100%%;height:100%%;border:0}</style><body style="background:red"><script>
window.__grafanaRenderBindingSupported = true;
addEventListener('message', event => {
  const frame = document.querySelector('iframe');
  if (event.source !== frame.contentWindow) return;
  const {density,width,height} = event.data;
  if (density === %s && width === 1200 && height === 800) document.body.style.background = 'lime';
  frame.remove();
  requestAnimationFrame(() => requestAnimationFrame(() => window.__grafanaImageRendererMessageChannel('{}')));
});
</script><iframe sandbox="allow-scripts" srcdoc="%s"></iframe>`, r.URL.Query().Get("density"), html.EscapeString(wrapper))
			assert.NoError(t, err)
		}))
		t.Cleanup(server.Close)
		_, port, err := net.SplitHostPort(server.Listener.Addr().String())
		require.NoError(t, err)
		portNumber, err := strconv.Atoi(port)
		require.NoError(t, err)
		svc := StartImageRenderer(t, func(tb testing.TB, req *testcontainers.GenericContainerRequest) {
			req.HostAccessPorts = []int{portNumber}
		})
		for _, tt := range []struct {
			name     string
			scale    string
			expected float64
		}{
			{name: "standard", scale: "1", expected: 1},
			{name: "high", scale: "2", expected: 2},
			{name: "standard after high", scale: "1", expected: 1},
			{name: "fractional", scale: "0.75", expected: 0.75},
			{name: "clamped", scale: "5", expected: 4},
			{name: "default", expected: 1},
		} {
			t.Run(tt.name, func(t *testing.T) {
				req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, svc.HTTPEndpoint+"/render", nil)
				require.NoError(t, err)
				query := req.URL.Query()
				query.Set("url", "http://host.testcontainers.internal:"+port+"/?density="+fmt.Sprintf("%g", tt.expected))
				query.Set("encoding", "png")
				query.Set("width", "1200")
				query.Set("height", "800")
				if tt.scale != "" {
					query.Set("deviceScaleFactor", tt.scale)
				}
				req.URL.RawQuery = query.Encode()
				req.Header.Set("X-Auth-Token", "-")
				resp, err := http.DefaultClient.Do(req)
				require.NoError(t, err)
				t.Cleanup(func() { assert.NoError(t, resp.Body.Close()) })
				require.Equal(t, http.StatusOK, resp.StatusCode)
				image, err := png.Decode(resp.Body)
				require.NoError(t, err)
				require.Equal(t, int(1200*tt.expected), image.Bounds().Dx())
				require.Equal(t, int(800*tt.expected), image.Bounds().Dy())
				require.Equal(t, color.NRGBA{G: 255, A: 255}, color.NRGBAModel.Convert(image.At(0, 0)), "sandboxed content must retain CSS dimensions and render at the requested density")
			})
		}
	})

	t.Run("render PNG of service's root route", func(t *testing.T) {
		t.Parallel()

		svc := StartImageRenderer(t)

		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, svc.HTTPEndpoint+"/render?url=http://localhost:8081/&encoding=png", nil)
		require.NoError(t, err, "could not construct HTTP request to /render")
		req.Header.Set("Accept", "image/png")
		req.Header.Set("X-Auth-Token", "-")
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err, "could not make HTTP request to /render")
		require.Equal(t, http.StatusOK, resp.StatusCode, "expected HTTP 200 OK from /render")

		_, err = png.Decode(resp.Body)
		require.NoError(t, err, "could not decode PNG body")
	})
}
