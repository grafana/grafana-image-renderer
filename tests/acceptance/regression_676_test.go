package acceptance

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Regression test for: https://github.com/grafana/grafana-image-renderer/issues/676
func TestRegression676(t *testing.T) {
	LongTest(t)
	t.Parallel()

	for tool, cmd := range map[string]string{
		"openssl tools":          "openssl version",
		"update-ca-certificates": "command -v update-ca-certificates",
		"certutil":               "command -v certutil",
	} {
		t.Run("image contains: "+tool, func(t *testing.T) {
			t.Parallel()

			exitCode, logs := RunImageRendererWithCommand(t, []string{"sh", "-c"}, []string{cmd})
			require.Zero(t, exitCode, "%q did not exist in the container (command failed)", tool)
			require.NotEmpty(t, logs, "%q produced no output", tool)
		})
	}
}
