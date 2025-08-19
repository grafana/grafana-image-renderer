package server

import (
	"context"
	"fmt"

	"github.com/grafana/grafana-image-renderer/pkg/api"
	"github.com/grafana/grafana-image-renderer/pkg/chromium"
	"github.com/grafana/grafana-image-renderer/pkg/metrics"
	"github.com/urfave/cli/v3"
)

func NewCmd() *cli.Command {
	return &cli.Command{
		Name:  "server",
		Usage: "Run the server part of the service.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "addr",
				Usage: "The address to listen on for HTTP requests.",
				Value: ":8081",
			},
			&cli.StringFlag{
				Name:      "browser",
				Usage:     "The path to the browser's binary. This is resolved against PATH.",
				TakesFile: true,
				Value:     "chromium",
			},
		},
		Action: run,
	}
}

func run(ctx context.Context, c *cli.Command) error {
	metrics := metrics.NewRegistry()
	browser, err := chromium.NewBrowser(c.String("browser"))
	if err != nil {
		return fmt.Errorf("failed to create browser: %w", err)
	}
	handler, err := api.NewHandler(metrics, browser)
	if err != nil {
		return fmt.Errorf("failed to create API handler: %w", err)
	}
	return api.ListenAndServe(ctx, c.String("addr"), handler)
}
