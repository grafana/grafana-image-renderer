package server

import (
	"context"
	"fmt"
	"slices"

	"github.com/grafana/grafana-image-renderer/cmd/config"
	"github.com/grafana/grafana-image-renderer/pkg/api"
	"github.com/grafana/grafana-image-renderer/pkg/metrics"
	"github.com/grafana/grafana-image-renderer/pkg/service"
	"github.com/grafana/grafana-image-renderer/pkg/traces"
	"github.com/urfave/cli/v3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

func NewCmd() *cli.Command {
	return &cli.Command{
		Name:  "server",
		Usage: "Run the server part of the service.",
		Flags: slices.Concat([]cli.Flag{
			&cli.StringFlag{
				Name:     "addr",
				Usage:    "The address to listen on for HTTP requests.",
				Category: "Server",
				Value:    ":8081",
				Sources:  config.FromConfig("server.addr", "SERVER_ADDR"),
			},
			&cli.StringSliceFlag{
				Name:     "auth-token",
				Usage:    "The X-Auth-Token header value that must be sent to the service to permit requests. May be repeated.",
				Category: "Server",
				Value:    []string{"-"},
				Sources:  config.FromConfig("auth.token", "AUTH_TOKEN"),
			},

			&cli.StringFlag{
				Name:      "browser",
				Usage:     "The path to the browser's binary. This is resolved against PATH.",
				Category:  "Browser",
				TakesFile: true,
				Value:     "chromium",
				Sources:   config.FromConfig("browser.path", "BROWSER_PATH"),
			},
			&cli.StringSliceFlag{
				Name:     "browser-flags",
				Usage:    "Flags to pass to the browser. These are syntaxed `<flag>` or `<flag>=<value>`. No -- should be passed in for the flag; these are implied.",
				Category: "Browser",
				Sources:  config.FromConfig("browser.flags", "BROWSER_FLAGS"),
			},
			&cli.BoolFlag{
				Name:     "browser-gpu",
				Usage:    "Enable GPU support in the browser.",
				Category: "Browser",
				Sources:  config.FromConfig("browser.gpu", "BROWSER_GPU"),
			},
		}, traces.TracerFlags()),
		Action: run,
	}
}

func run(ctx context.Context, c *cli.Command) error {
	metrics := metrics.NewRegistry()
	tracerProvider, err := traces.NewTracerProvider(ctx, c)
	if err != nil {
		return fmt.Errorf("failed to set up tracer: %w", err)
	}
	if tracerProvider != nil {
		defer func() { _ = tracerProvider.Shutdown(ctx) }()
		ctx = traces.WithTracerProvider(ctx, tracerProvider)
		otel.SetTracerProvider(tracerProvider)
		otel.SetTextMapPropagator(propagation.TraceContext{})
	}
	browser := service.NewBrowserService(c.String("browser"), c.StringSlice("browser-flags"),
		service.WithViewport(1000, 500),
		service.WithGPU(c.Bool("browser-gpu")))
	versions := service.NewVersionService()
	handler, err := api.NewHandler(metrics, browser, api.AuthTokens(c.StringSlice("auth-token")), versions)
	if err != nil {
		return fmt.Errorf("failed to create API handler: %w", err)
	}
	return api.ListenAndServe(ctx, c.String("addr"), handler)
}
