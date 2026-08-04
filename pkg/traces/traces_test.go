package traces

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.41.0"
)

// resource.Default() and our service attributes must share a schema URL;
// resource.Merge rejects mismatches (startup crash when tracing.endpoint is set).
func TestResourceMergeMatchesSDKSchema(t *testing.T) {
	t.Parallel()

	_, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("grafana-image-renderer"),
		),
	)
	require.NoError(t, err)
}
