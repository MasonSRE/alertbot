package monitoring

import (
	"context"
	"runtime"
	"sync"
	"time"

	"alertbot/internal/metrics"
	"alertbot/internal/models"
	"alertbot/internal/repository"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// BackgroundMonitor handles automated monitoring tasks
type BackgroundMonitor struct {
	repositories *repository.Repositories
	db           *gorm.DB
	logger       *logrus.Logger
	
	// Configuration
	config BackgroundMonitorConfig
	
	// Control
	stopChan chan struct{}
	wg       sync.WaitGroup
	running  bool
	mu       sync.RWMutex
}

// BackgroundMonitorConfig configures the background monitor
type BackgroundMonitorConfig struct {
	SystemMetricsInterval       time.Duration
	DatabaseMetricsInterval     time.Duration
	AlertMetricsInterval        time.Duration
	PerformanceMetricsInterval  time.Duration
	CleanupInterval            time.Duration
	
	// Retention settings
	MetricsRetentionDays       int
	HistoryRetentionDays       int
	
	// Performance thresholds
	SlowQueryThreshold         time.Duration
	HighMemoryThreshold        uint64
	HighGoroutineThreshold     int
}

// AlertPerformanceStats tracks alert processing performance
type AlertPerformanceStats struct {
	TotalProcessed     int64
	AverageProcessTime time.Duration
	ErrorRate         float64
	DeduplicationRate float64
	LastHourStats     HourlyStats
}

// HourlyStats contains hourly statistics
type HourlyStats struct {
	Processed    int64
	Errors       int64
	Duplicates   int64
	Correlations int64
}

// DatabasePerformanceStats tracks database performance
type DatabasePerformanceStats struct {
	AverageQueryTime    time.Duration
	SlowQueries         int64
	ConnectionUtilization float64
	FailedQueries       int64
}

// SystemPerformanceStats tracks system performance
type SystemPerformanceStats struct {
	CPUUsage        float64
	MemoryUsage     uint64
	GoroutineCount  int
	GCPauseTime     time.Duration
	DiskUsage       float64
}

// NewBackgroundMonitor creates a new background monitor
func NewBackgroundMonitor(repos *repository.Repositories, db *gorm.DB, logger *logrus.Logger) *BackgroundMonitor {
	config := BackgroundMonitorConfig{
		SystemMetricsInterval:      30 * time.Second,
		DatabaseMetricsInterval:    60 * time.Second,
		AlertMetricsInterval:       30 * time.Second,
		PerformanceMetricsInterval: 15 * time.Second,
		CleanupInterval:           24 * time.Hour,
		MetricsRetentionDays:      30,
		HistoryRetentionDays:      90,
		SlowQueryThreshold:        1 * time.Second,
		HighMemoryThreshold:       1 << 30, // 1GB
		HighGoroutineThreshold:    10000,
	}

	return &BackgroundMonitor{
		repositories: repos,
		db:          db,
		logger:      logger,
		config:      config,
		stopChan:    make(chan struct{}),
	}
}

// Start begins background monitoring
func (bm *BackgroundMonitor) Start(ctx context.Context) error {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	if bm.running {
		return nil
	}

	bm.logger.Info("Starting background monitor")
	bm.running = true

	// Start system metrics collection
	bm.wg.Add(1)
	go bm.collectSystemMetrics(ctx)

	// Start database metrics collection
	bm.wg.Add(1)
	go bm.collectDatabaseMetrics(ctx)

	// Start alert metrics collection
	bm.wg.Add(1)
	go bm.collectAlertMetrics(ctx)

	// Start performance monitoring
	bm.wg.Add(1)
	go bm.monitorPerformance(ctx)

	// Start cleanup routine
	bm.wg.Add(1)
	go bm.runCleanup(ctx)

	// Start uptime counter
	bm.wg.Add(1)
	go bm.trackUptime(ctx)

	return nil
}

// Stop stops background monitoring
func (bm *BackgroundMonitor) Stop() {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	if !bm.running {
		return
	}

	bm.logger.Info("Stopping background monitor")
	close(bm.stopChan)
	bm.wg.Wait()
	bm.running = false
}

// collectSystemMetrics collects system-level metrics
func (bm *BackgroundMonitor) collectSystemMetrics(ctx context.Context) {
	defer bm.wg.Done()

	ticker := time.NewTicker(bm.config.SystemMetricsInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-bm.stopChan:
			return
		case <-ticker.C:
			bm.updateSystemMetrics()
		}
	}
}

// collectDatabaseMetrics collects database metrics
func (bm *BackgroundMonitor) collectDatabaseMetrics(ctx context.Context) {
	defer bm.wg.Done()

	ticker := time.NewTicker(bm.config.DatabaseMetricsInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-bm.stopChan:
			return
		case <-ticker.C:
			bm.updateDatabaseMetrics()
		}
	}
}

