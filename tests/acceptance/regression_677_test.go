package acceptance

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
)

// Regression test for: https://github.com/grafana/grafana-image-renderer/issues/677
func TestRegression677(t *testing.T) {
	LongTest(t)
	t.Parallel()

	t.Run("entrypoint is wrapped in tini", func(t *testing.T) {
		t.Parallel()

		client, err := testcontainers.NewDockerClientWithOpts(t.Context())
		require.NoError(t, err, "could not create Docker client")

		resp, err := client.ImageInspect(t.Context(), GetDockerImage(t))
		require.NoError(t, err, "could not inspect image")

		require.Equal(t, "tini", resp.Config.Entrypoint[0], "entrypoint is not tini")
	})
}
