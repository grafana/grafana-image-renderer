package acceptance

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestChromiumInstalled(t *testing.T) {
	LongTest(t)
	t.Parallel()

	exitCode, logs := RunImageRendererWithCommand(t, []string{"/usr/lib/chromium/chromium", "--version"}, nil)
	require.Zero(t, exitCode, "Chromium did not exist in the container (--version failed)")
	require.Regexp(t, `(?m)^Chromium [0-9]+(?:\.[0-9]+)+(?:\s|$)`, logs, "chromium --version did not return expected output")
}
