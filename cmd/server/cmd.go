package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/grafana/grafana-image-renderer/pkg/chromium"
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
	browserOpts := chromium.BrowserOpts{
		Binary: c.String("browser"),
	}

	metrics := metrics.NewRegistry()
	// TODO: Move to some api pkg
	mux := http.NewServeMux()
	mux.Handle("GET /metrics", promhttp.HandlerFor(metrics, promhttp.HandlerOpts{Registry: metrics}))
	mux.Handle("GET /healthz", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}))
	mux.Handle("GET /version", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		version, err := browserOpts.GetVersion(r.Context())
		if err != nil {
			http.Error(w, "failed to get browser version", http.StatusInternalServerError)
			return
		}

		_, _ = w.Write([]byte(version))
	}))

	handler := middleware.RequestMetrics(mux)
	handler = middleware.Recovery(handler) // must come last!

	server := &http.Server{
		Addr:         c.String("addr"),
		Handler:      handler,
		ReadTimeout:  time.Second * 5,
		WriteTimeout: time.Second * 5,
		BaseContext: func(l net.Listener) context.Context {
			return ctx
		},
	}

	slog.Info("serving http", "addr", server.Addr)
	// listen with ctx
	listener, err := (&net.ListenConfig{}).Listen(ctx, "tcp", server.Addr)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	go func() {
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("failed to serve http", "error", err)
		}
	}()
	go func() {
		<-ctx.Done()
		slog.Info("shutting down http server")
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		_ = server.Shutdown(ctx)
	}()
	<-ctx.Done()
	return nil
}
