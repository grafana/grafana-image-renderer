package api_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grafana/grafana-image-renderer/pkg/api"
	"github.com/grafana/grafana-image-renderer/pkg/traces"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
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

	spanRecorder := tracetest.NewSpanRecorder()
	tracerProvider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder))
	t.Cleanup(func() {
		require.NoError(t, tracerProvider.Shutdown(context.Background()))
	})

	errorWriter := &errorResponseWriter{
		ResponseWriter: httptest.NewRecorder(),
	}
	ctx := traces.WithTracerProvider(t.Context(), tracerProvider)
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/healthz", nil)
	route := api.HandleGetHealthz()
	route.ServeHTTP(errorWriter, req)

	spans := spanRecorder.Ended()
	require.Len(t, spans, 1)
	require.Equal(t, codes.Error, spans[0].Status().Code)
}

type errorResponseWriter struct {
	http.ResponseWriter
}

func (e *errorResponseWriter) Write(b []byte) (int, error) {
	return 0, errors.New("write failed")
}
