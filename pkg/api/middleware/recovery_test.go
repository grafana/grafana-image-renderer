package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grafana/grafana-image-renderer/pkg/api/middleware"
	"github.com/stretchr/testify/require"
)

func TestRecovery(t *testing.T) {
	t.Parallel()

	panickingHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("oh no")
	})
	okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	t.Run("non-panicking handler returns fine", func(t *testing.T) {
		t.Parallel()

		rec := httptest.NewRecorder()
		middleware := middleware.Recovery(okHandler)
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)

		require.NotPanics(t, func() {
			middleware.ServeHTTP(rec, req)
		}, "middlewared handler panicked on request")
		require.Equal(t, http.StatusOK, rec.Code, "http status code on response")
	})

	t.Run("panicking handler returns error", func(t *testing.T) {
		t.Parallel()

		rec := httptest.NewRecorder()
		middleware := middleware.Recovery(panickingHandler)
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)

		require.NotPanics(t, func() {
			middleware.ServeHTTP(rec, req)
		}, "middlewared handler panicked on request")
		require.Equal(t, http.StatusInternalServerError, rec.Code, "http status code on response")
	})
}
