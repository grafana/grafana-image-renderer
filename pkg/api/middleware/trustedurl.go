package middleware

import (
	"net/http"
	"net/url"

	"github.com/prometheus/client_golang/prometheus"
)

var MetricTrustedURLRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
	Name: "http_trusted_url_requests_total",
	Help: "Counts the requests with URL queries",
}, []string{"result"})

// TrustedURL ensures that the `url` query parameter (if it exists) aims at an HTTP or HTTPS website, disallowing e.g. `chrome://` and `file://`.
func TrustedURL(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tracer := tracer(r.Context())
		_, span := tracer.Start(r.Context(), "TrustedURL")
		defer span.End()

		urlQuery := r.URL.Query().Get("url")
		if urlQuery == "" {
			// Nothing to check: let it through.
			MetricTrustedURLRequests.WithLabelValues("missing-query").Inc()
			span.End() // we don't want to track the next middleware in this span
			h.ServeHTTP(w, r)
			return
		}

		parsed, err := url.Parse(urlQuery)
		if err != nil {
			MetricTrustedURLRequests.WithLabelValues("invalid-url").Inc()
			http.Error(w, "Invalid URL in query", http.StatusBadRequest)
			return
		}

		if parsed.Scheme != "http" && parsed.Scheme != "https" {
			http.Error(w, "Forbidden query url protocol", http.StatusForbidden) // TODO: Wrong use of Forbidden, should be BadRequest...
			MetricTrustedURLRequests.WithLabelValues("invalid-protocol").Inc()
			return
		}

		MetricTrustedURLRequests.WithLabelValues("valid").Inc()
		span.End() // we don't want to track the next middleware in this span
		h.ServeHTTP(w, r)
	})
}
