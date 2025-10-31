package config

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"strings"
	"time"
	_ "time/tzdata" // include fallback tzdata if none exist on the file system

	"github.com/chromedp/cdproto/network"
	"github.com/grafana/grafana-image-renderer/pkg/sandbox"
	"github.com/urfave/cli/v3"
)

type LoggingConfig struct {
	// Level is the minimum level to log.
	Level LogLevel
}

func LoggingFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:    "log.level",
			Usage:   fmt.Sprintf("The minimum level to log at (enum: %s, %s, %s, %s)", LogLevelDebug, LogLevelInfo, LogLevelWarn, LogLevelError),
			Value:   LogLevelInfo.String(),
			Sources: FromConfig("log.level", "LOG_LEVEL"),
			Validator: func(s string) error {
				if _, err := LogLevel(s).ToSlog(); err != nil {
					return err
				}
				return nil
			},
		},
	}
}

func LoggingConfigFromCommand(c *cli.Command) (LoggingConfig, error) {
	return LoggingConfig{
		Level: LogLevel(c.String("log.level")),
	}, nil
}

type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
)

func (l LogLevel) String() string {
	return string(l)
}

func (l LogLevel) ToSlog() (slog.Leveler, error) {
	switch LogLevel(strings.ToLower(string(l))) {
	case LogLevelDebug:
		return slog.LevelDebug, nil
	case LogLevelInfo:
		return slog.LevelInfo, nil
	case LogLevelWarn:
		return slog.LevelWarn, nil
	case LogLevelError:
		return slog.LevelError, nil
	default:
		return nil, fmt.Errorf("unknown log level: %s", l)
	}
}

type ServerConfig struct {
	// Addr is the HTTP address to listen on.
	// This must be a TCP address, e.g. ":8080" or "[::1]:1234".
	Addr string
	// AuthTokens are the tokens that must be presented in the X-Auth-Token header to authorize requests.
	AuthTokens      []string
	CertificateFile string
	KeyFile         string
	MinTLSVersion   TLSVersion
}

func ServerFlags() []cli.Flag {
	return slices.Concat(
		[]cli.Flag{
			&cli.StringFlag{
				Name:    "server.addr",
				Usage:   "The address to listen on for HTTP requests.",
				Value:   ":8081",
				Sources: FromConfig("server.addr", "SERVER_ADDR"),
			},
			&cli.StringSliceFlag{
				Name:    "server.auth-token",
				Aliases: []string{"server.auth-tokens", "server.token", "server.tokens"},
				Usage:   "The X-Auth-Token header value that must be sent to the service to permit requests. May be repeated.",
				Value:   []string{"-"},
				Sources: FromConfig("server.auth-token", "AUTH_TOKEN"),
			},
			&cli.StringFlag{
				Name:      "server.certificate-file",
				Aliases:   []string{"server.certificate", "server.cert-file", "server.cert"},
				Usage:     "A path to a TLS certificate file to use for HTTPS. If not set, HTTP is used.",
				TakesFile: true,
				Sources:   FromConfig("server.certificate-file", "SERVER_CERTIFICATE_FILE"),
			},
			&cli.StringFlag{
				Name:      "server.key-file",
				Aliases:   []string{"server.key"},
				Usage:     "A path to a TLS key file to use for HTTPS.",
				TakesFile: true,
				Sources:   FromConfig("server.key-file", "SERVER_KEY_FILE"),
			},
			&cli.StringFlag{
				Name:    "server.min-tls-version",
				Usage:   "The minimum TLS version to accept for HTTPS connections. (enum: 1.0, 1.1, 1.2, 1.3)",
				Value:   "1.2",
				Sources: FromConfig("server.min-tls-version", "SERVER_MIN_TLS_VERSION"),
			},
		},
	)
}

