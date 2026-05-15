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

	// Debian and Alpine ship different files for the same logical font:
	//   - Debian's fonts-inter ships Inter-Regular.otf; Alpine's font-inter ships Inter.ttc (v4.1+).
	//   - Debian's fonts-urw-base35 ships a Helvetica -> Nimbus Sans alias; Alpine's font-urw-base35
	//     does not, so Helvetica falls back to the default sans-serif (DejaVuSans).
	// Accept any of the variant-specific filenames so the same test covers both runtime images.
	for font, fileNames := range map[string][]string{
		"Inter":      {"Inter-Regular.otf", "Inter.ttc"},
		"Helvetica":  {"NimbusSans-Regular.otf", "DejaVuSans.ttf"},
		"sans-serif": {"DejaVuSans.ttf"},
	} {
		t.Run(fmt.Sprintf("font %q is provided by one of %v", font, fileNames), func(t *testing.T) {
			t.Parallel()

			// use fc-match ${font}
			exitCode, logs := RunImageRendererWithCommand(t, []string{"fc-match", font}, []string{})
			require.Zero(t, exitCode, "font %q not found in the container (fc-match failed)", font)
			require.NotEmpty(t, logs, "font %q produced no output", font)
			require.Zero(t, strings.Count(logs, "\n"), "should only contain 1 line")
			actual, _, ok := strings.Cut(logs, ":")
			require.True(t, ok, "fc-match output format unexpected for font %q: %s", font, logs)
			require.Contains(t, fileNames, actual, "file-name not in the accepted set for font %q", font)
		})
	}
}
