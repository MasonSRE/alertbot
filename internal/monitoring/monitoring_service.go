package monitoring

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"alertbot/internal/metrics"
	"alertbot/internal/models"
	"alertbot/internal/repository"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// MonitoringService provides comprehensive system monitoring
type MonitoringService struct {
	repositories *repository.Repositories
	logger       *logrus.Logger
	db           *gorm.DB
	
	// Health check components
	healthCheckers map[string]HealthChecker
	mu             sync.RWMutex
	
	// Monitoring configuration
	config MonitoringConfig
	
	// Background monitoring
	stopChan chan struct{}
	wg       sync.WaitGroup
}

// MonitoringConfig holds monitoring configuration
type MonitoringConfig struct {
	HealthCheckInterval     time.Duration
	MetricsCollectionInterval time.Duration
	AlertThresholds         AlertThresholds
	EnableSystemMetrics     bool
	EnableDatabaseMetrics   bool
	EnableServiceMetrics    bool
}

// AlertThresholds defines when to generate alerts
type AlertThresholds struct {
	HighMemoryUsage      float64 // MB
	HighCPUUsage        float64 // Percentage
	HighDatabaseConnections int
	HighResponseTime    time.Duration
	LowDiskSpace        float64 // Percentage
}

// HealthChecker interface for health check components
type HealthChecker interface {
	CheckHealth(ctx context.Context) HealthStatus
	Name() string
}