func ServerConfigFromCommand(c *cli.Command) (ServerConfig, error) {
	minTLSVersion := TLSVersion(c.String("server.min-tls-version"))
	if _, err := minTLSVersion.ToTLSConstant(); err != nil {
		return ServerConfig{}, fmt.Errorf("invalid server min-tls-version: %w", err)
	}

	return ServerConfig{
		Addr:            c.String("server.addr"),
		AuthTokens:      c.StringSlice("server.auth-token"),
		CertificateFile: c.String("server.certificate-file"),
		KeyFile:         c.String("server.key-file"),
		MinTLSVersion:   minTLSVersion,
	}, nil
}

type TLSVersion string

const (
	TLSVersion1_0 TLSVersion = "1.0"
	TLSVersion1_1 TLSVersion = "1.1"
	TLSVersion1_2 TLSVersion = "1.2"
	TLSVersion1_3 TLSVersion = "1.3"
)

func (v TLSVersion) String() string {
	return string(v)
}

func (v TLSVersion) ToTLSConstant() (uint16, error) {
	switch v {
	case TLSVersion1_0:
		return tls.VersionTLS10, nil
	case TLSVersion1_1:
		return tls.VersionTLS11, nil
	case TLSVersion1_2:
		return tls.VersionTLS12, nil
	case TLSVersion1_3:
		return tls.VersionTLS13, nil
	default:
		return 0, fmt.Errorf("unknown TLS version: %s", v)
	}
}

type TracingConfig struct {
	Endpoint           string
	Insecure           *bool
	Headers            map[string]string
	Compressor         string
	Timeout            time.Duration
	TrustedCertificate string
	ClientCertificate  string
	ClientKey          string
	ServiceName        string
}

func TracingFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:    "tracing.endpoint",
			Usage:   "The tracing endpoint to send spans to. Use grpc://, http://, or https:// to specify the protocol (grpc:// is implied).",
			Sources: FromConfig("tracing.endpoint", "TRACING_ENDPOINT"),
			Validator: func(s string) error {
				before, _, found := strings.Cut(s, "://")
				if !found {
					return nil // grpc://
				}
				switch before {
				case "grpc", "http", "https":
					return nil
				default:
					return fmt.Errorf("unknown protocol scheme in tracing endpoint: %s", before)
				}
			},
		},
		&cli.BoolFlag{
			Name:    "tracing.insecure",
			Usage:   "Whether to skip TLS verification when connecting. If set, the scheme in the endpoint is overridden to be insecure.",
			Sources: FromConfig("tracing.insecure", "TRACING_INSECURE"),
		},
		&cli.StringSliceFlag{
			Name:    "tracing.header",
			Aliases: []string{"tracing.headers"},
			Usage:   "A header to add to requests to the tracing endpoint. Syntax is `${key}=${value}`. May be repeated. This is useful for things like authentication.",
			Sources: FromConfig("tracing.header", "TRACING_HEADER"),
			Validator: func(s []string) error {
				for _, kv := range s {
					if !strings.Contains(kv, "=") {
						return fmt.Errorf("invalid tracing header, missing =: %s", kv)
					}
				}
				return nil
			},
		},
		&cli.StringFlag{
			Name:    "tracing.compressor",
			Usage:   "The compression algorithm to use when sending traces. (enum: none, gzip)",
			Value:   "none",
			Sources: FromConfig("tracing.compressor", "TRACING_COMPRESSOR"),
			Validator: func(s string) error {
				if s == "" || s == "none" || s == "gzip" {
					return nil
				}
				return fmt.Errorf("invalid tracing compressor: %s", s)
			},
		},
		&cli.DurationFlag{
			Name:    "tracing.timeout",
			Usage:   "The timeout for requests to the tracing endpoint.",
			Value:   10 * time.Second,
			Sources: FromConfig("tracing.timeout", "TRACING_TIMEOUT"),
		},
		&cli.StringFlag{
			Name:      "tracing.trusted-certificate",
			Usage:     "A path to a PEM-encoded certificate to use as a trusted root when connecting to the tracing endpoint over gRPC or HTTPS.",
			TakesFile: true,
			Sources:   FromConfig("tracing.trusted_certificate", "TRACING_TRUSTED_CERTIFICATE"),
		},
		&cli.StringFlag{
			Name:      "tracing.client-certificate",
			Usage:     "A path to a PEM-encoded client certificate to use for mTLS when connecting to the tracing endpoint over gRPC or HTTPS.",
			TakesFile: true,
			Sources:   FromConfig("tracing.client_certificate", "TRACING_CLIENT_CERTIFICATE"),
		},
		&cli.StringFlag{
			Name:      "tracing.client-key",
			Usage:     "A path to a PEM-encoded client key to use for mTLS when connecting to the tracing endpoint over gRPC or HTTPS.",
			TakesFile: true,
			Sources:   FromConfig("tracing.client_key", "TRACING_CLIENT_KEY"),
		},
		&cli.StringFlag{
			Name:    "tracing.service-name",
			Usage:   "The service name to use in traces.",
			Value:   "grafana-image-renderer",
			Sources: FromConfig("tracing.service_name", "TRACING_SERVICE_NAME"),
		},
	}
}

