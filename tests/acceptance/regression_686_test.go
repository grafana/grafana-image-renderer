package acceptance

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
)

// Regression test for: https://github.com/grafana/grafana-image-renderer/issues/686
func TestRegression686(t *testing.T) {
	LongTest(t)
	t.Parallel()

	t.Run("kubernetes behaviour: user is set to numeric UID", func(t *testing.T) {
		t.Parallel()

		client, err := testcontainers.NewDockerClientWithOpts(t.Context())
		require.NoError(t, err, "could not create Docker client")

		resp, err := client.ImageInspect(t.Context(), GetDockerImage(t))
		require.NoError(t, err, "could not inspect image")

		uid, err := strconv.Atoi(resp.Config.User)
		require.NoError(t, err, "could not parse image user as integer")
		require.Greater(t, uid, 0, "image user UID should be greater than 0")

		t.Run("UID is same as nonroot user", func(t *testing.T) {
			t.Parallel()

			exitCode, logs := RunImageRendererWithCommand(t, []string{"id", "-u"}, nil,
				WithUser("nonroot"))
			require.Zero(t, exitCode, "could not run id -u as nonroot user")
			require.Equal(t, strconv.Itoa(uid), logs, "nonroot user UID does not match image user UID")
		})
	})
}
