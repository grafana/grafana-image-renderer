package acceptance

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRegression817(t *testing.T) {
	LongTest(t)
	t.Parallel()

	for font, fileName := range map[string]string{
		"Inter":      "Inter-Regular.otf",
		"Helvetica":  "NimbusSans-Regular.otf",
		"sans-serif": "DejaVuSans.ttf",
	} {
		t.Run(fmt.Sprintf("font %q is provided by %q", font, fileName), func(t *testing.T) {
			t.Parallel()

			// use fc-match ${font}
			exitCode, logs := RunImageRendererWithCommand(t, []string{"fc-match", font}, []string{})
			require.Zero(t, exitCode, "font %q not found in the container (fc-match failed)", font)
			require.NotEmpty(t, logs, "font %q produced no output", font)
			require.Zero(t, strings.Count(logs, "\n"), "should only contain 1 line")
			actual, _, ok := strings.Cut(logs, ":")
			require.True(t, ok, "fc-match output format unexpected for font %q: %s", font, logs)
			require.Equal(t, fileName, actual, "file-name mismatched")
		})
	}
}
