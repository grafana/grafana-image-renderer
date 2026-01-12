package config

import (
	"context"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v3"
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
		overrideConfig := defaultConfig
		overrideConfig.MinWidth = 1200

		browserConfig := &BrowserConfig{
			DefaultRequestConfig: defaultConfig,
			RequestConfigOverrides: []RequestConfigOverride{
				{Pattern: regexp.MustCompile(`^https://example\.com/.*$`), Config: overrideConfig},
			},
		}

		result := browserConfig.LookupRequestConfig(noopSpan(), "https://other-domain.com/dashboard")

		assert.Equal(t, defaultConfig, result)
	})

	t.Run("one url match found", func(t *testing.T) {
		overrideConfig := defaultConfig
		overrideConfig.MinWidth = 1200
		overrideConfig.MinHeight = 800

		browserConfig := &BrowserConfig{
			DefaultRequestConfig: defaultConfig,
			RequestConfigOverrides: []RequestConfigOverride{
				{Pattern: regexp.MustCompile(`^https://example\.com/.*$`), Config: overrideConfig},
			},
		}

		result := browserConfig.LookupRequestConfig(noopSpan(), "https://example.com/dashboard")

		assert.Equal(t, 1200, result.MinWidth)
		assert.Equal(t, 800, result.MinHeight)
		// Other fields should remain from the override config (which was copied from default)
		assert.Equal(t, defaultConfig.MaxWidth, result.MaxWidth)
		assert.Equal(t, defaultConfig.TimeBetweenScrolls, result.TimeBetweenScrolls)
	})

	// TODO: Ideally we would have a test for multiple url matches, but Go maps don't guarantee iteration order, so we can't test this easily.
	t.Run("multiple url matches", func(t *testing.T) {
		t.Skip()
	})

	t.Run("all properties overridden", func(t *testing.T) {
		overrideConfig := RequestConfig{
			TimeBetweenScrolls:              100 * time.Millisecond,
			MinWidth:                        1200,
			MinHeight:                       800,
			MaxWidth:                        4000,
			MaxHeight:                       4000,
			PageScaleFactor:                 2.0,
			Landscape:                       false,
			ReadinessTimeout:                60 * time.Second,
			ReadinessIterationInterval:      200 * time.Millisecond,
			ReadinessWaitForNQueryCycles:    3,
			ReadinessPriorWait:              2 * time.Second,
			ReadinessDisableQueryWait:       true,
			ReadinessFirstQueryTimeout:      5 * time.Second,
			ReadinessQueriesTimeout:         10 * time.Second,
			ReadinessDisableNetworkWait:     true,
			ReadinessNetworkIdleTimeout:     5 * time.Second,
			ReadinessDisableDOMHashCodeWait: true,
			ReadinessDOMHashCodeTimeout:     3 * time.Second,
		}

		browserConfig := &BrowserConfig{
			DefaultRequestConfig: defaultConfig,
			RequestConfigOverrides: []RequestConfigOverride{
				{Pattern: regexp.MustCompile(".*"), Config: overrideConfig},
			},
		}

		result := browserConfig.LookupRequestConfig(noopSpan(), "https://example.com/dashboard")

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
		overrideConfig := defaultConfig
		overrideConfig.MinWidth = 1200
		overrideConfig.ReadinessTimeout = 45 * time.Second

		browserConfig := &BrowserConfig{
			DefaultRequestConfig: defaultConfig,
			RequestConfigOverrides: []RequestConfigOverride{
				{Pattern: regexp.MustCompile(".*"), Config: overrideConfig},
			},
		}

		result := browserConfig.LookupRequestConfig(noopSpan(), "https://example.com/dashboard")

		// Overridden values
		assert.Equal(t, 1200, result.MinWidth)
		assert.Equal(t, 45*time.Second, result.ReadinessTimeout)

		// Non-overridden values should remain default (since we copied from default)
		assert.Equal(t, defaultConfig.MinHeight, result.MinHeight)
		assert.Equal(t, defaultConfig.MaxWidth, result.MaxWidth)
		assert.Equal(t, defaultConfig.MaxHeight, result.MaxHeight)
		assert.Equal(t, defaultConfig.TimeBetweenScrolls, result.TimeBetweenScrolls)
		assert.Equal(t, defaultConfig.PageScaleFactor, result.PageScaleFactor)
		assert.Equal(t, defaultConfig.Landscape, result.Landscape)
		assert.Equal(t, defaultConfig.ReadinessIterationInterval, result.ReadinessIterationInterval)
	})

	t.Run("regex pattern matching", func(t *testing.T) {
		overrideConfig := defaultConfig
		overrideConfig.MinWidth = 1400

		browserConfig := &BrowserConfig{
			DefaultRequestConfig: defaultConfig,
			RequestConfigOverrides: []RequestConfigOverride{
				{Pattern: regexp.MustCompile(`^https://grafana\.example\.com/d/[a-zA-Z0-9]+/.*$`), Config: overrideConfig},
			},
		}

		// Should match
		result := browserConfig.LookupRequestConfig(noopSpan(), "https://grafana.example.com/d/abc123/my-dashboard")
		assert.Equal(t, 1400, result.MinWidth)

		// Should not match (different path structure)
		result = browserConfig.LookupRequestConfig(noopSpan(), "https://grafana.example.com/explore")
		assert.Equal(t, defaultConfig.MinWidth, result.MinWidth)
	})

	t.Run("empty overrides returns default", func(t *testing.T) {
		browserConfig := &BrowserConfig{
			DefaultRequestConfig:   defaultConfig,
			RequestConfigOverrides: []RequestConfigOverride{},
		}

		result := browserConfig.LookupRequestConfig(noopSpan(), "https://example.com/dashboard")

		assert.Equal(t, defaultConfig, result)
	})

	t.Run("nil overrides returns default", func(t *testing.T) {
		browserConfig := &BrowserConfig{
			DefaultRequestConfig:   defaultConfig,
			RequestConfigOverrides: nil,
		}

		result := browserConfig.LookupRequestConfig(noopSpan(), "https://example.com/dashboard")

		assert.Equal(t, defaultConfig, result)
	})
}

