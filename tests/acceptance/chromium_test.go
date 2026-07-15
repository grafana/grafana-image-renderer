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

func TestChromiumPackageMetadataInstalled(t *testing.T) {
	LongTest(t)
	t.Parallel()

	const findChromiumPackage = `
for status in /var/lib/dpkg/status /var/lib/dpkg/status.d/*; do
	if [ -f "$status" ] && grep -q '^Package: chromium$' "$status"; then
		exit 0
	fi
done
echo 'package metadata not found: chromium'
exit 1
`

	exitCode, logs := RunImageRendererWithCommand(t, []string{"sh", "-c"}, []string{findChromiumPackage})
	require.Zero(t, exitCode, "Chromium package metadata did not exist in the container: %s", logs)
}
