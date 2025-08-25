package server

import (
	"context"
	"fmt"
	"time"

	"github.com/grafana/grafana-image-renderer/cmd/config"
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
				Name:    "addr",
				Usage:   "The address to listen on for HTTP requests.",
				Value:   ":8081",
				Sources: config.FromConfig("server.addr"),
			},
			&cli.StringFlag{
				Name:      "browser",
				Usage:     "The path to the browser's binary. This is resolved against PATH.",
				TakesFile: true,
				Value:     "chromium",
				Sources:   config.FromConfig("browser.path"),
			},
			&cli.StringSliceFlag{
				Name:    "browser-flags",
				Usage:   "Flags to pass to the browser. These are syntaxed `<flag>` or `<flag>=<value>`. No -- should be passed in for the flag; these are implied.",
				Sources: config.FromConfig("browser.flags"),
			},
			&cli.StringFlag{
				Name:    "auth-token",
				Usage:   "The X-Auth-Token header value that must be sent to the service to permit requests.",
				Value:   "-",
				Sources: config.FromConfig("auth.token"),
			},
		},
		Action: run,
	}
}

func run(ctx context.Context, c *cli.Command) error {
	metrics := metrics.NewRegistry()
	browser, err := chromium.NewBrowser(c.String("browser"), c.StringSlice("browser-flags"), chromium.RenderingOptions{
		Width:              1000,
		Height:             500,
		TimeZone:           "Etc/UTC",
		PaperSize:          chromium.PaperA4,
		PrintBackground:    true,
		Timeout:            time.Second * 30,
		FullHeight:         false,
		Landscape:          true,
		Format:             chromium.RenderingFormatPDF,
		DeviceScaleFactor:  2,
		TimeBetweenScrolls: time.Millisecond * 50,
	})
	if err != nil {
		return fmt.Errorf("failed to create browser: %w", err)
	}
	handler, err := api.NewHandler(metrics, browser, api.AuthToken(c.String("auth-token")))
	if err != nil {
		return fmt.Errorf("failed to create API handler: %w", err)
	}
	return api.ListenAndServe(ctx, c.String("addr"), handler)
}
