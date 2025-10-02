package traces

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/grafana/grafana-image-renderer/pkg/config"
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

func NewTracerProvider(ctx context.Context, cfg config.TracingConfig) (*sdktrace.TracerProvider, error) {
	if cfg.Endpoint == "" {
		slog.InfoContext(ctx, "no tracing endpoint configured, not setting up tracing")
		return nil, nil
	}

	var tlsCfg *tls.Config
	if cfg.TrustedCertificate != "" || cfg.ClientCertificate != "" || cfg.ClientKey != "" {
		tlsCfg = &tls.Config{}
	}
	if cfg.TrustedCertificate != "" {
		// read file
		basePool, err := x509.SystemCertPool()
		if err != nil {
			// This isn't expected, but is fine; we've been given a certificate, so that should be _enough_... although it doesn't give us great hopes for Chromium's ability to work...
			slog.WarnContext(ctx, "failed to load system cert pool, creating new cert pool", "error", err)
			basePool = x509.NewCertPool()
		}
		certData, err := os.ReadFile(cfg.TrustedCertificate)
		if err != nil {
			return nil, fmt.Errorf("failed to read trusted certificate file at %q: %w", cfg.TrustedCertificate, err)
		}
		if ok := basePool.AppendCertsFromPEM(certData); !ok {
			return nil, fmt.Errorf("failed to parse any PEM certificates from trusted certificate file at %q", cfg.TrustedCertificate)
		}
		tlsCfg.RootCAs = basePool
	}
	if cfg.ClientCertificate != "" && cfg.ClientKey != "" {
		cert, err := tls.LoadX509KeyPair(cfg.ClientCertificate, cfg.ClientKey)
		if err != nil {
			return nil, fmt.Errorf("failed to load client certificate or key from %q and %q: %w", cfg.ClientCertificate, cfg.ClientKey, err)
		}
		tlsCfg.Certificates = []tls.Certificate{cert}
	} else if (cfg.ClientCertificate != "") != (cfg.ClientKey != "") {
		return nil, fmt.Errorf("both client certificate and client key must be set to use mTLS")
	}

	var exporter *otlptrace.Exporter
	if strings.HasPrefix(cfg.Endpoint, "http://") || strings.HasPrefix(cfg.Endpoint, "https://") {
		slog.InfoContext(ctx, "setting up HTTP trace exporter", "endpoint", cfg.Endpoint)
		if cfg.Insecure == nil {
			v := strings.HasPrefix(cfg.Endpoint, "http://")
			cfg.Insecure = &v // force no nil ptr so we can unconditionally deref
		}
		_, cfg.Endpoint, _ = strings.Cut(cfg.Endpoint, "://")
		var urlPath string
		cfg.Endpoint, urlPath, _ = strings.Cut(cfg.Endpoint, "/")
		if urlPath != "" {
			urlPath = "/" + urlPath
		}

		var opts []otlptracehttp.Option
		opts = append(opts, otlptracehttp.WithEndpoint(cfg.Endpoint))
		if urlPath != "" {
			opts = append(opts, otlptracehttp.WithURLPath(urlPath))
		}
		if *cfg.Insecure {
			opts = append(opts, otlptracehttp.WithInsecure())
		}
		if len(cfg.Headers) > 0 {
			opts = append(opts, otlptracehttp.WithHeaders(cfg.Headers))
		}
		if cfg.Compressor != "" && cfg.Compressor != "none" {
			opts = append(opts, otlptracehttp.WithCompression(otlptracehttp.GzipCompression))
		}
		if cfg.Timeout > 0 {
			opts = append(opts, otlptracehttp.WithTimeout(cfg.Timeout))
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
		slog.InfoContext(ctx, "setting up gRPC trace exporter", "endpoint", cfg.Endpoint)
		_, cfg.Endpoint, _ = strings.Cut(cfg.Endpoint, "://")

		var opts []otlptracegrpc.Option
		opts = append(opts, otlptracegrpc.WithEndpoint(cfg.Endpoint))
		if cfg.Insecure != nil && *cfg.Insecure {
			opts = append(opts, otlptracegrpc.WithInsecure())
		}
		if len(cfg.Headers) > 0 {
			opts = append(opts, otlptracegrpc.WithHeaders(cfg.Headers))
		}
		if cfg.Compressor != "" && cfg.Compressor != "none" {
			opts = append(opts, otlptracegrpc.WithCompressor(cfg.Compressor))
		}
		if cfg.Timeout > 0 {
			opts = append(opts, otlptracegrpc.WithTimeout(cfg.Timeout))
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
			semconv.ServiceName(cfg.ServiceName),
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
