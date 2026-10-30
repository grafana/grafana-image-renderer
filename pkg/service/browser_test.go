package service

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/chromedp/cdproto"
	"github.com/chromedp/cdproto/cdp/jsonv2"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"github.com/grafana/grafana-image-renderer/pkg/traces"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/chromedp/cdproto/network"
	"github.com/grafana/grafana-image-renderer/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func TestReadinessNetworkObserverCompletionBeforeRequest(t *testing.T) {
	t.Parallel()

	observer := newReadinessNetworkObserver()
	observer.onRequestCompleted("req-1")

	// Typed event subscriptions are consumed independently; completion can be
	// handled before the corresponding request notification.
	assert.False(t, observer.onRequestWillBeSent(&network.EventRequestWillBeSent{
		RequestID: "req-1",
		Request:   &network.Request{URL: "https://example.com/main.js"},
	}))
	assert.Zero(t, observer.inflight(), "completed requests must not block readiness")
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

// browserTestTransport exercises actions through chromedp's real target/session
// machinery while replacing only Chrome's protocol connection.
type browserTestTransport struct {
	messages chan cdproto.Message
	done     chan struct{}
	once     sync.Once
	respond  func(*cdproto.Message) jsonv2.Value
}

func (b *browserTestTransport) Read(ctx context.Context, msg *cdproto.Message) error {
	select {
	case *msg = <-b.messages:
		return nil
	case <-b.done:
		return io.EOF
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (b *browserTestTransport) Write(ctx context.Context, msg *cdproto.Message) error {
	result := jsonv2.Value(`{}`)
	switch msg.Method {
	case "Target.attachToTarget":
		result = jsonv2.Value(`{"sessionId":"test-session"}`)
	case "Runtime.evaluate":
		// A worker target avoids unrelated frame/DOM initialization.
		result = jsonv2.Value(`{"result":{"type":"object","className":"WorkerGlobalScope"}}`)
	}
	if b.respond != nil {
		if r := b.respond(msg); r != nil {
			result = r
		}
	}
	select {
	case b.messages <- cdproto.Message{ID: msg.ID, SessionID: msg.SessionID, Result: result}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (b *browserTestTransport) Close() error {
	b.once.Do(func() { close(b.done) })
	return nil
}

func browserTestContext(t *testing.T, respond func(*cdproto.Message) jsonv2.Value) (context.Context, *browserTestTransport) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	t.Cleanup(cancel)
	transport := &browserTestTransport{messages: make(chan cdproto.Message, 32), done: make(chan struct{}), respond: respond}
	browser, err := chromedp.NewBrowserTransport(ctx, transport)
	require.NoError(t, err)
	ctx, cancelBrowser := chromedp.NewContext(ctx, chromedp.WithTargetID("test-target"), func(c *chromedp.Context) {
		c.Browser = browser
	})
	t.Cleanup(cancelBrowser)
	require.NoError(t, chromedp.Do(ctx))
	return ctx, transport
}

func TestPrinterProtocolParameters(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		printer Printer
		method  string
		want    map[string]any
	}{
		{
			name:    "PDF preserves zero margins and explicit false options",
			printer: &pdfPrinter{paperSize: PaperA4, printBackground: false, pageRanges: "1-2"},
			method:  page.PrintToPDF.Method,
			want: map[string]any{
				"marginTop": float64(0), "marginBottom": float64(0), "marginLeft": float64(0), "marginRight": float64(0),
				"printBackground": false, "landscape": false, "paperWidth": 8.2677, "paperHeight": 11.70,
				"scale": 0.5, "pageRanges": "1-2",
			},
		},
		{
			name:    "PNG excludes content beyond viewport",
			printer: &pngPrinter{},
			method:  page.CaptureScreenshot.Method,
			want:    map[string]any{"format": "png", "captureBeyondViewport": false},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			params := make(chan jsonv2.Value, 1)
			ctx, _ := browserTestContext(t, func(msg *cdproto.Message) jsonv2.Value {
				if string(msg.Method) == tt.method {
					params <- msg.Params
					return jsonv2.Value(`{"data":"cmVuZGVyZWQ="}`)
				}
				return nil
			})
			output := make(chan []byte, 1)
			cfg := config.BrowserConfig{DefaultRequestConfig: config.RequestConfig{PageScaleFactor: 2}}
			require.NoError(t, chromedp.Do(ctx, tt.printer.action(output, cfg, "https://example.com")))
			assert.Equal(t, []byte("rendered"), <-output)
			var got map[string]any
			require.NoError(t, json.Unmarshal(<-params, &got))
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestWaitForReadyBindingBeforeAction(t *testing.T) {
	t.Parallel()
	ctx, transport := browserTestContext(t, func(msg *cdproto.Message) jsonv2.Value {
		if msg.Method == "Runtime.evaluate" {
			var params runtime.EvaluateParams
			if err := json.Unmarshal(msg.Params, &params); err == nil && params.Expression != "self" {
				return jsonv2.Value(`{"result":{"type":"boolean","value":true}}`)
			}
		}
		return nil
	})
	ctx, _ = ContextWithReadinessModeTracker(ctx)
	action := waitForReady(ctx, config.BrowserConfig{DefaultRequestConfig: config.RequestConfig{
		ReadinessTimeout: time.Second,
	}}, "https://example.com")
	for _, name := range []string{"unrelatedBinding", "__grafanaImageRendererMessageChannel", "__grafanaImageRendererMessageChannel"} {
		payload, err := json.Marshal(runtime.EventBindingCalled{Name: name})
		require.NoError(t, err)
		transport.messages <- cdproto.Message{SessionID: "test-session", Method: cdproto.MethodType(runtime.BindingCalled.Method), Params: payload}
	}
	require.NoError(t, chromedp.Do(ctx, action))
	assert.Equal(t, ReadinessModeBinding, ReadinessModeFromContext(ctx))
}

func TestBrowserNetworkObserverEventOrder(t *testing.T) {
	t.Parallel()
	for _, order := range []string{"rsf", "rfs", "srf", "sfr", "frs", "fsr"} {
		t.Run(order, func(t *testing.T) {
			t.Parallel()
			recorder := tracetest.NewSpanRecorder()
			provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
			t.Cleanup(func() { require.NoError(t, provider.Shutdown(context.Background())) })
			ctx := traces.WithTracerProvider(t.Context(), provider)
			observer := &browserNetworkObserver{requests: make(map[network.RequestID]*browserNetworkRequest)}
			mime := "application/test-" + order
			registry := prometheus.NewRegistry()
			registry.MustRegister(MetricBrowserRequestSize)
			MetricBrowserRequestSize.WithLabelValues(mime)
			readSample := func() (uint64, float64) {
				metrics, err := registry.Gather()
				require.NoError(t, err)
				require.NotEmpty(t, metrics)
				for _, metric := range metrics[0].Metric {
					for _, label := range metric.Label {
						if label.GetName() == "mime_type" && label.GetValue() == mime {
							return metric.Histogram.GetSampleCount(), metric.Histogram.GetSampleSum()
						}
					}
				}
				t.Fatal("request size metric missing for MIME type")
				return 0, 0
			}
			countBefore, sumBefore := readSample()

			for _, event := range order {
				observer.update(ctx, "request", func(r *browserNetworkRequest) {
					switch event {
					case 'r':
						r.request = &network.EventRequestWillBeSent{
							Request:  &network.Request{URL: "https://example.com", Method: "GET"},
							WallTime: 1700000000.25, Timestamp: 100,
						}
					case 's':
						r.response = &network.EventResponseReceived{
							Response: &network.Response{Status: 503, MimeType: mime}, Timestamp: 100.5,
						}
					case 'f':
						r.finished = &network.EventLoadingFinished{EncodedDataLength: 2048}
					}
				})
			}
			spans := recorder.Ended()
			require.Len(t, spans, 1)
			assert.Equal(t, time.Unix(1700000000, 250000000), spans[0].StartTime())
			assert.Equal(t, 500*time.Millisecond, spans[0].EndTime().Sub(spans[0].StartTime()))
			assert.Equal(t, codes.Error, spans[0].Status().Code)
			assert.Equal(t, "503 Service Unavailable", spans[0].Status().Description)
			assert.Empty(t, observer.requests, "all joined events should be released")

			count, sum := readSample()
			assert.Equal(t, countBefore+1, count)
			assert.Equal(t, sumBefore+2048, sum)
		})
	}
}

func TestAllocatorTransport(t *testing.T) {
	t.Parallel()
	if os.PathSeparator != '/' {
		t.Skip("uses a Unix shell fixture")
	}
	for _, tc := range []struct {
		name      string
		flags     []string
		websocket bool
	}{
		{name: "default pipe"},
		{name: "explicit debugging port", flags: []string{"--remote-debugging-port=0"}, websocket: true},
		{name: "explicit debugging address", flags: []string{"--remote-debugging-address=127.0.0.1"}, websocket: true},
	} {
		for _, mode := range []string{"direct", "namespaced"} {
			t.Run(tc.name+"/"+mode, func(t *testing.T) {
				t.Parallel()
				cwd := t.TempDir()
				browserPath := filepath.Join(cwd, "browser")
				argsPath := filepath.Join(cwd, "args")
				const script = `#!/bin/sh
printf '%s\n' "$*" > "$GIR_ALLOCATOR_ARGS"
`
				require.NoError(t, os.WriteFile(browserPath, []byte(script), 0o700))
				cfg := config.BrowserConfig{
					Path: "/opt/chromium", Namespaced: mode == "namespaced", TimeZone: time.UTC, Flags: tc.flags,
					WSURLReadTimeout:     time.Second,
					DefaultRequestConfig: config.RequestConfig{MinWidth: 800, MinHeight: 600},
				}
				options, err := (&BrowserService{}).createAllocatorOptions(t.Context(), cfg, "https://example.com", cwd)
				require.NoError(t, err)
				// Record arguments before exiting; startup output can race process exit.
				// Use a private profile directory in the namespaced case too.
				options = append(options, chromedp.ExecPath(browserPath), chromedp.UserDataDir(cwd), chromedp.Env("GIR_ALLOCATOR_ARGS="+argsPath))
				ctx, cancelAllocator := chromedp.NewExecAllocator(t.Context(), options...)
				t.Cleanup(cancelAllocator)
				ctx, cancelBrowser := chromedp.NewContext(ctx)
				t.Cleanup(cancelBrowser)
				require.Error(t, chromedp.Do(ctx))
				args, err := os.ReadFile(argsPath)
				require.NoError(t, err)
				got := string(args)
				if tc.websocket {
					assert.Contains(t, got, "--remote-debugging-port=0")
					assert.NotContains(t, got, "--remote-debugging-pipe")
				} else {
					assert.Contains(t, got, "--remote-debugging-pipe")
					assert.NotContains(t, got, "--remote-debugging-port")
				}
				assert.Contains(t, got, "--window-size=800,600")
				if mode == "namespaced" {
					assert.Contains(t, got, "_internal_sandbox bootstrap --tmp "+cwd+" --cwd /tmp --trace  -- /opt/chromium --")
				} else {
					assert.NotContains(t, got, "_internal_sandbox")
				}
			})
		}
	}
}

func TestBrowserHeadersAndCookiesProtocol(t *testing.T) {
	t.Parallel()
	commands := make(chan cdproto.Message, 2)
	ctx, _ := browserTestContext(t, func(msg *cdproto.Message) jsonv2.Value {
		if msg.Method == "Network.setExtraHTTPHeaders" || msg.Method == "Network.setCookie" {
			commands <- *msg
		}
		return nil
	})
	require.NoError(t, chromedp.Do(ctx,
		setHeaders(network.Headers{"Accept-Language": "pt-BR"}),
		setCookies([]*network.SetCookieParams{{Name: "renderKey", Value: "test-token", Domain: "example.com", Secure: new(false), HTTPOnly: new(true)}}),
	))
	for _, want := range []string{
		`{"headers":{"Accept-Language":"pt-BR"}}`,
		`{"name":"renderKey","value":"test-token","domain":"example.com","secure":false,"httpOnly":true}`,
	} {
		assert.JSONEq(t, want, string((<-commands).Params))
	}
	// Empty headers/cookies must remain valid no-op actions.
	require.NoError(t, chromedp.Do(ctx, setHeaders(nil), setCookies(nil)))
	assert.Empty(t, commands)
}
