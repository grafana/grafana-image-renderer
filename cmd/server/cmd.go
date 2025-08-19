package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/grafana/grafana-image-renderer/pkg/metrics"
	"github.com/grafana/grafana-image-renderer/pkg/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/urfave/cli/v3"
)

func NewCmd() *cli.Command {
	return &cli.Command{
		Name:  "server",
		Usage: "Run the server part of the service.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "addr",
				Usage:       "The address to listen on for HTTP requests.",
				DefaultText: ":8081",
			},
		},
		Action: run,
	}
}

func run(ctx context.Context, c *cli.Command) error {
	metrics := metrics.NewRegistry()
	mux := http.NewServeMux()
	mux.Handle("GET /metrics", promhttp.HandlerFor(metrics, promhttp.HandlerOpts{Registry: metrics}))
	mux.Handle("GET /healthz", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}))

	handler := middleware.RequestMetrics(mux)
	handler = middleware.Recovery(handler) // must come last!
	if err := http.ListenAndServe(c.String("addr"), handler); err != nil {
		return fmt.Errorf("http server failed: %w", err)
	}
	return nil
}
