package server

import (
	"context"
	"fmt"
	"log/slog"
	"slices"

	"github.com/grafana/grafana-image-renderer/pkg/api"
	"github.com/grafana/grafana-image-renderer/pkg/config"
	"github.com/grafana/grafana-image-renderer/pkg/metrics"
	"github.com/grafana/grafana-image-renderer/pkg/service"
	"github.com/grafana/grafana-image-renderer/pkg/traces"
	"github.com/urfave/cli/v3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.uber.org/automaxprocs/maxprocs"
)

func NewCmd() *cli.Command {
	return &cli.Command{
		Name:   "server",
		Usage:  "Run the server part of the service.",
		Flags:  slices.Concat(config.ServerFlags(), config.TracingFlags(), config.BrowserFlags(), config.RateLimitFlags(), config.APIFlags()),
		Action: run,
	}
}

type Cfg struct {
	server    config.ServerConfig
	browser   config.BrowserConfig
	tracing   config.TracingConfig
	rateLimit config.RateLimitConfig
	api       config.APIConfig
}

func ParseConfig(c *cli.Command) (*Cfg, error) {
	serverConfig, err := config.ServerConfigFromCommand(c)
	if err != nil {
		return nil, fmt.Errorf("failed to parse server config: %w", err)
	}
	browserConfig, err := config.BrowserConfigFromCommand(c)
	if err != nil {
		return nil, fmt.Errorf("failed to parse browser config: %w", err)
	}
	tracingConfig, err := config.TracingConfigFromCommand(c)
	if err != nil {
		return nil, fmt.Errorf("failed to parse tracing config: %w", err)
	}
	rateLimitConfig, err := config.RateLimitConfigFromCommand(c)
	if err != nil {
		return nil, fmt.Errorf("failed to parse process tracker config: %w", err)
	}
	apiConfig, err := config.APIConfigFromCommand(c)
	if err != nil {
		return nil, fmt.Errorf("failed to parse api config: %w", err)
	}
	return &Cfg{
		server:    serverConfig,
		browser:   browserConfig,
		tracing:   tracingConfig,
		rateLimit: rateLimitConfig,
		api:       apiConfig,
	}, nil
}

func run(ctx context.Context, c *cli.Command) error {
	_, err := maxprocs.Set(
		// We use maxprocs over automaxprocs because we need a new minimum value.
		// 2 is the absolute minimum we can handle, because we use multiple goroutines many places for timeouts.
		maxprocs.Min(2),
		maxprocs.Logger(maxProcsLog))
	if err != nil {
		slog.Info("failed to set GOMAXPROCS", "err", err)
	}

	cfg, err := ParseConfig(c)
	if err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}
	tracerProvider, err := traces.NewTracerProvider(ctx, cfg.tracing)
	if err != nil {
		return fmt.Errorf("failed to set up tracer: %w", err)
	}
	if tracerProvider != nil {
		defer func() { _ = tracerProvider.Shutdown(ctx) }()
		ctx = traces.WithTracerProvider(ctx, tracerProvider)
		otel.SetTracerProvider(tracerProvider)
		otel.SetTextMapPropagator(propagation.TraceContext{})
	}
	processStatService := service.NewProcessStatService(cfg.rateLimit)
	browser := service.NewBrowserService(cfg.browser, processStatService)
	versions := service.NewVersionService()
	metrics := metrics.NewRegistry()
	handler, err := api.NewHandler(metrics, cfg.server, cfg.api, cfg.rateLimit, processStatService, browser, versions)
	if err != nil {
		return fmt.Errorf("failed to create API handler: %w", err)
	}
	return api.ListenAndServe(ctx, cfg.server, handler)
}

func maxProcsLog(format string, args ...any) {
	slog.Debug(fmt.Sprintf(format, args...), "component", "automaxprocs")
}
