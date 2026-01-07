package middleware

import (
	"log/slog"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/codes"
)

var MetricRecoveredRequests = prometheus.NewCounter(prometheus.CounterOpts{
	Name: "http_recovered_requests_total",
	Help: "How many HTTP requests have panicked but recovered to not crash the application?",
})

// Recovery ensures a single HTTP handler cannot panic the entire application.
// This ensures we can withstand bugs as best as possible, but they will still be logged and output as metrics, so we can react.
func Recovery(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tracer := tracer(r.Context())
		ctx, span := tracer.Start(r.Context(), "Recovery")
		defer span.End()
		r = r.WithContext(ctx)

		defer func() {
			if err := recover(); err != nil {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				MetricRecoveredRequests.Inc()
				slog.ErrorContext(ctx, "Request panicked", "panic", err, "url", r.URL, "method", r.Method)
				span.SetStatus(codes.Error, "panic in HTTP handler")
			}
		}()

		h.ServeHTTP(w, r)
		span.SetStatus(codes.Ok, "no panic")
	})
}
