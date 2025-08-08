package api

import (
	"strconv"
	"time"

	"alertbot/internal/monitoring"

	"github.com/gin-gonic/gin"
)

type MonitoringHandler struct {
	monitoringService *monitoring.MonitoringService
	backgroundMonitor *monitoring.BackgroundMonitor
	response         *ResponseHelper
}

func NewMonitoringHandler(monitoringService *monitoring.MonitoringService, backgroundMonitor *monitoring.BackgroundMonitor) *MonitoringHandler {
	return &MonitoringHandler{
		monitoringService: monitoringService,
		backgroundMonitor: backgroundMonitor,
		response:         NewResponseHelper(),
	}
}

// GetSystemHealth returns comprehensive system health information
func (h *MonitoringHandler) GetSystemHealth(c *gin.Context) {
	health, err := h.monitoringService.GetSystemHealth(c.Request.Context())
	if err != nil {
		h.response.InternalServerError(c, "Failed to get system health", err.Error())
		return
	}

	// Set HTTP status based on health
	if health.Status == "unhealthy" {
		c.JSON(503, gin.H{
			"success": false,
			"data":    health,
			"message": "System is unhealthy",
		})
		return
	}

	if health.Status == "degraded" {
		c.JSON(200, gin.H{
			"success": true,
			"data":    health,
			"message": "System is in degraded state",
			"warning": true,
		})
		return
	}

	h.response.Success(c, health, "System is healthy")
}

// GetMetricsSummary returns a summary of key metrics
func (h *MonitoringHandler) GetMetricsSummary(c *gin.Context) {
	summary := h.monitoringService.GetMetricsSummary(c.Request.Context())
	h.response.Success(c, summary, "Metrics summary retrieved successfully")
}

// GetPerformanceStats returns detailed performance statistics
func (h *MonitoringHandler) GetPerformanceStats(c *gin.Context) {
	if h.backgroundMonitor == nil {
		h.response.ServiceUnavailable(c, "Background monitor not available")
		return
	}

	stats := h.backgroundMonitor.GetStats()
	h.response.Success(c, stats, "Performance statistics retrieved successfully")
}

// GetHealthCheck returns a simple health check (for load balancers)
func (h *MonitoringHandler) GetHealthCheck(c *gin.Context) {
	// Simple health check without detailed information
	health, err := h.monitoringService.GetSystemHealth(c.Request.Context())
	if err != nil {
		c.JSON(503, gin.H{
			"status": "unhealthy",
			"error":  err.Error(),
		})
		return
	}

	if health.Status == "healthy" {
		c.JSON(200, gin.H{
			"status":    "healthy",
			"timestamp": time.Now(),
		})
	} else {
		c.JSON(503, gin.H{
			"status":    health.Status,
			"timestamp": time.Now(),
		})
	}
}

// GetReadinessCheck returns readiness status (for Kubernetes)
func (h *MonitoringHandler) GetReadinessCheck(c *gin.Context) {
	health, err := h.monitoringService.GetSystemHealth(c.Request.Context())
	if err != nil {
		c.JSON(503, gin.H{
			"ready": false,
			"error": err.Error(),
		})
		return
	}

	// Check critical components
	ready := true
	criticalComponents := []string{"database", "alert_service"}
	
	for _, component := range criticalComponents {
		if status, exists := health.Components[component]; exists {
			if status.Status == "unhealthy" {
				ready = false
				break
			}
		}
	}

	if ready {
		c.JSON(200, gin.H{
			"ready":     true,
			"timestamp": time.Now(),
		})
	} else {
		c.JSON(503, gin.H{
			"ready":     false,
			"timestamp": time.Now(),
		})
	}
}

// GetLivenessCheck returns liveness status (for Kubernetes)
func (h *MonitoringHandler) GetLivenessCheck(c *gin.Context) {
	// Simple liveness check - if we can respond, we're alive
	c.JSON(200, gin.H{
		"alive":     true,
		"timestamp": time.Now(),
	})
}

// GetComponentHealth returns health status for a specific component
func (h *MonitoringHandler) GetComponentHealth(c *gin.Context) {
	componentName := c.Param("component")
	if componentName == "" {
		h.response.BadRequest(c, "Component name is required", nil)
		return
	}

	health, err := h.monitoringService.GetSystemHealth(c.Request.Context())
	if err != nil {
		h.response.InternalServerError(c, "Failed to get system health", err.Error())
		return
	}

	if component, exists := health.Components[componentName]; exists {
		h.response.Success(c, component, "Component health retrieved successfully")
	} else {
		h.response.NotFound(c, "Component")
	}
}

