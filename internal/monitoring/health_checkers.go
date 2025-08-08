package monitoring

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"alertbot/internal/models"
	"alertbot/internal/repository"

	"gorm.io/gorm"
)

// DatabaseHealthChecker checks database health
type DatabaseHealthChecker struct {
	db *gorm.DB
}

func NewDatabaseHealthChecker(db *gorm.DB) *DatabaseHealthChecker {
	return &DatabaseHealthChecker{db: db}
}

func (d *DatabaseHealthChecker) Name() string {
	return "database"
}

func (d *DatabaseHealthChecker) CheckHealth(ctx context.Context) HealthStatus {
	start := time.Now()
	status := HealthStatus{
		Name:      d.Name(),
		Timestamp: start,
		Details:   make(map[string]interface{}),
	}
	
	defer func() {
		status.Duration = time.Since(start)
	}()

	if d.db == nil {
		status.Status = "unhealthy"
		status.Message = "Database connection is nil"
		return status
	}

	// Test basic connectivity
	sqlDB, err := d.db.DB()
	if err != nil {
		status.Status = "unhealthy"
		status.Message = "Failed to get underlying SQL DB: " + err.Error()
		return status
	}

	// Ping database
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	
	if err := sqlDB.PingContext(pingCtx); err != nil {
		status.Status = "unhealthy"
		status.Message = "Database ping failed: " + err.Error()
		return status
	}

	// Check database stats
	stats := sqlDB.Stats()
	status.Details["open_connections"] = stats.OpenConnections
	status.Details["in_use"] = stats.InUse
	status.Details["idle"] = stats.Idle
	status.Details["wait_count"] = stats.WaitCount
	status.Details["wait_duration"] = stats.WaitDuration.String()

	// Check if we can perform a simple query
	var count int64
	if err := d.db.WithContext(ctx).Raw("SELECT COUNT(*) FROM information_schema.tables").Scan(&count).Error; err != nil {
		status.Status = "degraded"
		status.Message = "Database query failed: " + err.Error()
		return status
	}

	status.Details["tables_count"] = count

	// Check connection usage
	maxConn := stats.MaxOpenConnections
	if maxConn > 0 {
		usagePercent := float64(stats.OpenConnections) / float64(maxConn) * 100
		status.Details["connection_usage_percent"] = usagePercent
		
		if usagePercent > 90 {
			status.Status = "degraded"
			status.Message = "High database connection usage"
			return status
		}
	}

	status.Status = "healthy"
	status.Message = "Database is accessible and responsive"
	return status
}

// SystemHealthChecker checks system-level health
type SystemHealthChecker struct{}

func NewSystemHealthChecker() *SystemHealthChecker {
	return &SystemHealthChecker{}
}

func (s *SystemHealthChecker) Name() string {
	return "system"
}

func (s *SystemHealthChecker) CheckHealth(ctx context.Context) HealthStatus {
	start := time.Now()
	status := HealthStatus{
		Name:      s.Name(),
		Timestamp: start,
		Details:   make(map[string]interface{}),
	}
	
	defer func() {
		status.Duration = time.Since(start)
	}()

	// Get memory stats
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	heapMB := float64(m.HeapAlloc) / 1024 / 1024
	sysMB := float64(m.Sys) / 1024 / 1024
	
	status.Details["heap_alloc_mb"] = heapMB
	status.Details["sys_mb"] = sysMB
	status.Details["num_goroutines"] = runtime.NumGoroutine()
	status.Details["num_cpu"] = runtime.NumCPU()
	status.Details["gc_runs"] = m.NumGC
	status.Details["gc_pause_total_ms"] = float64(m.PauseTotalNs) / 1000000

	// Check memory usage thresholds
	if heapMB > 1000 { // 1GB threshold
		status.Status = "degraded"
		status.Message = "High memory usage"
		return status
	}

	// Check goroutine count
	numGoroutines := runtime.NumGoroutine()
	if numGoroutines > 10000 {
		status.Status = "degraded"
		status.Message = "High goroutine count"
		return status
	}

	status.Status = "healthy"
	status.Message = "System resources are within normal limits"
	return status
}

// AlertServiceHealthChecker checks alert service health
type AlertServiceHealthChecker struct {
	repositories *repository.Repositories
}

func NewAlertServiceHealthChecker(repos *repository.Repositories) *AlertServiceHealthChecker {
	return &AlertServiceHealthChecker{repositories: repos}
}

func (a *AlertServiceHealthChecker) Name() string {
	return "alert_service"
}

