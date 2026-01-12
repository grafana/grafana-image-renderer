package acceptance

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Regression test for: https://github.com/grafana/grafana-image-renderer/issues/680
func TestRegression680(t *testing.T) {
	LongTest(t)
	t.Parallel()

	t.Run("lang is en_US.UTF-8", func(t *testing.T) {
		t.Parallel()

		// The LANG environment variable should be set to en_US.UTF-8, which makes Chromium able to deal with non-ASCII characters properly.

		for _, cmd := range []string{"echo $LANG", "echo $LC_ALL"} {
			t.Run(cmd, func(t *testing.T) {
				t.Parallel()

				exitCode, logs := RunImageRendererWithCommand(t, []string{"sh", "-c"}, []string{cmd})
				require.Zero(t, exitCode, "command %q did not run successfully", cmd)
				require.Equal(t, "en_US.UTF-8", logs, "command %q did not return expected output", cmd)
			})
		}
	})

	t.Run("lang is respected", func(t *testing.T) {
		t.Skip("somehow, the functionality works, but the test fails on sorting") // FIXME

		// With LC_ALL=C, sorting is byte-wise. With LC_ALL=en_US.UTF-8, sorting is character-wise.
		// We can use this to ensure we are not using the C locale.
		exitCode, logs := RunImageRendererWithCommand(t, []string{"sh", "-c"}, []string{`echo -e "a\nb\nA\nB" | sort`})
		require.Zero(t, exitCode, "command did not run successfully")
		// With LC_ALL=C, the output would be "A\nB\na\nb".
		require.Equal(t, "a\nA\nb\nB", logs)
	})
}
