package acceptance

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestServiceStartsHealthily(t *testing.T) {
	LongTest(t)
	t.Parallel()

	svc := StartImageRenderer(t)

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, svc.HTTPEndpoint, nil)
	require.NoError(t, err, "could not construct HTTP request to /")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err, "could not make HTTP request to /")
	require.Equal(t, http.StatusOK, resp.StatusCode, "expected HTTP 200 OK from /")
}
