package service_test

import (
	"context"
	"testing"

	"github.com/grafana/grafana-image-renderer/pkg/service"
	"github.com/stretchr/testify/require"
)

func TestReadinessModeTracker_DefaultsToUnknown(t *testing.T) {
	t.Parallel()

	require.Equal(t, service.ReadinessModeUnknown, service.ReadinessModeFromContext(t.Context()))

	ctx, tracker := service.ContextWithReadinessModeTracker(t.Context())
	require.Equal(t, service.ReadinessModeUnknown, tracker.Get())
	require.Equal(t, service.ReadinessModeUnknown, service.ReadinessModeFromContext(ctx))
}

func TestReadinessModeTracker_SetAndGet(t *testing.T) {
	t.Parallel()

	ctx, tracker := service.ContextWithReadinessModeTracker(t.Context())
	service.SetReadinessMode(ctx, service.ReadinessModeBinding)

	require.Equal(t, service.ReadinessModeBinding, tracker.Get())
	require.Equal(t, service.ReadinessModeBinding, service.ReadinessModeFromContext(ctx))

	service.SetReadinessMode(ctx, service.ReadinessModeLegacy)
	require.Equal(t, service.ReadinessModeLegacy, service.ReadinessModeFromContext(ctx))
}

func TestSetReadinessMode_NoTrackerIsNoop(t *testing.T) {
	t.Parallel()

	service.SetReadinessMode(context.Background(), service.ReadinessModeBinding)
	require.Equal(t, service.ReadinessModeUnknown, service.ReadinessModeFromContext(context.Background()))
}
