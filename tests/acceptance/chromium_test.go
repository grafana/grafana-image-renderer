package acceptance

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestChromiumInstalled(t *testing.T) {
	LongTest(t)
	t.Parallel()

	exitCode, logs := RunImageRendererWithCommand(t, []string{"chromium", "--version"}, nil)
	require.Zero(t, exitCode, "chromium did not exist in the container (--version failed)")
	require.Regexp(t, `^Chromium [0-9]+`, logs, "chromium --version did not return expected output")
}
