package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grafana/grafana-image-renderer/pkg/api/middleware"
	"github.com/stretchr/testify/require"
)

func TestMetricsMiddlewareDoesNotPanic(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	middleware := middleware.Recovery(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)

	require.NotPanics(t, func() {
		middleware.ServeHTTP(rec, req)
	}, "middlewared handler panicked on request")
	require.Equal(t, http.StatusOK, rec.Code, "http status code on response")
}
