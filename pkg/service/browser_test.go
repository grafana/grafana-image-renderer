package service

import (
	"testing"

	"github.com/chromedp/cdproto/network"
	"github.com/grafana/grafana-image-renderer/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithHeaderForDomain(t *testing.T) {
	t.Parallel()

	cfg := config.BrowserConfig{}
	var err error
	cfg, err = WithHeaderForDomain("X-Access-Token", "access-token", "grafana.example.com")(cfg)
	require.NoError(t, err)

	require.Contains(t, cfg.HeadersByDomain, "grafana.example.com")
	assert.Equal(t, network.Headers{
		"X-Access-Token": "access-token",
	}, cfg.HeadersByDomain["grafana.example.com"])
}

func TestWithHeaderForDomainRejectsInvalidOptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		header string
		domain string
	}{
		{name: "empty header", domain: "grafana.example.com"},
		{name: "empty domain", header: "X-Access-Token"},
		{name: "domain normalizes to empty", header: "X-Access-Token", domain: "."},
		{name: "domain with scheme", header: "X-Access-Token", domain: "https://grafana.example.com"},
		{name: "domain with path", header: "X-Access-Token", domain: "grafana.example.com/path"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := WithHeaderForDomain(tt.header, "token", tt.domain)(config.BrowserConfig{})
			require.ErrorIs(t, err, ErrInvalidBrowserOption)
		})
	}
}

func TestHeadersForRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		configuredDomain string
		requestURL       string
		wantForwarded    bool
	}{
		{
			name:             "exact HTTP host",
			configuredDomain: "grafana.example.com",
			requestURL:       "http://grafana.example.com/d/dashboard",
			wantForwarded:    true,
		},
		{
			name:             "port is ignored",
			configuredDomain: "grafana.example.com:3000",
			requestURL:       "https://grafana.example.com:8443/api/ds/query",
			wantForwarded:    true,
		},
		{
			name:             "hostname is case insensitive and ignores trailing dot",
			configuredDomain: "Grafana.Example.Com.",
			requestURL:       "https://GRAFANA.EXAMPLE.COM./public/build/app.js",
			wantForwarded:    true,
		},
		{
			name:             "IPv6 host",
			configuredDomain: "[2001:db8::1]:3000",
			requestURL:       "http://[2001:db8::1]:8080/d/dashboard",
			wantForwarded:    true,
		},
		{
			name:             "subdomain does not match",
			configuredDomain: "grafana.example.com",
			requestURL:       "https://assets.grafana.example.com/app.js",
		},
		{
			name:             "lookalike domain does not match",
			configuredDomain: "grafana.example.com",
			requestURL:       "https://grafana.example.com.attacker.test/",
		},
		{
			name:             "redirect destination is checked independently",
			configuredDomain: "grafana.example.com",
			requestURL:       "https://attacker.test/redirect-target",
		},
		{
			name:             "userinfo cannot disguise destination",
			configuredDomain: "grafana.example.com",
			requestURL:       "https://grafana.example.com@attacker.test/",
		},
		{
			name:             "non-HTTP scheme does not match",
			configuredDomain: "grafana.example.com",
			requestURL:       "file://grafana.example.com/etc/passwd",
		},
		{
			name:             "malformed URL does not match",
			configuredDomain: "grafana.example.com",
			requestURL:       "://grafana.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg, err := WithHeaderForDomain("X-Access-Token", "secret", tt.configuredDomain)(config.BrowserConfig{})
			require.NoError(t, err)

			base := network.Headers{"Accept-Language": "en-US"}
			got := headersForRequest(tt.requestURL, base, cfg.HeadersByDomain)

			assert.Equal(t, "en-US", got["Accept-Language"])
			if tt.wantForwarded {
				assert.Equal(t, "secret", got["X-Access-Token"])
			} else {
				assert.NotContains(t, got, "X-Access-Token")
			}
			assert.NotContains(t, base, "X-Access-Token", "input headers must not be mutated")
		})
	}
}

func TestShouldTrackReadinessNetworkRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		event    *network.EventRequestWillBeSent
		expected bool
	}{
		{
			name:     "nil event",
			event:    nil,
			expected: false,
		},
		{
			name:     "nil request payload",
			event:    &network.EventRequestWillBeSent{},
			expected: false,
		},
		{
			name: "http URL",
			event: &network.EventRequestWillBeSent{
				Request: &network.Request{URL: "http://example.com/path"},
			},
			expected: true,
		},
		{
			name: "https URL uppercase scheme",
			event: &network.EventRequestWillBeSent{
				Request: &network.Request{URL: "HTTPS://example.com/path"},
			},
			expected: true,
		},
		{
			name: "blob URL",
			event: &network.EventRequestWillBeSent{
				Request: &network.Request{URL: "blob:http://example.com/worker"},
			},
			expected: false,
		},
		{
			name: "event source request",
			event: &network.EventRequestWillBeSent{
				Type:    network.ResourceTypeEventSource,
				Request: &network.Request{URL: "https://example.com/events"},
			},
			expected: false,
		},
		{
			name: "websocket request",
			event: &network.EventRequestWillBeSent{
				Type:    network.ResourceTypeWebSocket,
				Request: &network.Request{URL: "wss://example.com/live"},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, shouldTrackReadinessNetworkRequest(tt.event))
		})
	}
}

func TestReadinessNetworkObserverTracksUniqueRequestIDs(t *testing.T) {
	t.Parallel()

	observer := newReadinessNetworkObserver()

	first := &network.EventRequestWillBeSent{
		RequestID: network.RequestID("req-1"),
		Request:   &network.Request{URL: "https://example.com/main.js"},
	}
	assert.True(t, observer.onRequestWillBeSent(first))
	assert.Equal(t, int64(1), observer.inflight())

	// Redirect / duplicate request events for the same RequestID should not increase inflight count.
	assert.False(t, observer.onRequestWillBeSent(first))
	assert.Equal(t, int64(1), observer.inflight())

	observer.onRequestCompleted(network.RequestID("req-1"))
	assert.Equal(t, int64(0), observer.inflight())

	// Duplicate completion events should be ignored and never make inflight negative.
	observer.onRequestCompleted(network.RequestID("req-1"))
	assert.Equal(t, int64(0), observer.inflight())
}

func TestReadinessNetworkObserverIgnoresUntrackedRequests(t *testing.T) {
	t.Parallel()

	observer := newReadinessNetworkObserver()

	assert.False(t, observer.onRequestWillBeSent(&network.EventRequestWillBeSent{
		RequestID: network.RequestID("blob-1"),
		Request:   &network.Request{URL: "blob:https://example.com/worker"},
	}))
	assert.Equal(t, int64(0), observer.inflight())

	observer.onRequestCompleted(network.RequestID("blob-1"))
	assert.Equal(t, int64(0), observer.inflight())
}

func TestWithPageScaleFactorClampsToMax(t *testing.T) {
	t.Parallel()

	t.Run("clamps when above max", func(t *testing.T) {
		t.Parallel()

		cfg := config.BrowserConfig{
			DefaultRequestConfig: config.RequestConfig{
				PageScaleFactor:    1.0,
				MaxPageScaleFactor: 4.0,
			},
		}

		updated, err := WithPageScaleFactor(10)(cfg)
		require.NoError(t, err)
		assert.Equal(t, 4.0, updated.DefaultRequestConfig.PageScaleFactor)
	})

	t.Run("keeps requested value when within max", func(t *testing.T) {
		t.Parallel()

		cfg := config.BrowserConfig{
			DefaultRequestConfig: config.RequestConfig{
				PageScaleFactor:    1.0,
				MaxPageScaleFactor: 4.0,
			},
		}

		updated, err := WithPageScaleFactor(2)(cfg)
		require.NoError(t, err)
		assert.Equal(t, 2.0, updated.DefaultRequestConfig.PageScaleFactor)
	})

	t.Run("clamps to a raised max", func(t *testing.T) {
		t.Parallel()

		cfg := config.BrowserConfig{
			DefaultRequestConfig: config.RequestConfig{
				PageScaleFactor:    1.0,
				MaxPageScaleFactor: 5.0,
			},
		}

		updated, err := WithPageScaleFactor(10)(cfg)
		require.NoError(t, err)
		assert.Equal(t, 5.0, updated.DefaultRequestConfig.PageScaleFactor)
	})

	t.Run("rejects non-positive factor", func(t *testing.T) {
		t.Parallel()

		cfg := config.BrowserConfig{
			DefaultRequestConfig: config.RequestConfig{
				PageScaleFactor:    1.0,
				MaxPageScaleFactor: 4.0,
			},
		}

		_, err := WithPageScaleFactor(0)(cfg)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidBrowserOption)
	})
}
