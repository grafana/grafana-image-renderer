package acceptance

import (
	"image/png"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/klippa-app/go-pdfium/requests"
	"github.com/stretchr/testify/require"
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
