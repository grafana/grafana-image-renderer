package metrics_test

import (
	"testing"

	"github.com/grafana/grafana-image-renderer/pkg/metrics"
	"github.com/stretchr/testify/require"
)

func TestNewRegistryWorks(t *testing.T) {
	t.Parallel()

	require.NotPanics(t, func() {
		// We use MustRegister, which can panic. We just want to make sure it doesn't in this case.
		metrics.NewRegistry()
	}, "expected NewRegistry to not panic")
}
