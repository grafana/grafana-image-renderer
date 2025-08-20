package api

import (
	"net/http"

	"github.com/grafana/grafana-image-renderer/pkg/api/middleware"
	"github.com/grafana/grafana-image-renderer/pkg/chromium"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// NewHandler creates the API and wires it together.
func NewHandler(
	metrics interface {
		prometheus.Gatherer
		prometheus.Registerer
	},
	browser *chromium.Browser,
	token AuthToken,
) (http.Handler, error) {
	mux := http.NewServeMux()
	mux.Handle("GET /metrics", promhttp.HandlerFor(metrics, promhttp.HandlerOpts{Registry: metrics}))
	mux.Handle("GET /healthz", HandleGetHealthz())
	mux.Handle("GET /version", HandleGetVersion(browser))
	mux.Handle("POST /render", middleware.RequireAuthToken(middleware.TrustedURL(HandlePostRender(browser)), string(token)))

	handler := middleware.RequestMetrics(mux)
	handler = middleware.Recovery(handler) // must come last!
	return handler, nil
}

type AuthToken string