// UpdateMonitoringConfig updates monitoring configuration
func (h *MonitoringHandler) UpdateMonitoringConfig(c *gin.Context) {
	var config monitoring.MonitoringConfig
	if !h.response.BindAndValidate(c, &config) {
		return
	}

	// Validate configuration
	if config.HealthCheckInterval <= 0 {
		h.response.BadRequest(c, "Health check interval must be greater than 0", nil)
		return
	}

	if config.MetricsCollectionInterval <= 0 {
		h.response.BadRequest(c, "Metrics collection interval must be greater than 0", nil)
		return
	}

	h.monitoringService.UpdateConfig(config)
	h.response.Success(c, config, "Monitoring configuration updated successfully")
}

// GetMonitoringConfig returns current monitoring configuration
func (h *MonitoringHandler) GetMonitoringConfig(c *gin.Context) {
	// Return default configuration - in a real implementation,
	// this would be stored in the database or configuration file
	defaultConfig := monitoring.MonitoringConfig{
		HealthCheckInterval:       30 * time.Second,
		MetricsCollectionInterval: 15 * time.Second,
		AlertThresholds: monitoring.AlertThresholds{
			HighMemoryUsage:         500,
			HighCPUUsage:           80,
			HighDatabaseConnections: 90,
			HighResponseTime:       5 * time.Second,
			LowDiskSpace:           10,
		},
		EnableSystemMetrics:   true,
		EnableDatabaseMetrics: true,
		EnableServiceMetrics:  true,
	}

	h.response.Success(c, defaultConfig, "Monitoring configuration retrieved successfully")
}

// GetSystemInfo returns basic system information
func (h *MonitoringHandler) GetSystemInfo(c *gin.Context) {
	health, err := h.monitoringService.GetSystemHealth(c.Request.Context())
	if err != nil {
		h.response.InternalServerError(c, "Failed to get system health", err.Error())
		return
	}

	h.response.Success(c, health.SystemInfo, "System information retrieved successfully")
}

// GetSystemMetrics returns detailed system metrics
func (h *MonitoringHandler) GetSystemMetrics(c *gin.Context) {
	health, err := h.monitoringService.GetSystemHealth(c.Request.Context())
	if err != nil {
		h.response.InternalServerError(c, "Failed to get system health", err.Error())
		return
	}

	h.response.Success(c, health.Metrics, "System metrics retrieved successfully")
}

// GetHealthHistory returns health check history (if implemented)
func (h *MonitoringHandler) GetHealthHistory(c *gin.Context) {
	component := c.Query("component")
	limitStr := c.DefaultQuery("limit", "100")
	
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 1000 {
		limit = 100
	}

	// This would be implemented to return historical health data
	// For now, return a placeholder response
	historyData := map[string]interface{}{
		"component":  component,
		"limit":      limit,
		"message":    "Health history feature not yet implemented",
		"data":       []interface{}{},
	}

	h.response.Success(c, historyData, "Health history retrieved successfully")
}

// TriggerHealthCheck manually triggers a health check
func (h *MonitoringHandler) TriggerHealthCheck(c *gin.Context) {
	health, err := h.monitoringService.GetSystemHealth(c.Request.Context())
	if err != nil {
		h.response.InternalServerError(c, "Failed to trigger health check", err.Error())
		return
	}

	h.response.Success(c, gin.H{
		"triggered_at": time.Now(),
		"health":      health,
	}, "Health check triggered successfully")
}

// GetAlerts returns current monitoring alerts
func (h *MonitoringHandler) GetAlerts(c *gin.Context) {
	health, err := h.monitoringService.GetSystemHealth(c.Request.Context())
	if err != nil {
		h.response.InternalServerError(c, "Failed to get monitoring alerts", err.Error())
		return
	}

	h.response.Success(c, health.Alerts, "Monitoring alerts retrieved successfully")
}

// ExportMetrics exports metrics in Prometheus format (handled by /metrics endpoint)
func (h *MonitoringHandler) ExportMetrics(c *gin.Context) {
	// This is typically handled by the prometheus handler at /metrics
	// But we can provide a JSON version of key metrics here
	summary := h.monitoringService.GetMetricsSummary(c.Request.Context())
	
	h.response.Success(c, gin.H{
		"format":   "json",
		"exported_at": time.Now(),
		"metrics": summary,
	}, "Metrics exported successfully")
}