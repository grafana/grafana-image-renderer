package config

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v3"
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
		{in: "hu_HU", want: "hu-HU"},
		{in: "es-419", want: "es-419"},
		{in: "zh-Hant-TW", want: "zh-Hant-TW"},
		{in: "en-US-x-foo", want: "en-US-x-foo"},
		{in: "und", wantErr: true},
		{in: "root", wantErr: true},
		{in: "und-FR", wantErr: true},
		{in: "und-u-nu-arab", wantErr: true},
		{in: "x-foo", wantErr: true},
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
		// The pattern documented in troubleshooting.md: it matches both dashboard and single-panel URLs.
		cfg, err := parse(t,
			"--browser.locale=fr-FR",
			`--browser.override=/d(-solo)?/abc123\b=--browser.locale=en-US`,
		)
		require.NoError(t, err)
		assert.Equal(t, "fr-FR", cfg.DefaultRequestConfig.Locale.String())
		require.Len(t, cfg.RequestConfigOverrides, 1)
		assert.Equal(t, "en-US", cfg.RequestConfigOverrides[0].Config.Locale.String())
		for url, want := range map[string]string{
			"http://grafana/d/abc123/sales":                "en-US",
			"http://grafana/d/abc123?render=1":             "en-US",
			"http://grafana/d-solo/abc123/sales?panelId=2": "en-US",
			"http://grafana/d/abc1234/other":               "fr-FR",
			"http://grafana/d/xyz/other":                   "fr-FR",
		} {
			assert.Equal(t, want, cfg.LookupRequestConfig(noopSpan(), url).Locale.String(), url)
		}
	})
}