func TracingConfigFromCommand(c *cli.Command) (TracingConfig, error) {
	headers := make(map[string]string)
	for _, kv := range c.StringSlice("tracing.header") {
		k, v, found := strings.Cut(kv, "=")
		if !found {
			return TracingConfig{}, fmt.Errorf("invalid tracing header, missing =: %s", kv)
		}
		headers[k] = v
	}

	var insecure *bool
	if c.IsSet("tracing.insecure") {
		v := c.Bool("tracing.insecure")
		insecure = &v
	}

	return TracingConfig{
		Endpoint:           c.String("tracing.endpoint"),
		Insecure:           insecure,
		Headers:            headers,
		Compressor:         c.String("tracing.compressor"),
		Timeout:            c.Duration("tracing.timeout"),
		TrustedCertificate: c.String("tracing.trusted-certificate"),
		ClientCertificate:  c.String("tracing.client-certificate"),
		ClientKey:          c.String("tracing.client-key"),
		ServiceName:        c.String("tracing.service-name"),
	}, nil
}

type BrowserConfig struct {
	// Path is the path to the browser binary.
	// This is resolved against PATH.
	Path string
	// Flags are the parameters to pass to the browser.
	// A leading `--` is implied, but still valid to pass in.
	Flags []string
	// GPU indicates whether to enable GPU support in the browser.
	GPU bool
	// Sandbox indicates whether to enable the browser's sandbox.
	// This may require extra configuration on the service to work properly in Kubernetes and similar environments, but in exchange, it is a very good security practice.
	Sandbox bool
	// Namespaced indicates whether to run the browser in a new namespace jail.
	// This is implemented by the service itself. It requires Linux and some capabilities (CAP_SYS_ADMIN, CAP_SYS_CHROOT) or a privileged user.
	// Most users don't need this, but it may be interesting for users who care more about security than performance.
	Namespaced bool
	// TimeZone is the timezone for the browser to use.
	TimeZone *time.Location // DeepClone: can be copied, because the value should be immutable
	// Cookies are injected into the browser for every request.
	// The browser will only send cookies to the domains they are valid for, in the situations they are valid to share.
	Cookies []*network.SetCookieParams // DeepClone: values can't just be copied
	// Headers are set on every request the browser makes, not only to a specific domain.
	// This is useful to pass around trace IDs and similar, but should be avoided for sensitive data (e.g. authentication).
	Headers network.Headers // DeepClone: can't just be copied (is a map)
	// TimeBetweenScrolls changes how long we wait for a scroll event to complete before starting a new one.
	//
	// We will scroll the entire web-page by the entire viewport over and over until we have seen everything.
	// That means for a viewport that is 500px high, and a webpage that is 2500px high, we will scroll 5 times, meaning a total wait duration of 6 * duration (as we have to wait on the first & last scrolls as well).
	TimeBetweenScrolls time.Duration
	// ReadinessTimeout is the maximum time to wait for the web-page to become ready (i.e. no longer loading anything).
	ReadinessTimeout           time.Duration
	ReadinessIterationInterval time.Duration
	// ReadinessPriorWait is the time to wait before checking for how ready the page is.
	// This lets you force the webpage to take a beat and just do its thing before the service starts looking for whether it's time to render anything.
	ReadinessPriorWait              time.Duration
	ReadinessDisableQueryWait       bool
	ReadinessFirstQueryTimeout      time.Duration
	ReadinessQueriesTimeout         time.Duration
	ReadinessDisableNetworkWait     bool
	ReadinessNetworkIdleTimeout     time.Duration
	ReadinessDisableDOMHashCodeWait bool
	ReadinessDOMHashCodeTimeout     time.Duration

	// MinWidth is the minimum width of the browser viewport.
	// If larger than MaxWidth, MaxWidth is used instead.
	MinWidth int
	// MinHeight is the minimum height of the browser viewport.
	// If larger than MaxHeight, MaxHeight is used instead.
	MinHeight int
	// MaxWidth is the maximum width of the browser viewport.
	// A request cannot request a larger browser viewport than this.
	// If negative, it is ignored.
	MaxWidth int
	// MaxHeight is the maximum height of the browser viewport.
	// A request cannot request a larger browser viewport than this, except for when capturing full-page screenshots.
	// If negative, it is ignored.
	MaxHeight       int
	PageScaleFactor float64
	Landscape       bool
}

