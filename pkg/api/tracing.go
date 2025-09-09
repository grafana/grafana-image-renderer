package api

import (
	"context"

	"github.com/grafana/grafana-image-renderer/pkg/traces"
	"go.opentelemetry.io/otel/trace"
)

func tracer(ctx context.Context) trace.Tracer {
	return traces.TracerFromContext(ctx, "github.com/grafana/grafana-image-renderer/pkg/api")
}
