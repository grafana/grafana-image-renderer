package service_test

import (
	"runtime/debug"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/grafana/grafana-image-renderer/pkg/service"
	"github.com/stretchr/testify/require"
)

func TestPrettyVersion(t *testing.T) {
	t.Parallel()
	if !isGitVcs(t) {
		t.Skip("skipping test, not built with -buildvcs")
	}

	svc := service.NewVersionService()
	version := svc.GetPrettyVersion()
	require.NotEmpty(t, version, "expected service version to be non-empty")
}

func TestRenderVersion(t *testing.T) {
	t.Parallel()

	svc := service.NewVersionService()
	version := svc.GetRenderVersion()
	_, err := semver.NewVersion(version)
	require.NoError(t, err, "expected render version to be valid semver")
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