func (c BrowserConfig) DeepClone() BrowserConfig {
	cpy := c
	for i, cookie := range c.Cookies {
		cloned := *cookie
		cpy.Cookies[i] = &cloned
	}
	cpy.Headers = network.Headers(maps.Clone(c.Headers))
	return cpy
}

func BrowserFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:      "browser.path",
			Usage:     "The path to the browser's binary. This is resolved against PATH.",
			TakesFile: true,
			Value:     "chromium",
			Sources:   FromConfig("browser.path", "BROWSER_PATH"),
		},
		&cli.StringSliceFlag{
			Name:    "browser.flag",
			Aliases: []string{"browser.flags"},
			Usage:   "Flags to pass to the browser. These are syntaxed `${flag}` or `${flag}=${value}`.",
			Sources: FromConfig("browser.flag", "BROWSER_FLAG"),
		},
		&cli.BoolFlag{
			Name:    "browser.gpu",
			Usage:   "Enable GPU support in the browser.",
			Sources: FromConfig("browser.gpu", "BROWSER_GPU"),
		},
		&cli.BoolFlag{
			Name:    "browser.sandbox",
			Usage:   "Enable the browser's sandbox. Sets the `no-sandbox` flag to `false` for you.",
			Sources: FromConfig("browser.sandbox", "BROWSER_SANDBOX"),
		},
		&cli.BoolFlag{
			Name:    "browser.namespaced",
			Usage:   "Enable namespacing the browser. This requires Linux and the CAP_SYS_ADMIN and CAP_SYS_CHROOT capabilities, or a privileged user.",
			Sources: FromConfig("browser.namespaced", "BROWSER_NAMESPACED"),
			Validator: func(b bool) error {
				if b {
					ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
					defer cancel()
					if !sandbox.Supported(ctx) {
						// We could just not do the sandbox, but we don't want to employ a less secure environment than requested.
						// If this is important to the user, they will have to disable this themselves rather than silently be unaware.
						return fmt.Errorf("browser namespacing is not supported on this system")
					}
				}
				return nil
			},
		},
		&cli.StringFlag{
			Name:    "browser.timezone",
			Aliases: []string{"browser.tz", "browser.time-zone"},
			Usage:   "The timezone for the browser to use, e.g. 'America/New_York'.",
			Value:   "Etc/UTC",
			Sources: FromConfig("browser.timezone", "BROWSER_TIMEZONE", "TZ" /* standard practice in containers */),
		},
		&cli.StringSliceFlag{
			Name:    "browser.header",
			Aliases: []string{"browser.headers"},
			Usage:   "Headers to add to every request the browser makes. Syntax is `${key}=${value}`. May be repeated.",
			Sources: FromConfig("browser.header", "BROWSER_HEADER"),
			Validator: func(s []string) error {
				for _, kv := range s {
					if !strings.Contains(kv, "=") {
						return fmt.Errorf("invalid browser header, missing =: %s", kv)
					}
				}
				return nil
			},
		},
		&cli.DurationFlag{
			Name:    "browser.time-between-scrolls",
			Usage:   "The time between scroll events when capturing a full-page screenshot.",
			Value:   time.Millisecond * 50,
			Sources: FromConfig("browser.time-between-scrolls", "BROWSER_TIME_BETWEEN_SCROLLS"),
		},
		&cli.DurationFlag{
			Name:    "browser.readiness.timeout",
			Usage:   "The maximum time to wait for a web-page to become ready (i.e. no longer loading anything). If <= 0, the timeout is disabled.",
			Value:   time.Second * 30,
			Sources: FromConfig("browser.readiness.timeout", "BROWSER_READINESS_TIMEOUT"),
		},
		&cli.DurationFlag{
			Name:  "browser.readiness.iteration-interval",
			Usage: "How long to wait between each iteration of checking whether the page is ready. Must be positive.",
			Value: time.Millisecond * 100,
			Validator: func(d time.Duration) error {
				if d <= 0 {
					return fmt.Errorf("browser readiness iteration-interval must be positive (got %v)", d)
				}
				return nil
			},
		},
		&cli.DurationFlag{
			Name:    "browser.readiness.prior-wait",
			Usage:   "The time to wait before checking for how ready the page is. This lets you force the webpage to take a beat and just do its thing before the service starts looking for whether it's time to render anything. If <= 0, this is disabled.",
			Value:   time.Second,
			Sources: FromConfig("browser.readiness.prior-wait", "BROWSER_READINESS_PRIOR_WAIT"),
		},
		&cli.BoolFlag{
			Name:    "browser.readiness.disable-query-wait",
			Usage:   "Disable waiting for queries to finish before capturing.",
			Sources: FromConfig("browser.readiness.disable-query-wait", "BROWSER_READINESS_DISABLE_QUERY_WAIT"),
		},
		&cli.DurationFlag{
			Name:    "browser.readiness.give-up-on-first-query",
			Usage:   "How long to wait before giving up on a first query being registered. If <= 0, the give-up is disabled.",
			Value:   time.Second * 3,
			Sources: FromConfig("browser.readiness.give-up-on-first-query", "BROWSER_READINESS_GIVE_UP_ON_FIRST_QUERY"),
		},
		&cli.DurationFlag{
			Name:    "browser.readiness.give-up-on-all-queries",
			Usage:   "How long to wait before giving up on all running queries. If <= 0, the give-up is disabled.",
			Value:   0,
			Sources: FromConfig("browser.readiness.give-up-on-all-queries", "BROWSER_READINESS_GIVE_UP_ON_ALL_QUERIES"),
		},
		&cli.BoolFlag{
			Name:    "browser.readiness.disable-network-wait",
			Usage:   "Disable waiting for network requests to finish before capturing.",
			Sources: FromConfig("browser.readiness.disable-network-wait", "BROWSER_READINESS_DISABLE_NETWORK_WAIT"),
		},
		&cli.DurationFlag{
			Name:    "browser.readiness.network-idle-timeout",
			Usage:   "How long to wait before giving up on the network being idle. If <= 0, the timeout is disabled.",
			Value:   0,
			Sources: FromConfig("browser.readiness.network-idle-timeout", "BROWSER_READINESS_NETWORK_IDLE_TIMEOUT"),
		},
		&cli.BoolFlag{
			Name:    "browser.readiness.disable-dom-hashcode-wait",
			Usage:   "Disable waiting for the DOM to stabilize (i.e. not change) before capturing.",
			Sources: FromConfig("browser.readiness.disable-dom-hashcode-wait", "BROWSER_READINESS_DISABLE_DOM_HASHCODE_WAIT"),
		},
		&cli.DurationFlag{
			Name:    "browser.readiness.dom-hashcode-timeout",
			Usage:   "How long to wait before giving up on the DOM stabilizing (i.e. not changing). If <= 0, the timeout is disabled.",
			Value:   0,
			Sources: FromConfig("browser.readiness.dom-hashcode-timeout", "BROWSER_READINESS_DOM_HASHCODE_TIMEOUT"),
		},
		&cli.IntFlag{
			Name:    "browser.min-width",
			Usage:   "The minimum width of the browser viewport. This is the default width in requests.",
			Value:   1000,
			Sources: FromConfig("browser.min-width", "BROWSER_MIN_WIDTH"),
			Validator: func(i int) error {
				if i < 100 {
					return fmt.Errorf("browser min-width must be at least 100")
				}
				return nil
			},
		},
		&cli.IntFlag{
			Name:    "browser.min-height",
			Usage:   "The minimum height of the browser viewport. This is the default height in requests.",
			Value:   500,
			Sources: FromConfig("browser.min-height", "BROWSER_MIN_HEIGHT"),
			Validator: func(i int) error {
				if i < 100 {
					return fmt.Errorf("browser min-height must be at least 100")
				}
				return nil
			},
		},
		&cli.IntFlag{
			Name:    "browser.max-width",
			Usage:   "The maximum width of the browser viewport. Requests cannot request a larger width than this. Negative means ignored.",
			Value:   3000,
			Sources: FromConfig("browser.max-width", "BROWSER_MAX_WIDTH"),
			Validator: func(i int) error {
				if i >= 0 && i < 100 {
					return fmt.Errorf("browser max-width must be at least 100, or negative to be ignored")
				}
				return nil
			},
		},
		&cli.IntFlag{
			Name:    "browser.max-height",
			Usage:   "The maximum height of the browser viewport. Requests cannot request a larger height than this, except for when capturing full-page screenshots. Negative means ignored.",
			Value:   3000,
			Sources: FromConfig("browser.max-height", "BROWSER_MAX_HEIGHT"),
			Validator: func(i int) error {
				if i >= 0 && i < 100 {
					return fmt.Errorf("browser max-height must be at least 100, or negative to be ignored")
				}
				return nil
			},
		},
		&cli.Float64Flag{
			Name:    "browser.page-scale-factor",
			Usage:   "The page scale factor of the browser.",
			Value:   1.0,
			Sources: FromConfig("browser.page-scale-factor", "BROWSER_PAGE_SCALE_FACTOR"),
		},
		&cli.BoolFlag{
			Name:    "browser.portrait",
			Usage:   "Use a portrait viewport instead of the default landscape.",
			Sources: FromConfig("browser.portrait", "BROWSER_PORTRAIT"),
		},
	}
}

