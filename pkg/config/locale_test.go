package config

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v3"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/text/language"
)

func TestParseLocale(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		in      string
		want    string
		wantErr bool
	}{
		{in: "", want: "und"},
		{in: "  ", want: "und"},
		{in: "fr-FR", want: "fr-FR"},
		{in: "fr-fr", want: "fr-FR"},
		{in: "fr_FR", want: "fr-FR"},
		{in: "de-CH", want: "de-CH"},
		{in: "pt-BR", want: "pt-BR"},
		{in: "fr-FR-u-nu-arab", want: "fr-FR-u-nu-arab"},
		{in: "und", wantErr: true},
		{in: "zz-ZZ", wantErr: true},
		{in: "xx", wantErr: true},
		{in: "fr-CH, fr", wantErr: true},
		{in: "fr-FR;q=0.9", wantErr: true},
		{in: "not a locale", wantErr: true},
		{in: "C", wantErr: true},
		{in: "en_US.UTF-8", wantErr: true},
	} {
		t.Run(tc.in, func(t *testing.T) {
			t.Parallel()

			got, err := ParseLocale(tc.in)
			if tc.wantErr {
				require.Error(t, err)
				assert.Equal(t, language.Und, got)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got.String())
		})
	}
}

func TestBrowserLocaleFlag(t *testing.T) {
	t.Parallel()

	parse := func(t *testing.T, args ...string) (BrowserConfig, error) {
		t.Helper()
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
		err := cmd.Run(t.Context(), append([]string{""}, args...))
		if parseErr != nil {
			return browserConfig, parseErr
		}
		return browserConfig, err
	}

	t.Run("unset means no override", func(t *testing.T) {
		t.Parallel()
		cfg, err := parse(t)
		require.NoError(t, err)
		assert.Equal(t, language.Und, cfg.DefaultRequestConfig.Locale)
	})

	t.Run("value is canonicalized", func(t *testing.T) {
		t.Parallel()
		cfg, err := parse(t, "--browser.locale=fr_fr")
		require.NoError(t, err)
		assert.Equal(t, "fr-FR", cfg.DefaultRequestConfig.Locale.String())
	})

	t.Run("invalid value fails with the value in the error", func(t *testing.T) {
		t.Parallel()
		_, err := parse(t, "--browser.locale=bogus value")
		require.Error(t, err)
		assert.Contains(t, err.Error(), `invalid browser locale "bogus value"`)
	})

	t.Run("override pattern sets a different locale", func(t *testing.T) {
		t.Parallel()
		cfg, err := parse(t,
			"--browser.locale=fr-FR",
			"--browser.override=/d/abc/=--browser.locale=en-US",
		)
		require.NoError(t, err)
		assert.Equal(t, "fr-FR", cfg.DefaultRequestConfig.Locale.String())
		require.Len(t, cfg.RequestConfigOverrides, 1)
		assert.Equal(t, "en-US", cfg.RequestConfigOverrides[0].Config.Locale.String())
		assert.Equal(t, "fr-FR", cfg.LookupRequestConfig(trace.SpanFromContext(context.Background()), "http://grafana/d/xyz/other").Locale.String())
		assert.Equal(t, "en-US", cfg.LookupRequestConfig(trace.SpanFromContext(context.Background()), "http://grafana/d/abc/sales").Locale.String())
	})
}
