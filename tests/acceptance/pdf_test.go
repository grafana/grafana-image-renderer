package acceptance

import (
	"bytes"
	"image/png"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	pdfcore "github.com/unidoc/unipdf/v4/core"
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

		parser, err := pdfcore.NewParser(bytes.NewReader(body))
		require.NoError(t, err, "could not parse PDF response body")
		require.NotPanics(t, func() { parser.PdfVersion() }, "could not read PDF version, not a valid PDF")
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
