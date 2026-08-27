package api_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grafana/grafana-image-renderer/pkg/api"
	"github.com/stretchr/testify/require"
)

func TestGetHealthzReturnsOK(t *testing.T) {
	t.Parallel()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/healthz", nil)
	route := api.HandleGetHealthz()
	route.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "OK", rec.Body.String())
}

func TestGetHealthzHandlesWriteError(t *testing.T) {
	t.Parallel()
	errorWriter := &errorResponseWriter{
		ResponseWriter: httptest.NewRecorder(),
	}
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/healthz", nil)
	route := api.HandleGetHealthz()
	route.ServeHTTP(errorWriter, req)
}

type errorResponseWriter struct {
	http.ResponseWriter
}

func (e *errorResponseWriter) Write(b []byte) (int, error) {
	return 0, errors.New("write failed")
}