func BrowserConfigFromCommand(c *cli.Command) (BrowserConfig, error) {
	timeZone := time.UTC
	if tz := c.String("browser.timezone"); tz != "" {
		var err error
		timeZone, err = time.LoadLocation(tz)
		if err != nil {
			return BrowserConfig{}, fmt.Errorf("invalid browser timezone %q: %w", tz, err)
		}
	}

	var headers network.Headers
	if hdrs := c.StringSlice("browser.header"); len(hdrs) > 0 {
		headers = make(network.Headers, len(hdrs))
		for _, kv := range hdrs {
			h, v, _ := strings.Cut(kv, "=")
			headers[h] = v
		}
	}

	minWidth := c.Int("browser.min-width")
	minHeight := c.Int("browser.min-height")
	maxWidth := c.Int("browser.max-width")
	maxHeight := c.Int("browser.max-height")
	if maxWidth >= 0 && minWidth > maxWidth {
		return BrowserConfig{}, fmt.Errorf("browser min-width (%d) cannot be larger than max-width (%d)", minWidth, maxWidth)
	}
	if maxHeight >= 0 && minHeight > maxHeight {
		return BrowserConfig{}, fmt.Errorf("browser min-height (%d) cannot be larger than max-height (%d)", minHeight, maxHeight)
	}

	return BrowserConfig{
		Path:                            c.String("browser.path"),
		Flags:                           c.StringSlice("browser.flag"),
		GPU:                             c.Bool("browser.gpu"),
		Sandbox:                         c.Bool("browser.sandbox"),
		Namespaced:                      c.Bool("browser.namespaced"),
		TimeZone:                        timeZone,
		Cookies:                         nil,
		Headers:                         headers,
		TimeBetweenScrolls:              c.Duration("browser.time-between-scrolls"),
		ReadinessTimeout:                c.Duration("browser.readiness.timeout"),
		ReadinessIterationInterval:      c.Duration("browser.readiness.iteration-interval"),
		ReadinessPriorWait:              c.Duration("browser.readiness.prior-wait"),
		ReadinessDisableQueryWait:       c.Bool("browser.readiness.disable-query-wait"),
		ReadinessFirstQueryTimeout:      c.Duration("browser.readiness.give-up-on-first-query"),
		ReadinessQueriesTimeout:         c.Duration("browser.readiness.give-up-on-all-queries"),
		ReadinessDisableNetworkWait:     c.Bool("browser.readiness.disable-network-wait"),
		ReadinessNetworkIdleTimeout:     c.Duration("browser.readiness.network-idle-timeout"),
		ReadinessDisableDOMHashCodeWait: c.Bool("browser.readiness.disable-dom-hashcode-wait"),
		ReadinessDOMHashCodeTimeout:     c.Duration("browser.readiness.dom-hashcode-timeout"),
		MinWidth:                        minWidth,
		MinHeight:                       minHeight,
		MaxWidth:                        maxWidth,
		MaxHeight:                       maxHeight,
		PageScaleFactor:                 c.Float64("browser.page-scale-factor"),
		Landscape:                       !c.Bool("browser.portrait"),
	}, nil
}