// TestBrowserOverrideFlag tests the eager config building with --browser.override flag
func TestBrowserOverrideFlag(t *testing.T) {
	t.Parallel()

	// Create a test command with browser flags
	testCmd := &cli.Command{
		Flags: BrowserFlags(),
		Action: func(ctx context.Context, c *cli.Command) error {
			return nil
		},
		Reader:    nopReader{},
		Writer:    nopWriter{},
		ErrWriter: nopWriter{},
	}

	t.Run("basic override", func(t *testing.T) {
		var browserConfig BrowserConfig
		var parseErr error

		cmd := &cli.Command{
			Flags: testCmd.Flags,
			Action: func(ctx context.Context, c *cli.Command) error {
				browserConfig, parseErr = BrowserConfigFromCommand(c)
				return parseErr
			},
			Reader:    nopReader{},
			Writer:    nopWriter{},
			ErrWriter: nopWriter{},
		}

		err := cmd.Run(t.Context(), []string{
			"",
			"--browser.readiness.timeout=15s",
			"--browser.override=^https://slow\\.example\\.com/.*=--browser.readiness.timeout=60s",
		})
		require.NoError(t, err)
		require.NoError(t, parseErr)

		// Default config should have 15s timeout
		assert.Equal(t, 15*time.Second, browserConfig.DefaultRequestConfig.ReadinessTimeout)

		// Override config for slow.example.com should have 60s timeout
		require.Len(t, browserConfig.RequestConfigOverrides, 1, "expected one override config")
		override := browserConfig.RequestConfigOverrides[0]
		assert.Equal(t, "^https://slow\\.example\\.com/.*", override.Pattern.String())
		assert.Equal(t, 60*time.Second, override.Config.ReadinessTimeout)

		// RenderRequestConfig should return the override for matching URLs
		result := browserConfig.LookupRequestConfig(noopSpan(), "https://slow.example.com/dashboard")
		assert.Equal(t, 60*time.Second, result.ReadinessTimeout)

		// RenderRequestConfig should return the default for non-matching URLs
		result = browserConfig.LookupRequestConfig(noopSpan(), "https://fast.example.com/dashboard")
		assert.Equal(t, 15*time.Second, result.ReadinessTimeout)
	})

	t.Run("multiple overrides", func(t *testing.T) {
		var browserConfig BrowserConfig
		var parseErr error

		cmd := &cli.Command{
			Flags: testCmd.Flags,
			Action: func(ctx context.Context, c *cli.Command) error {
				browserConfig, parseErr = BrowserConfigFromCommand(c)
				return parseErr
			},
			Reader:    nopReader{},
			Writer:    nopWriter{},
			ErrWriter: nopWriter{},
		}

		err := cmd.Run(t.Context(), []string{
			"",
			"--browser.min-width=1000",
			"--browser.override=^https://wide\\.example\\.com/.*=--browser.min-width=1920",
			"--browser.override=^https://slow\\.example\\.com/.*=--browser.readiness.timeout=120s",
		})
		require.NoError(t, err)
		require.NoError(t, parseErr)

		// Should have two override configs
		assert.Len(t, browserConfig.RequestConfigOverrides, 2)

		// Check wide override (first one)
		wideOverride := browserConfig.RequestConfigOverrides[0]
		assert.Equal(t, "^https://wide\\.example\\.com/.*", wideOverride.Pattern.String())
		assert.Equal(t, 1920, wideOverride.Config.MinWidth)

		// Check slow override (second one)
		slowOverride := browserConfig.RequestConfigOverrides[1]
		assert.Equal(t, "^https://slow\\.example\\.com/.*", slowOverride.Pattern.String())
		assert.Equal(t, 120*time.Second, slowOverride.Config.ReadinessTimeout)
	})

	t.Run("override inherits base config", func(t *testing.T) {
		var browserConfig BrowserConfig
		var parseErr error

		cmd := &cli.Command{
			Flags: testCmd.Flags,
			Action: func(ctx context.Context, c *cli.Command) error {
				browserConfig, parseErr = BrowserConfigFromCommand(c)
				return parseErr
			},
			Reader:    nopReader{},
			Writer:    nopWriter{},
			ErrWriter: nopWriter{},
		}

		err := cmd.Run(t.Context(), []string{
			"",
			"--browser.min-width=1500",
			"--browser.readiness.timeout=20s",
			"--browser.override=^https://slow\\.example\\.com/.*=--browser.readiness.timeout=60s",
		})
		require.NoError(t, err)
		require.NoError(t, parseErr)

		// Override config should inherit min-width from base
		require.Len(t, browserConfig.RequestConfigOverrides, 1)
		override := browserConfig.RequestConfigOverrides[0]
		assert.Equal(t, "^https://slow\\.example\\.com/.*", override.Pattern.String())
		assert.Equal(t, 1500, override.Config.MinWidth, "override should inherit min-width from base")
		assert.Equal(t, 60*time.Second, override.Config.ReadinessTimeout, "override should have its own timeout")
	})

	t.Run("invalid regex pattern fails at startup", func(t *testing.T) {
		var parseErr error

		cmd := &cli.Command{
			Flags: testCmd.Flags,
			Action: func(ctx context.Context, c *cli.Command) error {
				_, parseErr = BrowserConfigFromCommand(c)
				return parseErr
			},
			Reader:    nopReader{},
			Writer:    nopWriter{},
			ErrWriter: nopWriter{},
		}

		err := cmd.Run(t.Context(), []string{
			"",
			"--browser.override=[invalid(regex=--browser.readiness.timeout=60s",
		})
		require.Error(t, err)
	})

	t.Run("invalid override format fails at startup", func(t *testing.T) {
		var parseErr error

		cmd := &cli.Command{
			Flags: testCmd.Flags,
			Action: func(ctx context.Context, c *cli.Command) error {
				_, parseErr = BrowserConfigFromCommand(c)
				return parseErr
			},
			Reader:    nopReader{},
			Writer:    nopWriter{},
			ErrWriter: nopWriter{},
		}

		err := cmd.Run(t.Context(), []string{
			"",
			"--browser.override=no-equals-sign-here",
		})
		require.Error(t, err)
	})

	t.Run("invalid flag value in override fails at startup", func(t *testing.T) {
		var parseErr error

		cmd := &cli.Command{
			Flags: testCmd.Flags,
			Action: func(ctx context.Context, c *cli.Command) error {
				_, parseErr = BrowserConfigFromCommand(c)
				return parseErr
			},
			Reader:    nopReader{},
			Writer:    nopWriter{},
			ErrWriter: nopWriter{},
		}

		err := cmd.Run(t.Context(), []string{
			"",
			"--browser.override=.*=--browser.readiness.timeout=not-a-duration",
		})
		require.Error(t, err)
	})

	t.Run("last value wins for scalar flags in override", func(t *testing.T) {
		// This test verifies the critical behavior that when the same flag appears
		// multiple times (base config + override), the last value wins.
		// This is how overrides work: base flags are reconstructed first,
		// then override flags are appended, so override values take precedence.
		var browserConfig BrowserConfig
		var parseErr error

		cmd := &cli.Command{
			Flags: testCmd.Flags,
			Action: func(ctx context.Context, c *cli.Command) error {
				browserConfig, parseErr = BrowserConfigFromCommand(c)
				return parseErr
			},
			Reader:    nopReader{},
			Writer:    nopWriter{},
			ErrWriter: nopWriter{},
		}

		err := cmd.Run(t.Context(), []string{
			"",
			"--browser.readiness.timeout=15s",
			"--browser.min-width=1000",
			"--browser.min-height=500",
			// Override changes timeout and min-width, but NOT min-height
			"--browser.override=.*=--browser.readiness.timeout=60s --browser.min-width=1920",
		})
		require.NoError(t, err)
		require.NoError(t, parseErr)

		// Base config should have original values
		assert.Equal(t, 15*time.Second, browserConfig.DefaultRequestConfig.ReadinessTimeout)
		assert.Equal(t, 1000, browserConfig.DefaultRequestConfig.MinWidth)
		assert.Equal(t, 500, browserConfig.DefaultRequestConfig.MinHeight)

		// Override config should have overridden values for timeout and min-width
		require.Len(t, browserConfig.RequestConfigOverrides, 1)
		override := browserConfig.RequestConfigOverrides[0]
		assert.Equal(t, ".*", override.Pattern.String())
		assert.Equal(t, 60*time.Second, override.Config.ReadinessTimeout, "override timeout should win (last value)")
		assert.Equal(t, 1920, override.Config.MinWidth, "override min-width should win (last value)")
		// min-height was NOT in the override, so it should inherit from base
		assert.Equal(t, 500, override.Config.MinHeight, "min-height should be inherited from base")
	})
}

