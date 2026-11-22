package acceptance

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestChromiumInstalled(t *testing.T) {
	LongTest(t)
	t.Parallel()

	exitCode, logs := RunImageRendererWithCommand(t, []string{"chromium", "--version"}, nil)
	require.Zero(t, exitCode, "chromium did not exist in the container (--version failed)")
	require.Regexp(t, `^Chromium [0-9]+`, logs, "chromium --version did not return expected output")
	if version := os.Getenv("EXPECTED_CHROMIUM_VERSION"); version != "" {
		require.Equal(t, version, strings.Fields(logs)[1], "Chromium version in the built image")
	}
}