type RateLimitConfig struct {
	// Disabled indicates whether rate limiting is disabled.
	Disabled bool

	// TrackerDecay is the number N in decaying averages, `avg = ((N-1)*avg + new) / N`.
	// This must be a minimum of 1, which will not use a slow-moving average at all.
	TrackerDecay int64
	// TrackerInterval is how often to sample process statistics on the browser processes.
	// This must be a minimum of 1ms.
	TrackerInterval time.Duration

	// MinLimit is the minimum number of requests to permit.
	// Even if we don't have slots for it, we will permit at least this many requests.
	// Set to 0 to disable minimum; this is generally not recommended, especially in containerised environments like Kubernetes.
	MinLimit uint32
	// MaxLimit is the maximum number of requests to permit.
	// Even if we have memory slots for more, we won't exceed this.
	// Set to 0 to disable maximum; this is generally the way to go in horizontally scaled deployments.
	MaxLimit uint32

	// MaxAvailable is the maximum amount of memory (in bytes) available to processes.
	// If there is more memory than this, we will only consider this amount.
	// Set to 0 to use all available memory.
	MaxAvailable uint64
	// MinMemoryPerBrowser is the minimum amount of memory (in bytes) each browser process is expected to use.
	// If the process tracker reports less, this is the value used. Otherwise, we use the process tracker's value.
	// Set to 0 to disable the minimum.
	MinMemoryPerBrowser uint64
	// Headroom is how much memory (in bytes) should be left after the request's browser takes its share.
	// If this cannot be accommodated, we will reject the request.
	// Set to 0 to disable headroom.
	Headroom uint64
}

