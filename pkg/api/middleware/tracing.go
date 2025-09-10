package middleware

import (
	"context"
	"net/http"

	"github.com/grafana/grafana-image-renderer/pkg/traces"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func Tracing(h http.Handler) http.Handler {
	// We want the traceparent to return back to the requester, so they know what to look up (or what to share with support).
	h = returningTrace(h)
	// otelhttp.NewHandler adds some trace info and starts an http-request span.
	// It costs a few microseconds, but we can spare that.
	h = otelhttp.NewHandler(h, "http-request")
	// Finally, we want to use the same trace as the requester does.
	return propagatingTrace(h)
}

func propagatingTrace(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wireContext := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))
		r = r.WithContext(wireContext)
		h.ServeHTTP(w, r)
	})
}

func returningTrace(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		otel.GetTextMapPropagator().Inject(r.Context(), propagation.HeaderCarrier(w.Header()))
		h.ServeHTTP(w, r)
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
