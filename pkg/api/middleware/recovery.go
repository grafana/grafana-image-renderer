package middleware

import (
	"log/slog"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
)

var MetricRecoveredRequests = prometheus.NewCounter(prometheus.CounterOpts{
	Name: "http_recovered_requests",
	Help: "How many HTTP requests have panicked but recovered to not crash the application?",
})

// Recovery ensures a single HTTP handler cannot panic the entire application.
// This ensures we can withstand bugs as best as possible, but they will still be logged and output as metrics, so we can react.
func Recovery(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				MetricRecoveredRequests.Inc()
				slog.Error("Request panicked", "panic", err, "url", r.URL, "method", r.Method)
			}
		}()
		h.ServeHTTP(w, r)
	})
}
