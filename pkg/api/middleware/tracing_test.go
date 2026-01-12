package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grafana/grafana-image-renderer/pkg/api/middleware"
	"github.com/grafana/grafana-image-renderer/pkg/traces"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestTracing(t *testing.T) {
	t.Parallel()

	okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux := http.NewServeMux()
	mux.Handle("GET /", okHandler)

	t.Run("without TracerProvider in context", func(t *testing.T) {
		t.Parallel()

		rec := httptest.NewRecorder()
		middleware := middleware.Recovery(mux)
		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)

		middleware.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code, "http status code on response")
	})

	t.Run("with no-op TracerProvider in context", func(t *testing.T) {
		t.Parallel()

		tracerProvider := noop.NewTracerProvider()
		ctx := traces.WithTracerProvider(t.Context(), tracerProvider)

		for range 5 {
			// re-registering the tracer shouldn't do anything bad.

			rec := httptest.NewRecorder()
			middleware := middleware.Recovery(mux)
			req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)

			middleware.ServeHTTP(rec, req)
			require.Equal(t, http.StatusOK, rec.Code, "http status code on response")
		}
	})
}
