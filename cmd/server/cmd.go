package server

import (
	"context"
	"fmt"
	"slices"

	"github.com/grafana/grafana-image-renderer/pkg/api"
	"github.com/grafana/grafana-image-renderer/pkg/config"
	"github.com/grafana/grafana-image-renderer/pkg/metrics"
	"github.com/grafana/grafana-image-renderer/pkg/service"
	"github.com/grafana/grafana-image-renderer/pkg/traces"
	"github.com/urfave/cli/v3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

func NewCmd() *cli.Command {
	return &cli.Command{
		Name:   "server",
		Usage:  "Run the server part of the service.",
		Flags:  slices.Concat(config.ServerFlags(), config.TracingFlags(), config.BrowserFlags()),
		Action: run,
	}
}

func run(ctx context.Context, c *cli.Command) error {
	serverConfig, err := config.ServerConfigFromCommand(c)
	if err != nil {
		return fmt.Errorf("failed to parse server config: %w", err)
	}
	browserConfig, err := config.BrowserConfigFromCommand(c)
	if err != nil {
		return fmt.Errorf("failed to parse browser config: %w", err)
	}
	tracingConfig, err := config.TracingConfigFromCommand(c)
	if err != nil {
		return fmt.Errorf("failed to parse tracing config: %w", err)
	}
	tracerProvider, err := traces.NewTracerProvider(ctx, tracingConfig)
	if err != nil {
		return fmt.Errorf("failed to set up tracer: %w", err)
	}
	if tracerProvider != nil {
		defer func() { _ = tracerProvider.Shutdown(ctx) }()
		ctx = traces.WithTracerProvider(ctx, tracerProvider)
		otel.SetTracerProvider(tracerProvider)
		otel.SetTextMapPropagator(propagation.TraceContext{})
	}
	browser := service.NewBrowserService(browserConfig)
	versions := service.NewVersionService()
	metrics := metrics.NewRegistry()
	handler, err := api.NewHandler(metrics, serverConfig, browser, versions)
	if err != nil {
		return fmt.Errorf("failed to create API handler: %w", err)
	}
	return api.ListenAndServe(ctx, serverConfig, handler)
}
