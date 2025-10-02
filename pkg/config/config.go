package config

import (
	"crypto/tls"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"strings"
	"time"
	_ "time/tzdata" // include fallback tzdata if none exist on the file system

	"github.com/chromedp/cdproto/network"
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
	ReadinessTimeout time.Duration
	// LoadWait is the time to wait before checking for how ready the page is.
	// This lets you force the webpage to take a beat and just do its thing before the service starts looking for whether it's time to render anything.
	LoadWait time.Duration

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
			Name:    "browser.readiness-timeout",
			Usage:   "The maximum time to wait for a web-page to become ready (i.e. no longer loading anything).",
			Value:   time.Second * 30,
			Sources: FromConfig("browser.readiness-timeout", "BROWSER_READINESS_TIMEOUT"),
		},
		&cli.DurationFlag{
			Name:    "browser.load-wait",
			Usage:   "The time to wait before checking for how ready the page is. This lets you force the webpage to take a beat and just do its thing before the service starts looking for whether it's time to render anything.",
			Value:   time.Second,
			Sources: FromConfig("browser.load-wait", "BROWSER_LOAD_WAIT"),
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
		Path:               c.String("browser.path"),
		Flags:              c.StringSlice("browser.flag"),
		GPU:                c.Bool("browser.gpu"),
		Sandbox:            c.Bool("browser.sandbox"),
		TimeZone:           timeZone,
		Cookies:            nil,
		Headers:            headers,
		TimeBetweenScrolls: c.Duration("browser.time-between-scrolls"),
		ReadinessTimeout:   c.Duration("browser.readiness-timeout"),
		LoadWait:           c.Duration("browser.load-wait"),
		MinWidth:           minWidth,
		MinHeight:          minHeight,
		MaxWidth:           maxWidth,
		MaxHeight:          maxHeight,
		PageScaleFactor:    c.Float64("browser.page-scale-factor"),
		Landscape:          !c.Bool("browser.portrait"),
	}, nil
}
