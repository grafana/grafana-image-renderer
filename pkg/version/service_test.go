package version_test

import (
	"runtime/debug"
	"testing"

	"github.com/grafana/grafana-image-renderer/pkg/version"
	"github.com/stretchr/testify/require"
)

func TestServiceVersion(t *testing.T) {
	t.Parallel()
	if !isGitVcs(t) {
		t.Skip("skipping test, not built with -buildvcs")
	}

	version := version.ServiceVersion()
	require.NotEmpty(t, version, "expected service version to be non-empty")
}

func isGitVcs(tb testing.TB) bool {
	tb.Helper()

	info, ok := debug.ReadBuildInfo()
	if !ok {
		return false
	}

	for _, setting := range info.Settings {
		if setting.Key == "vcs" {
			return setting.Value == "git"
		}
	}
	return false
}