func RateLimitFlags() []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{
			Name:    "rate-limit.disabled",
			Usage:   "Disable rate limiting entirely.",
			Sources: FromConfig("rate-limit.disabled", "RATE_LIMIT_DISABLED"),
		},
		&cli.Int64Flag{
			Name:    "rate-limit.process-tracker.decay",
			Usage:   "The decay factor N to use in slow-moving averages of process statistics, where `avg = ((N-1)*avg + new) / N`. Must be at least 1.",
			Value:   5,
			Sources: FromConfig("rate-limit.process-tracker.decay", "RATE_LIMIT_PROCESS_TRACKER_DECAY"),
			Validator: func(i int64) error {
				if i < 1 {
					return fmt.Errorf("rate-limit.process-tracker.decay must be at least 1")
				}
				return nil
			},
		},
		&cli.DurationFlag{
			Name:    "rate-limit.process-tracker.interval",
			Usage:   "How often to sample process statistics on the browser processes. Must be >= 1ms.",
			Value:   50 * time.Millisecond,
			Sources: FromConfig("rate-limit.process-tracker.interval", "RATE_LIMIT_PROCESS_TRACKER_INTERVAL"),
			Validator: func(d time.Duration) error {
				if d < time.Millisecond {
					return fmt.Errorf("rate-limit.process-tracker.interval must be at least 1ms")
				}
				return nil
			},
		},
		&cli.Uint32Flag{
			Name:    "rate-limit.min-limit",
			Usage:   "The minimum number of requests to permit. Ratelimiting will not reject requests if the number of currently running requests is below this value. Set to 0 to disable minimum (not recommended).",
			Value:   3,
			Sources: FromConfig("rate-limit.min-limit", "RATE_LIMIT_MIN_LIMIT"),
		},
		&cli.Uint32Flag{
			Name:    "rate-limit.max-limit",
			Usage:   "The maximum number of requests to permit. Ratelimiting will reject requests if the number of currently running requests is at or above this value. Set to 0 to disable maximum. The v4 service used 5 by default.",
			Value:   0,
			Sources: FromConfig("rate-limit.max-limit", "RATE_LIMIT_MAX_LIMIT"),
		},
		&cli.Uint64Flag{
			Name:    "rate-limit.max-available",
			Usage:   "The maximum amount of memory (in bytes) available to processes. If more memory exists, only this amount is used. 0 disables the maximum.",
			Value:   0,
			Sources: FromConfig("rate-limit.max-available", "RATE_LIMIT_MAX_AVAILABLE"),
		},
		&cli.Uint64Flag{
			Name:    "rate-limit.min-memory-per-browser",
			Usage:   "The minimum amount of memory (in bytes) each browser process is expected to use. Set to 0 to disable the minimum.",
			Value:   64 * 1024 * 1024, // 64 MiB
			Sources: FromConfig("rate-limit.min-memory-per-browser", "RATE_LIMIT_MIN_MEMORY_PER_BROWSER"),
		},
		&cli.Uint64Flag{
			Name:    "rate-limit.headroom",
			Usage:   "The amount of memory (in bytes) to leave as headroom after allocating memory for browser processes. Set to 0 to disable headroom.",
			Value:   32 * 1024 * 1024, // 32 MiB
			Sources: FromConfig("rate-limit.headroom", "RATE_LIMIT_HEADROOM"),
		},
	}
}

func RateLimitConfigFromCommand(c *cli.Command) (RateLimitConfig, error) {
	return RateLimitConfig{
		Disabled:            c.Bool("rate-limit.disabled"),
		TrackerDecay:        c.Int64("rate-limit.process-tracker.decay"),
		TrackerInterval:     c.Duration("rate-limit.process-tracker.interval"),
		MinLimit:            c.Uint32("rate-limit.min-limit"),
		MaxLimit:            c.Uint32("rate-limit.max-limit"),
		MaxAvailable:        c.Uint64("rate-limit.max-available"),
		MinMemoryPerBrowser: c.Uint64("rate-limit.min-memory-per-browser"),
		Headroom:            c.Uint64("rate-limit.headroom"),
	}, nil
}
