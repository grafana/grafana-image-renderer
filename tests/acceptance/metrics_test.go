package acceptance

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMetricsEndpoint(t *testing.T) {
	LongTest(t)
	t.Parallel()

	svc := StartImageRenderer(t,
		WithEnv("ENABLE_METRICS", "true")) // only required for node.js

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, svc.HTTPEndpoint+"/metrics", nil)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
}
