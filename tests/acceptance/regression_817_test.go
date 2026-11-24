package acceptance

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRegression817(t *testing.T) {
	LongTest(t)
	t.Parallel()

	for font, families := range map[string][]string{
		"Inter":      {"Inter"},
		"Helvetica":  {"Nimbus Sans", "Liberation Sans"},
		"sans-serif": {"DejaVu Sans"},
	} {
		t.Run(fmt.Sprintf("font %q resolves to a compatible family", font), func(t *testing.T) {
			t.Parallel()

			exitCode, logs := RunImageRendererWithCommand(t, []string{"fc-match", "--format=%{family}", font}, nil)
			require.Zero(t, exitCode, "font %q not found in the container (fc-match failed)", font)
			require.Contains(t, families, logs, "font family mismatched")
		})
	}
}
