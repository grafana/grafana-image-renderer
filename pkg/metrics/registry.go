package metrics

import (
	"github.com/grafana/grafana-image-renderer/pkg/api/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

func NewRegistry() *prometheus.Registry {
	registry := prometheus.NewRegistry()
	registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		collectors.NewBuildInfoCollector(),

		middleware.MetricAuthenticatedRequestAttempt,
		middleware.MetricRequestsInFlight,
		middleware.MetricRequestDurations,
		middleware.MetricRecoveredRequests,
		middleware.MetricTrustedURLRequests,
	)
	return registry
}
