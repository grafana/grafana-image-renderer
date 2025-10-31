package cmd

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/grafana/grafana-image-renderer/cmd/healthcheck"
	"github.com/grafana/grafana-image-renderer/cmd/sandbox"
	"github.com/grafana/grafana-image-renderer/cmd/server"
	"github.com/grafana/grafana-image-renderer/pkg/config"
	"github.com/grafana/grafana-image-renderer/pkg/service"
	"github.com/urfave/cli/v3"
	"go.opentelemetry.io/otel/trace"
)

func NewRootCmd() *cli.Command {
	return &cli.Command{
		Name:    "grafana-image-renderer",
		Usage:   "A service for Grafana to render images and documents from Grafana websites.",
		Version: service.NewVersionService().GetPrettyVersion(),
		Flags:   config.LoggingFlags(),
		Before: func(ctx context.Context, c *cli.Command) (context.Context, error) {
			loggingConfig, err := config.LoggingConfigFromCommand(c)
			if err != nil {
				return ctx, fmt.Errorf("failed to parse logging config: %w", err)
			}
			leveler, err := loggingConfig.Level.ToSlog()
			if err != nil {
				return ctx, fmt.Errorf("failed to parse log level: %w", err)
			}
			slog.SetDefault(slog.New(
				&traceLogger{
					underlying: slog.NewTextHandler(c.Writer, &slog.HandlerOptions{AddSource: true, Level: leveler}),
				},
			))

			return ctx, nil
		},
		Commands: []*cli.Command{
			healthcheck.NewCmd(),
			server.NewCmd(),
			sandbox.NewCmd(),
		},
	}
}

var _ slog.Handler = (*traceLogger)(nil)

type traceLogger struct {
	underlying slog.Handler
}

func (t *traceLogger) Enabled(ctx context.Context, level slog.Level) bool {
	return t.underlying.Enabled(ctx, level)
}

func (t *traceLogger) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &traceLogger{underlying: t.underlying.WithAttrs(attrs)}
}

func (t *traceLogger) WithGroup(name string) slog.Handler {
	return &traceLogger{underlying: t.underlying.WithGroup(name)}
}

func (t *traceLogger) Handle(ctx context.Context, record slog.Record) error {
	if sc := trace.SpanContextFromContext(ctx); sc.HasTraceID() {
		record.AddAttrs(slog.String("trace_id", sc.TraceID().String()))
	}
	return t.underlying.Handle(ctx, record)
}
