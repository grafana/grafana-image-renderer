package service

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/grafana/grafana-image-renderer/pkg/config"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/shirou/gopsutil/v4/process"
)

var (
	MetricProcessMaxMemory = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "process_max_memory",
		ConstLabels: prometheus.Labels{
			"unit": "bytes",
		},
		Help: "Maximum memory used by the Chromium process in bytes. This is the max of all tracked processes.",
	})
	MetricProcessPeakMemoryAverage = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "process_peak_memory_avg",
		ConstLabels: prometheus.Labels{
			"unit": "bytes",
		},
		Help: "Peak memory used by the Chromium process in bytes. This is a slow-moving average of all tracked processes.",
	})
	MetricProcessPeakMemoryInstant = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name: "process_peak_memory",
		ConstLabels: prometheus.Labels{
			"unit": "bytes",
		},
		Help: "Peak memory used by any Chromium process in bytes. This is marked for every process.",
		Buckets: []float64{
			kibibyte,
			mibibyte,
			16 * mibibyte,
			32 * mibibyte,
			64 * mibibyte,
			128 * mibibyte,
			256 * mibibyte,
			368 * mibibyte,
			512 * mibibyte,
			768 * mibibyte,
			gibibyte,
			gibibyte + 512*mibibyte,
			2 * gibibyte,
			4 * gibibyte,
			6 * gibibyte,
			8 * gibibyte,
		},
	})
)

const (
	kibibyte = 1024
	mibibyte = 1024 * kibibyte
	gibibyte = 1024 * mibibyte
)

type ProcessStatService struct {
	cfg config.RateLimitConfig

	mu sync.Mutex
	// MaxMemory is the number of bytes a Chromium process uses at absolute max.
	// This is the max of all processes.
	MaxMemory int64
	// PeakMemory is the number of bytes a Chromium process uses at peak.
	// It is a slow-moving average of all values tracked.
	PeakMemory int64

	log *slog.Logger
}

func NewProcessStatService(cfg config.RateLimitConfig) *ProcessStatService {
	return &ProcessStatService{
		cfg: cfg,
		log: slog.With("service", "process_stat"),
	}
}

// TrackProcess starts a new goroutine to keep track of the process.
//
// We need to track all child-processes alongside the main PID we're given.
func (p *ProcessStatService) TrackProcess(ctx context.Context, pid int32) {
	go func() {
		logger := p.log.With("pid", pid)

		proc, err := process.NewProcessWithContext(ctx, pid)
		if errors.Is(err, context.Canceled) {
			return
		} else if err != nil {
			logger.Warn("failed to find process to track", "err", err)
			return
		}

		var peakMemory int64
		defer func() {
			// We only do the lock once per process. This reduces contention significantly.
			p.mu.Lock()
			defer p.mu.Unlock()

			if p.PeakMemory == 0 {
				p.PeakMemory = peakMemory
			} else {
				p.PeakMemory = (p.PeakMemory*(p.cfg.TrackerDecay-1) + peakMemory) / p.cfg.TrackerDecay
			}
			p.MaxMemory = max(p.MaxMemory, peakMemory)

			MetricProcessMaxMemory.Set(float64(p.MaxMemory))
			MetricProcessPeakMemoryAverage.Set(float64(p.PeakMemory))
			MetricProcessPeakMemoryInstant.Observe(float64(peakMemory))
		}()

		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(p.cfg.TrackerInterval):
			}

			if running, _ := proc.IsRunningWithContext(ctx); !running {
				return
			}

			peakMemory = max(peakMemory, recursiveMemory(ctx, proc))
		}
	}()
}

// recursiveMemory calculates the total memory used by a process and all its children.
// This is a best-effort function and may return partial results if processes exit while being queried, or are inaccessible to the current process.
// We don't return any errors, and silently will just return a bad value if this is the case. This is good _enough_ for our use case.
func recursiveMemory(ctx context.Context, proc *process.Process) int64 {
	mem, err := proc.MemoryInfoWithContext(ctx)
	if err != nil {
		return 0
	}

	sum := int64(mem.RSS)
	// We don't care about errors here. If we get no children, we'll just move on.
	children, _ := proc.ChildrenWithContext(ctx)
	for _, child := range children {
		sum += recursiveMemory(ctx, child)
	}
	return sum
}