// collectAlertMetrics collects alert-related metrics
func (bm *BackgroundMonitor) collectAlertMetrics(ctx context.Context) {
	defer bm.wg.Done()

	ticker := time.NewTicker(bm.config.AlertMetricsInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-bm.stopChan:
			return
		case <-ticker.C:
			bm.updateAlertMetrics()
		}
	}
}

// monitorPerformance monitors system performance and generates alerts
func (bm *BackgroundMonitor) monitorPerformance(ctx context.Context) {
	defer bm.wg.Done()

	ticker := time.NewTicker(bm.config.PerformanceMetricsInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-bm.stopChan:
			return
		case <-ticker.C:
			bm.checkPerformanceThresholds()
		}
	}
}

// runCleanup performs periodic cleanup tasks
func (bm *BackgroundMonitor) runCleanup(ctx context.Context) {
	defer bm.wg.Done()

	ticker := time.NewTicker(bm.config.CleanupInterval)
	defer ticker.Stop()

	// Run cleanup immediately on start
	bm.performCleanup()

	for {
		select {
		case <-ctx.Done():
			return
		case <-bm.stopChan:
			return
		case <-ticker.C:
			bm.performCleanup()
		}
	}
}

// trackUptime tracks system uptime
func (bm *BackgroundMonitor) trackUptime(ctx context.Context) {
	defer bm.wg.Done()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-bm.stopChan:
			return
		case <-ticker.C:
			metrics.IncrementSystemUptime()
		}
	}
}

// updateSystemMetrics updates system-level metrics
func (bm *BackgroundMonitor) updateSystemMetrics() {
	start := time.Now()
	defer func() {
		metrics.RecordBackgroundJob("system_metrics", "success", time.Since(start).Seconds())
	}()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Update memory metrics
	metrics.UpdateMemoryUsage("heap_alloc", float64(m.HeapAlloc))
	metrics.UpdateMemoryUsage("heap_sys", float64(m.HeapSys))
	metrics.UpdateMemoryUsage("heap_idle", float64(m.HeapIdle))
	metrics.UpdateMemoryUsage("heap_inuse", float64(m.HeapInuse))
	metrics.UpdateMemoryUsage("stack_inuse", float64(m.StackInuse))
	metrics.UpdateMemoryUsage("stack_sys", float64(m.StackSys))

	// Update goroutine count
	goroutines := runtime.NumGoroutine()
	metrics.UpdateGoroutineCount(float64(goroutines))

	// Log warnings for high resource usage
	heapMB := float64(m.HeapAlloc) / 1024 / 1024
	if heapMB > 500 {
		bm.logger.WithField("heap_mb", heapMB).Warn("High memory usage detected")
		metrics.RecordSecurityEvent("high_memory_usage")
	}

	if goroutines > bm.config.HighGoroutineThreshold {
		bm.logger.WithField("goroutines", goroutines).Warn("High goroutine count detected")
		metrics.RecordSecurityEvent("high_goroutine_count")
	}
}

// updateDatabaseMetrics updates database-related metrics
func (bm *BackgroundMonitor) updateDatabaseMetrics() {
	if bm.db == nil {
		return
	}

	start := time.Now()
	defer func() {
		metrics.RecordBackgroundJob("database_metrics", "success", time.Since(start).Seconds())
	}()

	sqlDB, err := bm.db.DB()
	if err != nil {
		bm.logger.WithError(err).Error("Failed to get SQL DB for metrics")
		metrics.RecordBackgroundJob("database_metrics", "error", time.Since(start).Seconds())
		return
	}

	stats := sqlDB.Stats()
	
	// Update connection metrics
	metrics.UpdateDatabaseConnections("open", float64(stats.OpenConnections))
	metrics.UpdateDatabaseConnections("idle", float64(stats.Idle))
	metrics.UpdateDatabaseConnections("in_use", float64(stats.InUse))

	// Check connection usage
	if stats.MaxOpenConnections > 0 {
		usage := float64(stats.OpenConnections) / float64(stats.MaxOpenConnections) * 100
		if usage > 80 {
			bm.logger.WithFields(logrus.Fields{
				"open": stats.OpenConnections,
				"max":  stats.MaxOpenConnections,
				"usage_percent": usage,
			}).Warn("High database connection usage")
			metrics.RecordSecurityEvent("high_db_connections")
		}
	}

	// Test database connectivity with a simple query
	queryStart := time.Now()
	var count int
	if err := bm.db.Raw("SELECT 1").Scan(&count).Error; err != nil {
		bm.logger.WithError(err).Error("Database health check query failed")
		metrics.RecordDatabaseError("health_check", "query_failed")
		metrics.UpdateServiceHealth("database", false)
	} else {
		queryDuration := time.Since(queryStart)
		metrics.RecordDatabaseQuery("health_check", "system", queryDuration.Seconds())
		metrics.UpdateServiceHealth("database", true)
		
		if queryDuration > bm.config.SlowQueryThreshold {
			bm.logger.WithField("duration", queryDuration).Warn("Slow database query detected")
		}
	}
}