// TestReconstructFlags tests that all flag values are reconstructed regardless of source,
// ensuring overrides properly inherit all base config values including defaults.
func TestReconstructFlags(t *testing.T) {
	t.Run("override only differs by explicitly overridden property", func(t *testing.T) {
		// Create a config file with only min-width set (page-scale-factor uses default)
		tempDir := t.TempDir()
		configFile := tempDir + "/config.yaml"
		configContent := `
browser:
  min-width: 1200
  override:
    - "^https://custom\\.example\\.com/.*=--browser.readiness.timeout=60s"
`
		require.NoError(t, os.WriteFile(configFile, []byte(configContent), 0644))

		oldDir, err := os.Getwd()
		require.NoError(t, err)
		require.NoError(t, os.Chdir(tempDir))
		defer func() { _ = os.Chdir(oldDir) }()

		var browserConfig BrowserConfig
		var parseErr error

		cmd := &cli.Command{
			Flags: BrowserFlags(),
			Action: func(ctx context.Context, c *cli.Command) error {
				browserConfig, parseErr = BrowserConfigFromCommand(c)
				return parseErr
			},
			Reader:    nopReader{},
			Writer:    nopWriter{},
			ErrWriter: nopWriter{},
		}

		err = cmd.Run(t.Context(), []string{""})
		require.NoError(t, err)
		require.NoError(t, parseErr)

		// Must have exactly one override
		require.Len(t, browserConfig.RequestConfigOverrides, 1, "expected one override config")
		override := browserConfig.RequestConfigOverrides[0]

		defaultCfg := browserConfig.DefaultRequestConfig
		overrideCfg := override.Config

		// The ONLY difference should be the readiness timeout (explicitly overridden)
		assert.Equal(t, 30*time.Second, defaultCfg.ReadinessTimeout, "default timeout should be 30s")
		assert.Equal(t, 60*time.Second, overrideCfg.ReadinessTimeout, "override timeout should be 60s")

		// All other properties should be identical (inherited from base config)
		assert.Equal(t, defaultCfg.MinWidth, overrideCfg.MinWidth,
			"override should inherit min-width from base")
		assert.Equal(t, defaultCfg.MinHeight, overrideCfg.MinHeight,
			"override should inherit min-height from base")
		assert.Equal(t, defaultCfg.MaxWidth, overrideCfg.MaxWidth,
			"override should inherit max-width from base")
		assert.Equal(t, defaultCfg.MaxHeight, overrideCfg.MaxHeight,
			"override should inherit max-height from base")
		assert.Equal(t, defaultCfg.PageScaleFactor, overrideCfg.PageScaleFactor,
			"override should inherit page-scale-factor from base (even though it's a default)")
		assert.Equal(t, defaultCfg.Landscape, overrideCfg.Landscape,
			"override should inherit landscape from base")
		assert.Equal(t, defaultCfg.TimeBetweenScrolls, overrideCfg.TimeBetweenScrolls,
			"override should inherit time-between-scrolls from base")
		assert.Equal(t, defaultCfg.ReadinessIterationInterval, overrideCfg.ReadinessIterationInterval,
			"override should inherit readiness iteration interval from base")
		assert.Equal(t, defaultCfg.ReadinessWaitForNQueryCycles, overrideCfg.ReadinessWaitForNQueryCycles,
			"override should inherit readiness wait-for-n-query-cycles from base")
		assert.Equal(t, defaultCfg.ReadinessPriorWait, overrideCfg.ReadinessPriorWait,
			"override should inherit readiness prior-wait from base")
		assert.Equal(t, defaultCfg.ReadinessDisableQueryWait, overrideCfg.ReadinessDisableQueryWait,
			"override should inherit readiness disable-query-wait from base")
		assert.Equal(t, defaultCfg.ReadinessFirstQueryTimeout, overrideCfg.ReadinessFirstQueryTimeout,
			"override should inherit readiness first-query-timeout from base")
		assert.Equal(t, defaultCfg.ReadinessQueriesTimeout, overrideCfg.ReadinessQueriesTimeout,
			"override should inherit readiness queries-timeout from base")
		assert.Equal(t, defaultCfg.ReadinessDisableNetworkWait, overrideCfg.ReadinessDisableNetworkWait,
			"override should inherit readiness disable-network-wait from base")
		assert.Equal(t, defaultCfg.ReadinessNetworkIdleTimeout, overrideCfg.ReadinessNetworkIdleTimeout,
			"override should inherit readiness network-idle-timeout from base")
		assert.Equal(t, defaultCfg.ReadinessDisableDOMHashCodeWait, overrideCfg.ReadinessDisableDOMHashCodeWait,
			"override should inherit readiness disable-dom-hashcode-wait from base")
		assert.Equal(t, defaultCfg.ReadinessDOMHashCodeTimeout, overrideCfg.ReadinessDOMHashCodeTimeout,
			"override should inherit readiness dom-hashcode-timeout from base")
	})
}

