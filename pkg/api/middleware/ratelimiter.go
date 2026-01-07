package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"sync/atomic"

	"github.com/grafana/grafana-image-renderer/pkg/config"
	"github.com/grafana/grafana-image-renderer/pkg/service"
	"github.com/pbnjay/memory"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var (
	MetricRateLimiterRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "http_rate_limiter_requests_total",
		Help: "Number of HTTP requests that pass through the rate-limiter, and their outcomes.",
	}, []string{"result", "why"})
	MetricRateLimiterSlots = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "http_rate_limiter_slots",
		Help: "The number of total available slots for handling requests, based on memory.",
	}, []string{"type"})
)

// Limiter unifies the limiter types.
type Limiter interface {
	Limit(http.Handler) http.Handler
}

type noOpLimiter struct{}

func (noOpLimiter) Limit(next http.Handler) http.Handler {
	return next
}

type processBasedLimiter struct {
	svc     *service.ProcessStatService
	cfg     config.RateLimitConfig
	running *atomic.Uint32
	logger  *slog.Logger
}

func (p processBasedLimiter) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tracer := tracer(r.Context())
		ctx, span := tracer.Start(r.Context(), "processBasedLimiter.Limit")
		defer span.End()

		fits, why := p.canFitRequest(ctx)
		span.SetAttributes(attribute.Bool("accepted", fits), attribute.String("reason", why))

		if !fits {
			span.SetStatus(codes.Error, "rate limit exceeded")
			span.SetAttributes(attribute.Bool("accepted", false), attribute.String("reason", why))
			MetricRateLimiterRequests.WithLabelValues("rejected", why).Inc()

			w.Header().Set("Retry-After", "5")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte("server is too busy, try again later"))
			return
		} else {
			p.running.Add(1)
			MetricRateLimiterRequests.WithLabelValues("accepted", why).Inc()
			// From sync.AddUint32:
			// > AddUint32 atomically adds delta to *addr and returns the new value.
			// > To subtract a signed positive constant value c from x, do AddUint32(&x, ^uint32(c-1)).
			// > In particular, to decrement x, do AddUint32(&x, ^uint32(0)).
			// > Consider using the more ergonomic and less error-prone [Uint32.Add] instead.
			defer p.running.Add(^uint32(0)) // decrement

			span.End() // we don't want to track the next middleware in this span
			next.ServeHTTP(w, r)
		}
	})
}

func (p processBasedLimiter) canFitRequest(ctx context.Context) (bool, string) {
	tracer := tracer(ctx)
	_, span := tracer.Start(ctx, "processBasedLimiter.canFitRequest", trace.WithAttributes(
		attribute.Int64("headroom", int64(p.cfg.Headroom)),
		attribute.Int64("min_memory_per_browser", int64(p.cfg.MinMemoryPerBrowser)),
		attribute.Int64("min_limit", int64(p.cfg.MinLimit)),
		attribute.Int64("max_limit", int64(p.cfg.MaxLimit)),
		attribute.Int64("max_available", int64(p.cfg.MaxAvailable))))
	defer span.End()

	currentlyRunning := p.running.Load()
	span.SetAttributes(attribute.Int64("currently_running", int64(currentlyRunning)))
	if currentlyRunning < p.cfg.MinLimit {
		return true, "below minimum limit"
	} else if p.cfg.MaxLimit > 0 && currentlyRunning >= p.cfg.MaxLimit {
		return false, "hit maximum limit"
	}

	totalMemory := memory.TotalMemory()
	if p.cfg.MaxAvailable > 0 && totalMemory > p.cfg.MaxAvailable {
		span.AddEvent("capping total memory to configured maximum")
		totalMemory = p.cfg.MaxAvailable
	}
	freeMemory := memory.FreeMemory()
	span.SetAttributes(
		attribute.Int64("total_memory", int64(totalMemory)),
		attribute.Int64("free_memory", int64(freeMemory)))

	if totalMemory != 0 {
		totalSlots := totalMemory / p.cfg.MinMemoryPerBrowser
		MetricRateLimiterSlots.WithLabelValues("total").Set(float64(totalSlots))
		MetricRateLimiterSlots.WithLabelValues("free").Set(float64(totalSlots - uint64(currentlyRunning)))
		span.SetAttributes(attribute.Int64("total_slots", int64(totalSlots)))
		if currentlyRunning >= uint32(totalSlots) {
			return false, "no memory slots exist based on total memory"
		}
	} else {
		span.AddEvent("unable to determine total memory, skipping total memory slot check")
	}

	if freeMemory != 0 {
		// Calculate whether we have enough for another slot.
		minRequired := max(p.cfg.MinMemoryPerBrowser, uint64(p.svc.PeakMemory))
		span.SetAttributes(attribute.Int64("min_required_per_browser", int64(minRequired)))
		if freeMemory < p.cfg.Headroom {
			return false, "free memory smaller than required headroom"
		} else if freeMemory-p.cfg.Headroom < minRequired {
			return false, "not enough free memory without headroom for another browser"
		}
		// We have enough free memory.
	} else {
		span.AddEvent("unable to determine free memory, skipping free memory check")
	}

	return true, "sufficient memory slots exist"
}

func NewRateLimiter(svc *service.ProcessStatService, cfg config.RateLimitConfig) (Limiter, error) {
	if cfg.Disabled {
		return noOpLimiter{}, nil
	}

	return processBasedLimiter{
		svc:     svc,
		cfg:     cfg,
		running: &atomic.Uint32{},
		logger:  slog.With("middleware", "rate_limiter"),
	}, nil
}
