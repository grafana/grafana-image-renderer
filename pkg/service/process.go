package service

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

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
	mu sync.Mutex
	// MaxMemory is the number of bytes a Chromium process uses at absolute max.
	// This is the max of all processes.
	MaxMemory int64
	// PeakMemory is the number of bytes a Chromium process uses at peak.
	// It is a slow-moving average of all values tracked.
	PeakMemory int64

	log *slog.Logger
}

func NewProcessStatService() *ProcessStatService {
	return &ProcessStatService{
		log: slog.With("service", "process_stat"),
	}
}

// TrackProcess starts a new goroutine to keep track of the process.
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

		peakMemory := 0
		defer func() {
			// We only do the lock once per process. This reduces contention significantly.
			p.mu.Lock()
			defer p.mu.Unlock()

			const decay = 5
			p.PeakMemory = (p.PeakMemory*(decay-1) + int64(peakMemory)) / decay
			p.MaxMemory = max(p.MaxMemory, int64(peakMemory))
			MetricProcessMaxMemory.Set(float64(p.MaxMemory))
			MetricProcessPeakMemoryAverage.Set(float64(p.PeakMemory))
			MetricProcessPeakMemoryInstant.Observe(float64(peakMemory))
		}()

		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(500 * time.Millisecond):
			}

			mem, err := proc.MemoryInfoWithContext(ctx)
			if errors.Is(err, context.Canceled) || errors.Is(err, process.ErrorProcessNotRunning) {
				return
			} else if err != nil {
				logger.Warn("failed to find memory info about process", "err", err)
				return
			}

			peakMemory = max(peakMemory, int(mem.RSS))
		}
	}()
}
