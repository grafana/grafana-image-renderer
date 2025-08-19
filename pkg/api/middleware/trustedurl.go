package middleware

import (
	"net/http"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

var MetricTrustedURLRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
	Name: "http_trusted_url_requests",
	Help: "Counts the requests with URL queries",
}, []string{"result"})

// TrustedURL ensures that the `url` query parameter (if it exists) aims at an HTTP or HTTPS website, disallowing e.g. `chrome://` and `file://`.
func TrustedURL(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		url := r.URL.Query().Get("url")
		if url != "" && !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
			http.Error(w, "Forbidden query url protocol", http.StatusForbidden) // TODO: Wrong use of Forbidden, should be BadRequest...
			MetricTrustedURLRequests.WithLabelValues("invalid-protocol").Inc()
			return
		} else if url == "" {
			MetricTrustedURLRequests.WithLabelValues("missing-query").Inc()
		} else {
			MetricTrustedURLRequests.WithLabelValues("valid").Inc()
		}
		h.ServeHTTP(w, r)
	})
}
