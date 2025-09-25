package traces

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/grafana/grafana-image-renderer/cmd/config"
	"github.com/urfave/cli/v3"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
	"google.golang.org/grpc/credentials"
)

type tracerProviderContextKey int

var tracerProviderKey tracerProviderContextKey

func TracerFromContext(ctx context.Context, name string) trace.Tracer {
	if t, ok := ctx.Value(tracerProviderKey).(trace.TracerProvider); ok && t != nil {
		return t.Tracer(name)
	}
	return noop.Tracer{}
}

func WithTracerProvider(ctx context.Context, t trace.TracerProvider) context.Context {
	return context.WithValue(ctx, tracerProviderKey, t)
}

func TracerFlags() []cli.Flag {
	const category = "Tracing"
	return []cli.Flag{
		&cli.StringFlag{
			Name:     "tracing-endpoint",
			Usage:    "The tracing endpoint to send spans to. Defaults to gRPC; if http:// or https:// is specified, HTTP will be used instead. grpc:// is fine but redundant.",
			Sources:  config.FromConfig("tracing.endpoint", "TRACING_ENDPOINT"),
			Category: category,
			Validator: func(s string) error {
				before, _, found := strings.Cut(s, "://")
				if !found {
					return nil
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
			Name:     "tracing-insecure",
			Usage:    "Whether to skip TLS verification when connecting. If set, the scheme in the endpoint is overridden to be insecure.",
			Category: category,
			Sources:  config.FromConfig("tracing.insecure", "TRACING_INSECURE"),
		},
		&cli.StringSliceFlag{
			Name:     "tracing-header",
			Usage:    "A header to add to requests to the tracing endpoint. Syntax is `<key>=<value>`. May be repeated. This is useful for things like authentication.",
			Category: category,
			Sources:  config.FromConfig("tracing.headers", "TRACING_HEADERS"),
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
			Name:     "tracing-compressor",
			Usage:    "The compression algorithm to use when sending traces. (enum: none, gzip)",
			Value:    "none",
			Category: category,
			Sources:  config.FromConfig("tracing.compressor", "TRACING_COMPRESSOR"),
			Validator: func(s string) error {
				if s == "" || s == "none" || s == "gzip" {
					return nil
				}
				return fmt.Errorf("invalid tracing compressor: %s", s)
			},
		},
		&cli.DurationFlag{
			Name:     "tracing-timeout",
			Usage:    "The timeout for requests to the tracing endpoint.",
			Value:    10 * time.Second,
			Category: category,
			Sources:  config.FromConfig("tracing.timeout", "TRACING_TIMEOUT"),
		},
		&cli.StringFlag{
			Name:      "tracing-trusted-certificate",
			Usage:     "A path to a PEM-encoded certificate to use as a trusted root when connecting to the tracing endpoint over gRPC or HTTPS.",
			Category:  category,
			TakesFile: true,
			Sources:   config.FromConfig("tracing.trusted_certificate", "TRACING_TRUSTED_CERTIFICATE"),
		},
		&cli.StringFlag{
			Name:      "tracing-client-certificate",
			Usage:     "A path to a PEM-encoded client certificate to use for mTLS when connecting to the tracing endpoint over gRPC or HTTPS.",
			Category:  category,
			TakesFile: true,
			Sources:   config.FromConfig("tracing.client_certificate", "TRACING_CLIENT_CERTIFICATE"),
		},
		&cli.StringFlag{
			Name:      "tracing-client-key",
			Usage:     "A path to a PEM-encoded client key to use for mTLS when connecting to the tracing endpoint over gRPC or HTTPS.",
			Category:  category,
			TakesFile: true,
			Sources:   config.FromConfig("tracing.client_key", "TRACING_CLIENT_KEY"),
		},
		&cli.StringFlag{
			Name:     "tracing-service-name",
			Usage:    "The service name to use in traces.",
			Category: category,
			Value:    "grafana-image-renderer",
			Sources:  config.FromConfig("tracing.service_name", "TRACING_SERVICE_NAME"),
		},
	}
}

func NewTracerProvider(ctx context.Context, c *cli.Command) (*sdktrace.TracerProvider, error) {
	endpoint := c.String("tracing-endpoint")
	if endpoint == "" {
		slog.InfoContext(ctx, "no tracing endpoint configured, not setting up tracing")
		return nil, nil
	}

	var insecure *bool
	if c.IsSet("tracing-insecure") {
		v := c.Bool("tracing-insecure")
		insecure = &v
	}
	var headers map[string]string
	if hs := c.StringSlice("tracing-header"); len(hs) > 0 {
		headers = make(map[string]string, len(hs))
		for _, kv := range hs {
			k, v, _ := strings.Cut(kv, "=") // validated in the cli flag
			headers[k] = v
		}
	}
	compressor := c.String("tracing-compressor")
	timeout := c.Duration("tracing-timeout")
	trustedCertificate := c.String("tracing-trusted-certificate")
	clientCertificate := c.String("tracing-client-certificate")
	clientKey := c.String("tracing-client-key")

	var tlsCfg *tls.Config
	if trustedCertificate != "" || clientCertificate != "" || clientKey != "" {
		tlsCfg = &tls.Config{}
	}
	if trustedCertificate != "" {
		// read file
		basePool, err := x509.SystemCertPool()
		if err != nil {
			// This isn't expected, but is fine; we've been given a certificate, so that should be _enough_... although it doesn't give us great hopes for Chromium's ability to work...
			slog.WarnContext(ctx, "failed to load system cert pool, creating new cert pool", "error", err)
			basePool = x509.NewCertPool()
		}
		certData, err := os.ReadFile(trustedCertificate)
		if err != nil {
			return nil, fmt.Errorf("failed to read trusted certificate file at %q: %w", trustedCertificate, err)
		}
		if ok := basePool.AppendCertsFromPEM(certData); !ok {
			return nil, fmt.Errorf("failed to parse any PEM certificates from trusted certificate file at %q", trustedCertificate)
		}
		tlsCfg.RootCAs = basePool
	}
	if clientCertificate != "" && clientKey != "" {
		cert, err := tls.LoadX509KeyPair(clientCertificate, clientKey)
		if err != nil {
			return nil, fmt.Errorf("failed to load client certificate or key from %q and %q: %w", clientCertificate, clientKey, err)
		}
		tlsCfg.Certificates = []tls.Certificate{cert}
	} else if (clientCertificate != "") != (clientKey != "") {
		return nil, fmt.Errorf("both client certificate and client key must be set to use mTLS")
	}

	var exporter *otlptrace.Exporter
	if strings.HasPrefix(endpoint, "http://") || strings.HasPrefix(endpoint, "https://") {
		slog.InfoContext(ctx, "setting up HTTP trace exporter", "endpoint", endpoint)
		if insecure == nil {
			v := strings.HasPrefix(endpoint, "http://")
			insecure = &v // force no nil ptr so we can unconditionally deref
		}
		_, endpoint, _ = strings.Cut(endpoint, "://")
		endpoint, urlPath, _ := strings.Cut(endpoint, "/")
		if urlPath != "" {
			urlPath = "/" + urlPath
		}

		var opts []otlptracehttp.Option
		opts = append(opts, otlptracehttp.WithEndpoint(endpoint))
		if urlPath != "" {
			opts = append(opts, otlptracehttp.WithURLPath(urlPath))
		}
		if *insecure {
			opts = append(opts, otlptracehttp.WithInsecure())
		}
		if len(headers) > 0 {
			opts = append(opts, otlptracehttp.WithHeaders(headers))
		}
		if compressor != "" && compressor != "none" {
			opts = append(opts, otlptracehttp.WithCompression(otlptracehttp.GzipCompression))
		}
		if timeout > 0 {
			opts = append(opts, otlptracehttp.WithTimeout(timeout))
		}
		if tlsCfg != nil {
			opts = append(opts, otlptracehttp.WithTLSClientConfig(tlsCfg))
		}

		var err error
		exporter, err = otlptracehttp.New(ctx, opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to create HTTP trace exporter: %w", err)
		}
	} else { // gRPC
		slog.InfoContext(ctx, "setting up gRPC trace exporter", "endpoint", endpoint)
		_, endpoint, _ = strings.Cut(endpoint, "://")

		var opts []otlptracegrpc.Option
		opts = append(opts, otlptracegrpc.WithEndpoint(endpoint))
		if insecure != nil && *insecure {
			opts = append(opts, otlptracegrpc.WithInsecure())
		}
		if len(headers) > 0 {
			opts = append(opts, otlptracegrpc.WithHeaders(headers))
		}
		if compressor != "" && compressor != "none" {
			opts = append(opts, otlptracegrpc.WithCompressor(compressor))
		}
		if timeout > 0 {
			opts = append(opts, otlptracegrpc.WithTimeout(timeout))
		}
		if tlsCfg != nil {
			opts = append(opts, otlptracegrpc.WithTLSCredentials(credentials.NewTLS(tlsCfg)))
		}

		var err error
		exporter, err = otlptracegrpc.New(ctx, opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to create gRPC trace exporter: %w", err)
		}
	}

	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(c.String("tracing-service-name")),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource for tracer: %w", err)
	}

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	return tracerProvider, nil
}
