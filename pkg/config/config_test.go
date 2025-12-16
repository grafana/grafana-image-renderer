package config

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

func noopSpan() trace.Span {
	tracer := noop.NewTracerProvider().Tracer("")
	_, span := tracer.Start(context.Background(), "test")
	return span
}

func TestRenderRequestConfig(t *testing.T) {
	defaultConfig := RequestConfig{
		TimeBetweenScrolls:              50 * time.Millisecond,
		MinWidth:                        1000,
		MinHeight:                       500,
		MaxWidth:                        3000,
		MaxHeight:                       3000,
		PageScaleFactor:                 1.0,
		Landscape:                       true,
		ReadinessTimeout:                30 * time.Second,
		ReadinessIterationInterval:      100 * time.Millisecond,
		ReadinessWaitForNQueryCycles:    1,
		ReadinessPriorWait:              1 * time.Second,
		ReadinessDisableQueryWait:       false,
		ReadinessFirstQueryTimeout:      3 * time.Second,
		ReadinessQueriesTimeout:         0,
		ReadinessDisableNetworkWait:     false,
		ReadinessNetworkIdleTimeout:     0,
		ReadinessDisableDOMHashCodeWait: false,
		ReadinessDOMHashCodeTimeout:     0,
	}

	t.Run("no url match found", func(t *testing.T) {
		browserConfig := &BrowserConfig{
			DefaultRequestConfig: defaultConfig,
			requestConfigOverrides: map[string]map[string]any{
				"^https://example\\.com/.*$": {
					"min-width": 1200,
				},
			},
		}

		result := browserConfig.RenderRequestConfig(noopSpan(), "https://other-domain.com/dashboard")

		assert.Equal(t, defaultConfig, result)
	})

	t.Run("one url match found", func(t *testing.T) {
		browserConfig := &BrowserConfig{
			DefaultRequestConfig: defaultConfig,
			requestConfigOverrides: map[string]map[string]any{
				"^https://example\\.com/.*$": {
					"min-width":  1200,
					"min-height": 800,
				},
			},
		}

		result := browserConfig.RenderRequestConfig(noopSpan(), "https://example.com/dashboard")

		assert.Equal(t, 1200, result.MinWidth)
		assert.Equal(t, 800, result.MinHeight)
		// Other fields should remain default
		assert.Equal(t, defaultConfig.MaxWidth, result.MaxWidth)
		assert.Equal(t, defaultConfig.TimeBetweenScrolls, result.TimeBetweenScrolls)
	})

	// TODO: Ideally we would have a test for multiple url matches, but Go maps don't guarantee iteration order, so we can't test this easily.
	t.Run("multiple url matches", func(t *testing.T) {
		t.Skip()
	})

	// test all properties overridden
	t.Run("all properties overridden", func(t *testing.T) {
		browserConfig := &BrowserConfig{
			DefaultRequestConfig: defaultConfig,
			requestConfigOverrides: map[string]map[string]any{
				".*": {
					"time-between-scrolls":                "100ms",
					"min-width":                           1200,
					"min-height":                          800,
					"max-width":                           4000,
					"max-height":                          4000,
					"page-scale-factor":                   2.0,
					"landscape":                           false,
					"readiness-timeout":                   "60s",
					"readiness-iteration-interval":        "200ms",
					"readiness-wait-for-n-query-cycles":   3,
					"readiness-prior-wait":                "2s",
					"readiness-disable-query-wait":        true,
					"readiness-first-query-timeout":       "5s",
					"readiness-queries-timeout":           "10s",
					"readiness-disable-network-wait":      true,
					"readiness-network-idle-timeout":      "5s",
					"readiness-disable-dom-hashcode-wait": true,
					"readiness-dom-hashcode-timeout":      "3s",
				},
			},
		}

		result := browserConfig.RenderRequestConfig(noopSpan(), "https://example.com/dashboard")

		assert.Equal(t, 100*time.Millisecond, result.TimeBetweenScrolls)
		assert.Equal(t, 1200, result.MinWidth)
		assert.Equal(t, 800, result.MinHeight)
		assert.Equal(t, 4000, result.MaxWidth)
		assert.Equal(t, 4000, result.MaxHeight)
		assert.Equal(t, 2.0, result.PageScaleFactor)
		assert.Equal(t, false, result.Landscape)
		assert.Equal(t, 60*time.Second, result.ReadinessTimeout)
		assert.Equal(t, 200*time.Millisecond, result.ReadinessIterationInterval)
		assert.Equal(t, 3, result.ReadinessWaitForNQueryCycles)
		assert.Equal(t, 2*time.Second, result.ReadinessPriorWait)
		assert.Equal(t, true, result.ReadinessDisableQueryWait)
		assert.Equal(t, 5*time.Second, result.ReadinessFirstQueryTimeout)
		assert.Equal(t, 10*time.Second, result.ReadinessQueriesTimeout)
		assert.Equal(t, true, result.ReadinessDisableNetworkWait)
		assert.Equal(t, 5*time.Second, result.ReadinessNetworkIdleTimeout)
		assert.Equal(t, true, result.ReadinessDisableDOMHashCodeWait)
		assert.Equal(t, 3*time.Second, result.ReadinessDOMHashCodeTimeout)
	})

	t.Run("only some properties overridden", func(t *testing.T) {
		browserConfig := &BrowserConfig{
			DefaultRequestConfig: defaultConfig,
			requestConfigOverrides: map[string]map[string]any{
				".*": {
					"min-width":         1200,
					"readiness-timeout": "45s",
				},
			},
		}

		result := browserConfig.RenderRequestConfig(noopSpan(), "https://example.com/dashboard")

		// Overridden values
		assert.Equal(t, 1200, result.MinWidth)
		assert.Equal(t, 45*time.Second, result.ReadinessTimeout)

		// Non-overridden values should remain default
		assert.Equal(t, defaultConfig.MinHeight, result.MinHeight)
		assert.Equal(t, defaultConfig.MaxWidth, result.MaxWidth)
		assert.Equal(t, defaultConfig.MaxHeight, result.MaxHeight)
		assert.Equal(t, defaultConfig.TimeBetweenScrolls, result.TimeBetweenScrolls)
		assert.Equal(t, defaultConfig.PageScaleFactor, result.PageScaleFactor)
		assert.Equal(t, defaultConfig.Landscape, result.Landscape)
		assert.Equal(t, defaultConfig.ReadinessIterationInterval, result.ReadinessIterationInterval)
	})

	t.Run("invalid duration values fall back to default", func(t *testing.T) {
		browserConfig := &BrowserConfig{
			DefaultRequestConfig: defaultConfig,
			requestConfigOverrides: map[string]map[string]any{
				".*": {
					"time-between-scrolls": "invalid-duration",
					"readiness-timeout":    "not-a-duration",
				},
			},
		}

		result := browserConfig.RenderRequestConfig(noopSpan(), "https://example.com/dashboard")

		// Invalid duration values should fall back to defaults
		assert.Equal(t, defaultConfig.TimeBetweenScrolls, result.TimeBetweenScrolls)
		assert.Equal(t, defaultConfig.ReadinessTimeout, result.ReadinessTimeout)
	})

	t.Run("regex pattern matching", func(t *testing.T) {
		browserConfig := &BrowserConfig{
			DefaultRequestConfig: defaultConfig,
			requestConfigOverrides: map[string]map[string]any{
				"^https://grafana\\.example\\.com/d/[a-zA-Z0-9]+/.*$": {
					"min-width": 1400,
				},
			},
		}

		// Should match
		result := browserConfig.RenderRequestConfig(noopSpan(), "https://grafana.example.com/d/abc123/my-dashboard")
		assert.Equal(t, 1400, result.MinWidth)

		// Should not match (different path structure)
		result = browserConfig.RenderRequestConfig(noopSpan(), "https://grafana.example.com/explore")
		assert.Equal(t, defaultConfig.MinWidth, result.MinWidth)
	})

	t.Run("empty overrides returns default", func(t *testing.T) {
		browserConfig := &BrowserConfig{
			DefaultRequestConfig:   defaultConfig,
			requestConfigOverrides: map[string]map[string]any{},
		}

		result := browserConfig.RenderRequestConfig(noopSpan(), "https://example.com/dashboard")

		assert.Equal(t, defaultConfig, result)
	})

	t.Run("nil overrides returns default", func(t *testing.T) {
		browserConfig := &BrowserConfig{
			DefaultRequestConfig:   defaultConfig,
			requestConfigOverrides: nil,
		}

		result := browserConfig.RenderRequestConfig(noopSpan(), "https://example.com/dashboard")

		assert.Equal(t, defaultConfig, result)
	})

	t.Run("unsupported key is ignored", func(t *testing.T) {
		browserConfig := &BrowserConfig{
			DefaultRequestConfig: defaultConfig,
			requestConfigOverrides: map[string]map[string]any{
				".*": {
					"unsupported-key": "some-value",
					"min-width":       1200,
				},
			},
		}

		result := browserConfig.RenderRequestConfig(noopSpan(), "https://example.com/dashboard")

		// Valid override should still work
		assert.Equal(t, 1200, result.MinWidth)
		// Everything else should be default
		assert.Equal(t, defaultConfig.MinHeight, result.MinHeight)
	})

	t.Run("invalid integer type falls back to default", func(t *testing.T) {
		browserConfig := &BrowserConfig{
			DefaultRequestConfig: defaultConfig,
			requestConfigOverrides: map[string]map[string]any{
				".*": {
					"min-width":  "not-an-int",
					"min-height": true,
					"max-width":  []int{1, 2, 3},
				},
			},
		}

		result := browserConfig.RenderRequestConfig(noopSpan(), "https://example.com/dashboard")

		assert.Equal(t, defaultConfig.MinWidth, result.MinWidth)
		assert.Equal(t, defaultConfig.MinHeight, result.MinHeight)
		assert.Equal(t, defaultConfig.MaxWidth, result.MaxWidth)
	})

	t.Run("invalid boolean type falls back to default", func(t *testing.T) {
		browserConfig := &BrowserConfig{
			DefaultRequestConfig: defaultConfig,
			requestConfigOverrides: map[string]map[string]any{
				".*": {
					"landscape":                    "not-a-bool",
					"readiness-disable-query-wait": 123,
				},
			},
		}

		result := browserConfig.RenderRequestConfig(noopSpan(), "https://example.com/dashboard")

		assert.Equal(t, defaultConfig.Landscape, result.Landscape)
		assert.Equal(t, defaultConfig.ReadinessDisableQueryWait, result.ReadinessDisableQueryWait)
	})

	t.Run("invalid float type falls back to default", func(t *testing.T) {
		browserConfig := &BrowserConfig{
			DefaultRequestConfig: defaultConfig,
			requestConfigOverrides: map[string]map[string]any{
				".*": {
					"page-scale-factor": "not-a-float",
				},
			},
		}

		result := browserConfig.RenderRequestConfig(noopSpan(), "https://example.com/dashboard")

		assert.Equal(t, defaultConfig.PageScaleFactor, result.PageScaleFactor)
	})

	t.Run("duration with wrong type falls back to default", func(t *testing.T) {
		browserConfig := &BrowserConfig{
			DefaultRequestConfig: defaultConfig,
			requestConfigOverrides: map[string]map[string]any{
				".*": {
					"time-between-scrolls": 100,
					"readiness-timeout":    true,
				},
			},
		}

		result := browserConfig.RenderRequestConfig(noopSpan(), "https://example.com/dashboard")

		assert.Equal(t, defaultConfig.TimeBetweenScrolls, result.TimeBetweenScrolls)
		assert.Equal(t, defaultConfig.ReadinessTimeout, result.ReadinessTimeout)
	})

	t.Run("JSON-unmarshaled float64 works for int fields", func(t *testing.T) {
		// JSON unmarshaling into map[string]any produces float64 for all numbers
		browserConfig := &BrowserConfig{
			DefaultRequestConfig: defaultConfig,
			requestConfigOverrides: map[string]map[string]any{
				".*": {
					"min-width":  float64(1200),
					"min-height": float64(800),
				},
			},
		}

		result := browserConfig.RenderRequestConfig(noopSpan(), "https://example.com/dashboard")

		assert.Equal(t, 1200, result.MinWidth)
		assert.Equal(t, 800, result.MinHeight)
	})
}