// TestOverrideWithOnlyPageScaleFactor is a regression test that verifies that when
// only page-scale-factor is set (to its default value), the override config inherits
// all other default values correctly.
func TestOverrideWithOnlyPageScaleFactor(t *testing.T) {
	var browserConfig BrowserConfig
	var parseErr error

	cmd := &cli.Command{
		Flags: BrowserFlags(),
		Action: func(ctx context.Context, c *cli.Command) error {
			browserConfig, parseErr = BrowserConfigFromCommand(c)
			return parseErr
		},
		Reader:    nopReader{},
		Writer:    nopWriter{},
		ErrWriter: nopWriter{},
	}

	err := cmd.Run(t.Context(), []string{
		"",
		"--browser.page-scale-factor=1.0",
		"--browser.override=grafana-with-override=--browser.page-scale-factor=1.0",
	})
	require.NoError(t, err)
	require.NoError(t, parseErr)

	// Default config should have default values
	assert.Equal(t, 1000, browserConfig.DefaultRequestConfig.MinWidth, "default MinWidth")
	assert.Equal(t, 500, browserConfig.DefaultRequestConfig.MinHeight, "default MinHeight")
	assert.Equal(t, 3000, browserConfig.DefaultRequestConfig.MaxWidth, "default MaxWidth")
	assert.Equal(t, 3000, browserConfig.DefaultRequestConfig.MaxHeight, "default MaxHeight")
	assert.Equal(t, 1.0, browserConfig.DefaultRequestConfig.PageScaleFactor, "default PageScaleFactor")

	// Override config should also have the SAME default values
	require.Len(t, browserConfig.RequestConfigOverrides, 1, "expected one override")
	override := browserConfig.RequestConfigOverrides[0]

	assert.Equal(t, browserConfig.DefaultRequestConfig.MinWidth, override.Config.MinWidth,
		"override should have same MinWidth as default")
	assert.Equal(t, browserConfig.DefaultRequestConfig.MinHeight, override.Config.MinHeight,
		"override should have same MinHeight as default")
	assert.Equal(t, browserConfig.DefaultRequestConfig.MaxWidth, override.Config.MaxWidth,
		"override should have same MaxWidth as default")
	assert.Equal(t, browserConfig.DefaultRequestConfig.MaxHeight, override.Config.MaxHeight,
		"override should have same MaxHeight as default")
	assert.Equal(t, browserConfig.DefaultRequestConfig.PageScaleFactor, override.Config.PageScaleFactor,
		"override should have same PageScaleFactor as default")
	assert.Equal(t, browserConfig.DefaultRequestConfig.TimeBetweenScrolls, override.Config.TimeBetweenScrolls,
		"override should have same TimeBetweenScrolls as default")
	assert.Equal(t, browserConfig.DefaultRequestConfig.Landscape, override.Config.Landscape,
		"override should have same Landscape as default")
}

