package middleware

import (
	"context"
	"net/http"
	"sync"

	"github.com/grafana/grafana-image-renderer/pkg/traces"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func Tracing(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO: Read traceparent header

		tracer := tracer(r.Context())
		ctx, span := tracer.Start(r.Context(), "HTTP request",
			trace.WithAttributes(
				attribute.String("http.method", r.Method),
				attribute.String("http.url", r.URL.String()),
				attribute.String("http.user_agent", r.UserAgent()),
			))
		defer span.End()
		r = r.WithContext(ctx)

		recorder := &statusRecordingResponseWriter{rw: w}
		h.ServeHTTP(recorder, r)
		span.SetAttributes(attribute.Int("http.status_code", recorder.status))
		if recorder.status >= 400 {
			span.SetStatus(codes.Error, http.StatusText(recorder.status))
		} else {
			span.SetStatus(codes.Ok, http.StatusText(recorder.status))
		}
	})
}

func TracingFor(name string, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tracer := tracer(r.Context())
		ctx, span := tracer.Start(r.Context(), name)
		defer span.End()
		r = r.WithContext(ctx)

		h.ServeHTTP(w, r)
	})
}

func tracer(ctx context.Context) trace.Tracer {
	return traces.TracerFromContext(ctx, "github.com/grafana/grafana-image-renderer/pkg/api/middleware")
}

var (
	_ http.ResponseWriter = (*statusRecordingResponseWriter)(nil)
	_ http.Flusher        = (*statusRecordingResponseWriter)(nil)
)

type statusRecordingResponseWriter struct {
	rw     http.ResponseWriter
	once   sync.Once
	status int
}

func (s *statusRecordingResponseWriter) Header() http.Header {
	return s.rw.Header()
}

func (s *statusRecordingResponseWriter) Write(b []byte) (int, error) {
	s.once.Do(func() {
		s.status = http.StatusOK
	})
	return s.rw.Write(b)
}

func (s *statusRecordingResponseWriter) WriteHeader(statusCode int) {
	s.once.Do(func() {
		s.status = statusCode
	})
	s.rw.WriteHeader(statusCode)
}

func (s *statusRecordingResponseWriter) Flush() {
	if flusher, ok := s.rw.(http.Flusher); ok {
		flusher.Flush()
	}
}
