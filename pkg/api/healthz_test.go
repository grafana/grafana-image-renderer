package api_test

import (
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

	require.Equal(t, http.StatusOK, rec.Code, "http status code on response")
	require.Equal(t, "OK", rec.Body.String(), "body on response")
}