// TestEagerConfigValidation tests that config validation happens at startup
func TestEagerConfigValidation(t *testing.T) {
	t.Parallel()

	// Create a command that exercises the eager config building
	cmd := &cli.Command{
		Flags: BrowserFlags(),
		Action: func(ctx context.Context, c *cli.Command) error {
			_, err := BrowserConfigFromCommand(c)
			return err
		},
		Reader:    nopReader{},
		Writer:    nopWriter{},
		ErrWriter: nopWriter{},
	}

	// Run with base config
	err := cmd.Run(t.Context(), []string{
		"",
		"--browser.readiness.timeout=15s",
	})
	require.NoError(t, err)

	// Run with override - validation should happen at startup, not request time
	err = cmd.Run(t.Context(), []string{
		"",
		"--browser.readiness.timeout=15s",
		"--browser.override=^https://example\\.com/.*=--browser.readiness.timeout=60s",
	})
	require.NoError(t, err)
}

func TestParseOverrideFlags(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "whitespace only",
			input:    "   ",
			expected: nil,
		},
		{
			name:     "single flag",
			input:    "--flag=value",
			expected: []string{"--flag=value"},
		},
		{
			name:     "two flags",
			input:    "--flag=value --flag2=value2",
			expected: []string{"--flag=value", "--flag2=value2"},
		},
		{
			name:     "multiple flags",
			input:    "--a=1 --b=2 --c=3",
			expected: []string{"--a=1", "--b=2", "--c=3"},
		},
		{
			name:     "flag with spaces in value",
			input:    "--flag=value with spaces --flag2=other",
			expected: []string{"--flag=value with spaces", "--flag2=other"},
		},
		{
			name:     "garbage before first flag is ignored",
			input:    "garbage --flag=value",
			expected: []string{"--flag=value"},
		},
		{
			name:     "leading whitespace",
			input:    "  --flag=value",
			expected: []string{"--flag=value"},
		},
		{
			name:     "trailing whitespace",
			input:    "--flag=value  ",
			expected: []string{"--flag=value"},
		},
		{
			name:     "real world example with duration",
			input:    "--browser.readiness.timeout=60s",
			expected: []string{"--browser.readiness.timeout=60s"},
		},
		{
			name:     "real world example with multiple flags",
			input:    "--browser.readiness.timeout=60s --browser.min-width=1200",
			expected: []string{"--browser.readiness.timeout=60s", "--browser.min-width=1200"},
		},
		{
			name:     "single dash in value should not split",
			input:    "--flag=some-value --flag2=other",
			expected: []string{"--flag=some-value", "--flag2=other"},
		},
		{
			name:     "flag with no value",
			input:    "--flag --flag2=value",
			expected: []string{"--flag", "--flag2=value"},
		},
		{
			name:     "single dash flag",
			input:    "-f=value",
			expected: []string{"-f=value"},
		},
		{
			name:     "mixed single and double dash flags",
			input:    "-f=value --long=other -g",
			expected: []string{"-f=value", "--long=other", "-g"},
		},
		{
			name:     "single dash flag with dash in value",
			input:    "-f=some-value -g=other",
			expected: []string{"-f=some-value", "-g=other"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseOverrideFlags(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
