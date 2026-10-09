package service

import (
	"context"
	"sync/atomic"
)

const (
	ReadinessModeBinding = "binding"
	ReadinessModeLegacy  = "legacy"
	ReadinessModeUnknown = "unknown"
)

type readinessModeContextKey struct{}

// ReadinessModeTracker records which readiness detection mode was used for a
// render request so HTTP/render metrics can label observations by mode.
//
// TODO: Remove the mode label from metrics (and this tracker) once a default
// readiness mode has been decided and the temporary dual-mode rollout ends.
type ReadinessModeTracker struct {
	mode atomic.Value // string
}

// ContextWithReadinessModeTracker attaches a readiness mode tracker to ctx.
// Callers should mutate the returned tracker (or use SetReadinessMode) during
// the request, then read the final value when recording metrics.
func ContextWithReadinessModeTracker(ctx context.Context) (context.Context, *ReadinessModeTracker) {
	t := &ReadinessModeTracker{}
	t.mode.Store(ReadinessModeUnknown)
	return context.WithValue(ctx, readinessModeContextKey{}, t), t
}

// SetReadinessMode records the readiness mode on the tracker in ctx, if any.
func SetReadinessMode(ctx context.Context, mode string) {
	if t, ok := ctx.Value(readinessModeContextKey{}).(*ReadinessModeTracker); ok && t != nil {
		t.Set(mode)
	}
}

// ReadinessModeFromContext returns the readiness mode recorded on ctx, or
// ReadinessModeUnknown if none was set.
func ReadinessModeFromContext(ctx context.Context) string {
	if t, ok := ctx.Value(readinessModeContextKey{}).(*ReadinessModeTracker); ok && t != nil {
		return t.Get()
	}
	return ReadinessModeUnknown
}

func (t *ReadinessModeTracker) Set(mode string) {
	if t == nil {
		return
	}
	t.mode.Store(mode)
}

func (t *ReadinessModeTracker) Get() string {
	if t == nil {
		return ReadinessModeUnknown
	}
	v, _ := t.mode.Load().(string)
	if v == "" {
		return ReadinessModeUnknown
	}
	return v
}
