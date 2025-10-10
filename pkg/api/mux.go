package api

import (
	"net/http"

	"github.com/grafana/grafana-image-renderer/pkg/api/middleware"
	"github.com/grafana/grafana-image-renderer/pkg/config"
	"github.com/grafana/grafana-image-renderer/pkg/service"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// NewHandler creates the API and wires it together.
func NewHandler(
	metrics interface {
		prometheus.Gatherer
		prometheus.Registerer
	},
	serverConfig config.ServerConfig,
	browser *service.BrowserService,
	versions *service.VersionService,
) (http.Handler, error) {
	mux := http.NewServeMux()
	mux.Handle("GET /", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("Grafana Image Renderer (Go)"))
	}))
	mux.Handle("GET /metrics", middleware.TracingFor("promhttp.HandlerFor", promhttp.HandlerFor(metrics, promhttp.HandlerOpts{Registry: metrics})))
	mux.Handle("GET /healthz", HandleGetHealthz())
	mux.Handle("GET /version", HandleGetVersion(versions, browser))
	mux.Handle("GET /render", middleware.RequireAuthToken(middleware.TrustedURL(HandleGetRender(browser)), serverConfig.AuthTokens...))
	mux.Handle("GET /render/csv", middleware.RequireAuthToken(middleware.TrustedURL(HandleGetRenderCSV(browser)), serverConfig.AuthTokens...))
	mux.Handle("GET /render/version", HandleGetRenderVersion(versions))

	handler := middleware.RequestMetrics(mux)
	handler = middleware.RequestLogger(handler)
	handler = middleware.Recovery(handler)
	handler = middleware.Tracing(handler)
	return handler, nil
}
