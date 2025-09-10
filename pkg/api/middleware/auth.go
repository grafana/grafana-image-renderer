package middleware

import (
	"net/http"
	"slices"

	"github.com/prometheus/client_golang/prometheus"
)

var MetricAuthenticatedRequestAttempt = prometheus.NewCounterVec(prometheus.CounterOpts{
	Name: "http_authenticated_request_attempt",
	Help: "Counts the attempts of authenticated requests",
}, []string{"result"})

// RequireAuthToken demands the request has a valid X-Auth-Token header attached to it.
func RequireAuthToken(h http.Handler, expectedTokens ...string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tracer := tracer(r.Context())
		ctx, span := tracer.Start(r.Context(), "RequireAuthToken")
		defer span.End()
		r = r.WithContext(ctx)

		token := r.Header.Get("X-Auth-Token")
		if token == "" {
			http.Error(w, "Missing X-Auth-Token header", http.StatusUnauthorized)
			MetricAuthenticatedRequestAttempt.WithLabelValues("missing-header").Inc()
			return
		}
		if slices.Contains(expectedTokens, token) {
			MetricAuthenticatedRequestAttempt.WithLabelValues("valid-token").Inc()
			h.ServeHTTP(w, r)
			return
		}

		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		MetricAuthenticatedRequestAttempt.WithLabelValues("invalid-token").Inc()
	})
}
