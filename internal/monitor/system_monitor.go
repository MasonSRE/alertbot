package monitor

import (
	"context"
	"runtime"
	"time"

	"alertbot/internal/metrics"

	"github.com/sirupsen/logrus"
)

// SystemMonitor monitors system resources and updates metrics
type SystemMonitor struct {
	logger   *logrus.Logger
	interval time.Duration
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewSystemMonitor creates a new system monitor
func NewSystemMonitor(logger *logrus.Logger, interval time.Duration) *SystemMonitor {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &SystemMonitor{
		logger:   logger,
		interval: interval,
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Start starts the system monitor
func (s *SystemMonitor) Start() {
	s.logger.Info("Starting system monitor")
	
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	// Update metrics immediately
	s.updateMetrics()

	for {
		select {
		case <-s.ctx.Done():
			s.logger.Info("System monitor stopped")
			return
		case <-ticker.C:
			s.updateMetrics()
		}
	}
}

// Stop stops the system monitor
func (s *SystemMonitor) Stop() {
	s.logger.Info("Stopping system monitor")
	s.cancel()
}

// updateMetrics collects and updates system metrics
func (s *SystemMonitor) updateMetrics() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Memory metrics
	metrics.UpdateMemoryUsage("heap_alloc", float64(m.HeapAlloc))
	metrics.UpdateMemoryUsage("heap_sys", float64(m.HeapSys))
	metrics.UpdateMemoryUsage("heap_idle", float64(m.HeapIdle))
	metrics.UpdateMemoryUsage("heap_inuse", float64(m.HeapInuse))
	metrics.UpdateMemoryUsage("stack_inuse", float64(m.StackInuse))
	metrics.UpdateMemoryUsage("stack_sys", float64(m.StackSys))

	// Goroutine count
	metrics.UpdateGoroutineCount(float64(runtime.NumGoroutine()))

	// GC metrics
	metrics.UpdateMemoryUsage("gc_sys", float64(m.GCSys))
	metrics.UpdateMemoryUsage("next_gc", float64(m.NextGC))

	// Increment uptime
	metrics.IncrementSystemUptime()

	s.logger.WithFields(logrus.Fields{
		"heap_alloc":    m.HeapAlloc / 1024 / 1024, // MB
		"heap_sys":      m.HeapSys / 1024 / 1024,   // MB
		"goroutines":    runtime.NumGoroutine(),
		"gc_runs":       m.NumGC,
	}).Debug("System metrics updated")
}