func (a *AlertServiceHealthChecker) CheckHealth(ctx context.Context) HealthStatus {
	start := time.Now()
	status := HealthStatus{
		Name:      a.Name(),
		Timestamp: start,
		Details:   make(map[string]interface{}),
	}
	
	defer func() {
		status.Duration = time.Since(start)
	}()

	if a.repositories == nil || a.repositories.Alert == nil {
		status.Status = "unhealthy"
		status.Message = "Alert repository is not available"
		return status
	}

	// Test alert repository functionality
	_, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Try to list alerts (limit to 1 for performance)
	filters := models.AlertFilters{Size: 1}
	alerts, total, err := a.repositories.Alert.List(filters)
	if err != nil {
		status.Status = "unhealthy"
		status.Message = "Failed to query alerts: " + err.Error()
		return status
	}

	status.Details["total_alerts"] = total
	status.Details["query_returned_count"] = len(alerts)

	// Get recent alert activity
	recentFilters := models.AlertFilters{Size: 10}
	recentAlerts, _, err := a.repositories.Alert.List(recentFilters)
	if err == nil {
		firingCount := 0
		for _, alert := range recentAlerts {
			if alert.Status == "firing" {
				firingCount++
			}
		}
		status.Details["recent_firing_alerts"] = firingCount
	}

	// Check alert history functionality
	if a.repositories.AlertHistory != nil {
		historyFilters := models.AlertHistoryFilters{Size: 1}
		_, historyTotal, err := a.repositories.AlertHistory.List(historyFilters)
		if err == nil {
			status.Details["total_history_entries"] = historyTotal
		}
	}

	// Performance check - measure query time
	queryStart := time.Now()
	_, _, err = a.repositories.Alert.List(models.AlertFilters{Size: 1})
	queryDuration := time.Since(queryStart)
	
	status.Details["query_duration_ms"] = queryDuration.Milliseconds()

	if queryDuration > 1*time.Second {
		status.Status = "degraded"
		status.Message = "Alert queries are slow"
		return status
	}

	if err != nil {
		status.Status = "unhealthy"
		status.Message = "Alert service query failed: " + err.Error()
		return status
	}

	status.Status = "healthy"
	status.Message = "Alert service is functioning normally"
	return status
}

// NotificationHealthChecker checks notification system health
type NotificationHealthChecker struct {
	repositories *repository.Repositories
}

func NewNotificationHealthChecker(repos *repository.Repositories) *NotificationHealthChecker {
	return &NotificationHealthChecker{repositories: repos}
}

func (n *NotificationHealthChecker) Name() string {
	return "notification_service"
}

func (n *NotificationHealthChecker) CheckHealth(ctx context.Context) HealthStatus {
	start := time.Now()
	status := HealthStatus{
		Name:      n.Name(),
		Timestamp: start,
		Details:   make(map[string]interface{}),
	}
	
	defer func() {
		status.Duration = time.Since(start)
	}()

	if n.repositories == nil || n.repositories.NotificationChannel == nil {
		status.Status = "unhealthy"
		status.Message = "Notification channel repository is not available"
		return status
	}

	// Test notification channel repository
	_, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	channels, err := n.repositories.NotificationChannel.List()
	if err != nil {
		status.Status = "unhealthy"
		status.Message = "Failed to query notification channels: " + err.Error()
		return status
	}

	enabledChannels := 0
	channelTypes := make(map[string]int)
	
	for _, channel := range channels {
		if channel.Enabled {
			enabledChannels++
		}
		channelTypes[channel.Type]++
	}

	status.Details["total_channels"] = len(channels)
	status.Details["enabled_channels"] = enabledChannels
	status.Details["channel_types"] = channelTypes

	if len(channels) == 0 {
		status.Status = "degraded"
		status.Message = "No notification channels configured"
		return status
	}

	if enabledChannels == 0 {
		status.Status = "degraded"
		status.Message = "No notification channels enabled"
		return status
	}

	// Check routing rules
	if n.repositories.RoutingRule != nil {
		rules, err := n.repositories.RoutingRule.List()
		if err == nil {
			enabledRules := 0
			for _, rule := range rules {
				if rule.Enabled {
					enabledRules++
				}
			}
			status.Details["total_rules"] = len(rules)
			status.Details["enabled_rules"] = enabledRules
			
			if enabledRules == 0 {
				status.Status = "degraded"
				status.Message = "No routing rules enabled"
				return status
			}
		}
	}

	status.Status = "healthy"
	status.Message = "Notification service is properly configured"
	return status
}

// WebSocketHealthChecker checks WebSocket functionality
type WebSocketHealthChecker struct {
	connectionCount *int
}

func NewWebSocketHealthChecker(connectionCount *int) *WebSocketHealthChecker {
	return &WebSocketHealthChecker{connectionCount: connectionCount}
}

func (w *WebSocketHealthChecker) Name() string {
	return "websocket"
}

func (w *WebSocketHealthChecker) CheckHealth(ctx context.Context) HealthStatus {
	start := time.Now()
	status := HealthStatus{
		Name:      w.Name(),
		Timestamp: start,
		Details:   make(map[string]interface{}),
	}
	
	defer func() {
		status.Duration = time.Since(start)
	}()

	connections := 0
	if w.connectionCount != nil {
		connections = *w.connectionCount
	}

	status.Details["active_connections"] = connections

	// WebSocket service is considered healthy if it's accepting connections
	// Even 0 connections is fine - it just means no clients are connected
	status.Status = "healthy"
	status.Message = fmt.Sprintf("WebSocket service available, %d active connections", connections)
	
	return status
}