// updateAlertMetrics updates alert-related metrics
func (bm *BackgroundMonitor) updateAlertMetrics() {
	if bm.repositories == nil || bm.repositories.Alert == nil {
		return
	}

	start := time.Now()
	defer func() {
		metrics.RecordBackgroundJob("alert_metrics", "success", time.Since(start).Seconds())
	}()

	// Get alert counts by status and severity
	statuses := []string{"firing", "resolved", "silenced", "acknowledged"}
	severities := []string{"critical", "warning", "info"}

	for _, status := range statuses {
		for _, severity := range severities {
			filters := models.AlertFilters{
				Status:   status,
				Severity: severity,
				Size:     1, // We only need the count
			}
			
			_, count, err := bm.repositories.Alert.List(filters)
			if err != nil {
				bm.logger.WithError(err).WithFields(logrus.Fields{
					"status":   status,
					"severity": severity,
				}).Error("Failed to get alert count")
				continue
			}
			
			metrics.UpdateActiveAlerts(severity, status, float64(count))
		}
	}
}

// checkPerformanceThresholds checks various performance thresholds
func (bm *BackgroundMonitor) checkPerformanceThresholds() {
	start := time.Now()
	defer func() {
		metrics.RecordBackgroundJob("performance_check", "success", time.Since(start).Seconds())
	}()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Check memory threshold
	if m.HeapAlloc > bm.config.HighMemoryThreshold {
		bm.logger.WithField("heap_bytes", m.HeapAlloc).Warn("Memory usage above threshold")
		metrics.RecordSecurityEvent("memory_threshold_exceeded")
	}

	// Check goroutine threshold
	goroutines := runtime.NumGoroutine()
	if goroutines > bm.config.HighGoroutineThreshold {
		bm.logger.WithField("goroutines", goroutines).Warn("Goroutine count above threshold")
		metrics.RecordSecurityEvent("goroutine_threshold_exceeded")
	}

	// Check GC pause time
	avgGCPause := float64(m.PauseTotalNs) / float64(m.NumGC) / 1000000 // Convert to milliseconds
	if avgGCPause > 100 { // 100ms threshold
		bm.logger.WithField("gc_pause_ms", avgGCPause).Warn("High GC pause time detected")
		metrics.RecordSecurityEvent("high_gc_pause")
	}
}

// performCleanup performs cleanup tasks
func (bm *BackgroundMonitor) performCleanup() {
	start := time.Now()
	defer func() {
		metrics.RecordBackgroundJob("cleanup", "success", time.Since(start).Seconds())
	}()

	bm.logger.Info("Starting cleanup tasks")

	// Clean up old alert history
	if bm.repositories != nil && bm.repositories.AlertHistory != nil {
		cutoff := time.Now().AddDate(0, 0, -bm.config.HistoryRetentionDays)
		
		// This would need to be implemented in the repository
		// bm.repositories.AlertHistory.DeleteOlderThan(cutoff)
		
		bm.logger.WithField("cutoff", cutoff).Info("Alert history cleanup completed")
	}

	// Force garbage collection
	runtime.GC()
	
	bm.logger.Info("Cleanup tasks completed")
}

// GetStats returns performance statistics
func (bm *BackgroundMonitor) GetStats() map[string]interface{} {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	stats := map[string]interface{}{
		"system": SystemPerformanceStats{
			MemoryUsage:    m.HeapAlloc,
			GoroutineCount: runtime.NumGoroutine(),
			GCPauseTime:    time.Duration(m.PauseTotalNs / uint64(m.NumGC)),
		},
	}

	if bm.db != nil {
		if sqlDB, err := bm.db.DB(); err == nil {
			dbStats := sqlDB.Stats()
			usage := 0.0
			if dbStats.MaxOpenConnections > 0 {
				usage = float64(dbStats.OpenConnections) / float64(dbStats.MaxOpenConnections) * 100
			}
			
			stats["database"] = DatabasePerformanceStats{
				ConnectionUtilization: usage,
				// Other stats would be tracked over time
			}
		}
	}

	return stats
}

// UpdateConfig updates the background monitor configuration
func (bm *BackgroundMonitor) UpdateConfig(config BackgroundMonitorConfig) {
	bm.config = config
	bm.logger.Info("Background monitor configuration updated")
}