// HealthStatus represents the health status of a component
type HealthStatus struct {
	Name      string                 `json:"name"`
	Status    string                 `json:"status"` // healthy, unhealthy, degraded
	Message   string                 `json:"message,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Duration  time.Duration          `json:"duration"`
}

// SystemHealth represents overall system health
type SystemHealth struct {
	Status       string                    `json:"status"`
	Timestamp    time.Time                 `json:"timestamp"`
	Components   map[string]HealthStatus   `json:"components"`
	SystemInfo   SystemInfo               `json:"system_info"`
	Metrics      SystemMetrics            `json:"metrics"`
	Alerts       []HealthAlert            `json:"alerts,omitempty"`
}

// SystemInfo contains system information
type SystemInfo struct {
	Version       string    `json:"version"`
	Uptime        time.Duration `json:"uptime"`
	StartTime     time.Time `json:"start_time"`
	GoVersion     string    `json:"go_version"`
	NumCPU        int       `json:"num_cpu"`
	NumGoroutines int       `json:"num_goroutines"`
}

// SystemMetrics contains key system metrics
type SystemMetrics struct {
	MemoryUsage      MemoryMetrics     `json:"memory"`
	DatabaseMetrics  DatabaseMetrics   `json:"database"`
	AlertMetrics     AlertMetrics      `json:"alerts"`
	APIMetrics       APIMetrics        `json:"api"`
}

// MemoryMetrics contains memory usage information
type MemoryMetrics struct {
	HeapAlloc    uint64  `json:"heap_alloc"`
	HeapSys      uint64  `json:"heap_sys"`
	HeapIdle     uint64  `json:"heap_idle"`
	HeapInuse    uint64  `json:"heap_inuse"`
	StackInuse   uint64  `json:"stack_inuse"`
	StackSys     uint64  `json:"stack_sys"`
	GCPause      float64 `json:"gc_pause_ns"`
	NumGC        uint32  `json:"num_gc"`
}

// DatabaseMetrics contains database metrics
type DatabaseMetrics struct {
	OpenConnections int           `json:"open_connections"`
	InUse          int           `json:"in_use"`
	Idle           int           `json:"idle"`
	WaitDuration   time.Duration `json:"wait_duration"`
	MaxLifetime    time.Duration `json:"max_lifetime"`
}

// AlertMetrics contains alert-related metrics
type AlertMetrics struct {
	TotalAlerts    int64 `json:"total_alerts"`
	FiringAlerts   int64 `json:"firing_alerts"`
	ResolvedAlerts int64 `json:"resolved_alerts"`
	AlertsPerHour  int64 `json:"alerts_per_hour"`
}

// APIMetrics contains API performance metrics
type APIMetrics struct {
	RequestsPerSecond float64       `json:"requests_per_second"`
	AvgResponseTime   time.Duration `json:"avg_response_time"`
	ErrorRate         float64       `json:"error_rate"`
}

// HealthAlert represents a health-related alert
type HealthAlert struct {
	Type        string    `json:"type"`
	Severity    string    `json:"severity"`
	Message     string    `json:"message"`
	Component   string    `json:"component"`
	Timestamp   time.Time `json:"timestamp"`
	Value       float64   `json:"value,omitempty"`
	Threshold   float64   `json:"threshold,omitempty"`
}

var startTime = time.Now()

// NewMonitoringService creates a new monitoring service
func NewMonitoringService(repos *repository.Repositories, logger *logrus.Logger, db *gorm.DB) *MonitoringService {
	config := MonitoringConfig{
		HealthCheckInterval:       30 * time.Second,
		MetricsCollectionInterval: 15 * time.Second,
		AlertThresholds: AlertThresholds{
			HighMemoryUsage:         500, // MB
			HighCPUUsage:           80,   // %
			HighDatabaseConnections: 90,
			HighResponseTime:       5 * time.Second,
			LowDiskSpace:           10, // %
		},
		EnableSystemMetrics:   true,
		EnableDatabaseMetrics: true,
		EnableServiceMetrics:  true,
	}

	ms := &MonitoringService{
		repositories:   repos,
		logger:        logger,
		db:           db,
		healthCheckers: make(map[string]HealthChecker),
		config:       config,
		stopChan:     make(chan struct{}),
	}

	// Register default health checkers
	ms.RegisterHealthChecker(NewDatabaseHealthChecker(db))
	ms.RegisterHealthChecker(NewSystemHealthChecker())
	ms.RegisterHealthChecker(NewAlertServiceHealthChecker(repos))

	return ms
}

// RegisterHealthChecker registers a health checker
func (ms *MonitoringService) RegisterHealthChecker(checker HealthChecker) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.healthCheckers[checker.Name()] = checker
}

// Start starts the monitoring service
func (ms *MonitoringService) Start(ctx context.Context) error {
	ms.logger.Info("Starting monitoring service")

	// Start background metrics collection
	if ms.config.EnableSystemMetrics {
		ms.wg.Add(1)
		go ms.collectSystemMetrics(ctx)
	}

	// Start health check routine
	ms.wg.Add(1)
	go ms.runHealthChecks(ctx)

	return nil
}

// Stop stops the monitoring service
func (ms *MonitoringService) Stop() {
	ms.logger.Info("Stopping monitoring service")
	close(ms.stopChan)
	ms.wg.Wait()
}

// GetSystemHealth returns the current system health status
func (ms *MonitoringService) GetSystemHealth(ctx context.Context) (*SystemHealth, error) {
	start := time.Now()
	defer func() {
		metrics.RecordServiceResponseTime("monitoring", time.Since(start).Seconds())
	}()

	health := &SystemHealth{
		Timestamp:  time.Now(),
		Components: make(map[string]HealthStatus),
		SystemInfo: ms.getSystemInfo(),
		Metrics:    ms.getSystemMetrics(ctx),
	}

	// Run health checks
	overallHealthy := true
	ms.mu.RLock()
	for name, checker := range ms.healthCheckers {
		status := checker.CheckHealth(ctx)
		health.Components[name] = status
		
		if status.Status != "healthy" {
			overallHealthy = false
		}
	}
	ms.mu.RUnlock()

	// Determine overall status
	if overallHealthy {
		health.Status = "healthy"
	} else {
		health.Status = "unhealthy"
	}

	// Generate health alerts
	health.Alerts = ms.generateHealthAlerts(health)

	// Update service health metrics
	for _, component := range health.Components {
		healthy := component.Status == "healthy"
		metrics.UpdateServiceHealth(component.Name, healthy)
	}

	return health, nil
}

// collectSystemMetrics collects system metrics in background
func (ms *MonitoringService) collectSystemMetrics(ctx context.Context) {
	defer ms.wg.Done()
	
	ticker := time.NewTicker(ms.config.MetricsCollectionInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ms.stopChan:
			return
		case <-ticker.C:
			ms.updateSystemMetrics()
		}
	}
}

// runHealthChecks runs health checks periodically
func (ms *MonitoringService) runHealthChecks(ctx context.Context) {
	defer ms.wg.Done()
	
	ticker := time.NewTicker(ms.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ms.stopChan:
			return
		case <-ticker.C:
			ms.performHealthChecks(ctx)
		}
	}
}

// updateSystemMetrics updates system metrics
func (ms *MonitoringService) updateSystemMetrics() {
	// Update memory metrics
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	metrics.UpdateMemoryUsage("heap", float64(m.HeapAlloc))
	metrics.UpdateMemoryUsage("stack", float64(m.StackInuse))
	metrics.UpdateMemoryUsage("gc", float64(m.PauseTotalNs))
	
	// Update goroutine count
	metrics.UpdateGoroutineCount(float64(runtime.NumGoroutine()))
	
	// Update database metrics if available
	if ms.config.EnableDatabaseMetrics && ms.db != nil {
		sqlDB, err := ms.db.DB()
		if err == nil {
			stats := sqlDB.Stats()
			metrics.UpdateDatabaseConnections("open", float64(stats.OpenConnections))
			metrics.UpdateDatabaseConnections("idle", float64(stats.Idle))
			metrics.UpdateDatabaseConnections("in_use", float64(stats.InUse))
		}
	}
}

// performHealthChecks performs all registered health checks
func (ms *MonitoringService) performHealthChecks(ctx context.Context) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	
	for name, checker := range ms.healthCheckers {
		start := time.Now()
		status := checker.CheckHealth(ctx)
		duration := time.Since(start)
		
		// Log health check results
		if status.Status != "healthy" {
			ms.logger.WithFields(logrus.Fields{
				"component": name,
				"status":    status.Status,
				"message":   status.Message,
				"duration":  duration,
			}).Warn("Health check failed")
		}
		
		// Update metrics
		healthy := status.Status == "healthy"
		metrics.UpdateServiceHealth(name, healthy)
		metrics.RecordServiceResponseTime(name, duration.Seconds())
	}
}

// getSystemInfo returns system information
func (ms *MonitoringService) getSystemInfo() SystemInfo {
	return SystemInfo{
		Version:       "1.0.0",
		Uptime:        time.Since(startTime),
		StartTime:     startTime,
		GoVersion:     runtime.Version(),
		NumCPU:        runtime.NumCPU(),
		NumGoroutines: runtime.NumGoroutine(),
	}
}

// getSystemMetrics returns current system metrics
func (ms *MonitoringService) getSystemMetrics(ctx context.Context) SystemMetrics {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	memMetrics := MemoryMetrics{
		HeapAlloc:  m.HeapAlloc,
		HeapSys:    m.HeapSys,
		HeapIdle:   m.HeapIdle,
		HeapInuse:  m.HeapInuse,
		StackInuse: m.StackInuse,
		StackSys:   m.StackSys,
		GCPause:    float64(m.PauseTotalNs),
		NumGC:      m.NumGC,
	}
	
	dbMetrics := DatabaseMetrics{}
	if ms.db != nil {
		if sqlDB, err := ms.db.DB(); err == nil {
			stats := sqlDB.Stats()
			dbMetrics = DatabaseMetrics{
				OpenConnections: stats.OpenConnections,
				InUse:          stats.InUse,
				Idle:           stats.Idle,
				WaitDuration:   stats.WaitDuration,
				MaxLifetime:    0, // MaxLifetime field not available in this Go version
			}
		}
	}
	
	alertMetrics := ms.getAlertMetrics(ctx)
	
	return SystemMetrics{
		MemoryUsage:     memMetrics,
		DatabaseMetrics: dbMetrics,
		AlertMetrics:    alertMetrics,
		APIMetrics:      APIMetrics{}, // Would be populated with actual API metrics
	}
}

// getAlertMetrics returns alert-related metrics
func (ms *MonitoringService) getAlertMetrics(ctx context.Context) AlertMetrics {
	if ms.repositories == nil || ms.repositories.Alert == nil {
		return AlertMetrics{}
	}
	
	// Get total alerts
	allFilters := models.AlertFilters{Size: 1}
	_, total, _ := ms.repositories.Alert.List(allFilters)
	
	// Get firing alerts
	firingFilters := models.AlertFilters{Status: "firing", Size: 1}
	_, firing, _ := ms.repositories.Alert.List(firingFilters)
	
	// Get resolved alerts
	resolvedFilters := models.AlertFilters{Status: "resolved", Size: 1}
	_, resolved, _ := ms.repositories.Alert.List(resolvedFilters)
	
	return AlertMetrics{
		TotalAlerts:    total,
		FiringAlerts:   firing,
		ResolvedAlerts: resolved,
		AlertsPerHour:  0, // Would require time-based aggregation
	}
}

// generateHealthAlerts generates alerts based on thresholds
func (ms *MonitoringService) generateHealthAlerts(health *SystemHealth) []HealthAlert {
	var alerts []HealthAlert
	
	// Check memory usage
	heapMB := float64(health.Metrics.MemoryUsage.HeapAlloc) / 1024 / 1024
	if heapMB > ms.config.AlertThresholds.HighMemoryUsage {
		alerts = append(alerts, HealthAlert{
			Type:      "memory",
			Severity:  "warning",
			Message:   fmt.Sprintf("High memory usage: %.2f MB", heapMB),
			Component: "system",
			Timestamp: time.Now(),
			Value:     heapMB,
			Threshold: ms.config.AlertThresholds.HighMemoryUsage,
		})
	}
	
	// Check database connections
	if health.Metrics.DatabaseMetrics.OpenConnections > ms.config.AlertThresholds.HighDatabaseConnections {
		alerts = append(alerts, HealthAlert{
			Type:      "database",
			Severity:  "warning",
			Message:   fmt.Sprintf("High database connections: %d", health.Metrics.DatabaseMetrics.OpenConnections),
			Component: "database",
			Timestamp: time.Now(),
			Value:     float64(health.Metrics.DatabaseMetrics.OpenConnections),
			Threshold: float64(ms.config.AlertThresholds.HighDatabaseConnections),
		})
	}
	
	// Check component health
	for name, component := range health.Components {
		if component.Status == "unhealthy" {
			alerts = append(alerts, HealthAlert{
				Type:      "component",
				Severity:  "critical",
				Message:   fmt.Sprintf("Component %s is unhealthy: %s", name, component.Message),
				Component: name,
				Timestamp: time.Now(),
			})
		}
	}
	
	return alerts
}

// UpdateConfig updates monitoring configuration
func (ms *MonitoringService) UpdateConfig(config MonitoringConfig) {
	ms.config = config
	ms.logger.Info("Monitoring configuration updated")
}

// GetMetricsSummary returns a summary of key metrics
func (ms *MonitoringService) GetMetricsSummary(ctx context.Context) map[string]interface{} {
	health, _ := ms.GetSystemHealth(ctx)
	
	return map[string]interface{}{
		"status":               health.Status,
		"uptime":              health.SystemInfo.Uptime.String(),
		"total_alerts":        health.Metrics.AlertMetrics.TotalAlerts,
		"firing_alerts":       health.Metrics.AlertMetrics.FiringAlerts,
		"memory_usage_mb":     float64(health.Metrics.MemoryUsage.HeapAlloc) / 1024 / 1024,
		"goroutines":          health.SystemInfo.NumGoroutines,
		"database_connections": health.Metrics.DatabaseMetrics.OpenConnections,
		"components_healthy":  len(health.Components),
		"alerts_count":        len(health.Alerts),
	}
}