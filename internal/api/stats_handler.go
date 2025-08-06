package api

import (
	"alertbot/internal/service"

	"github.com/gin-gonic/gin"
)

type StatsHandler struct {
	services *service.Services
	response *ResponseHelper
}

func NewStatsHandler(services *service.Services) *StatsHandler {
	return &StatsHandler{
		services: services,
		response: NewResponseHelper(),
	}
}

// GetAlertStats retrieves alert statistics
func (h *StatsHandler) GetAlertStats(c *gin.Context) {
	// Get query parameters
	startTime := c.Query("start_time")
	endTime := c.Query("end_time")
	groupBy := c.Query("group_by")

	// Default groupBy if not provided
	if groupBy == "" {
		groupBy = "severity"
	}

	// Validate groupBy parameter
	validGroupBy := map[string]bool{
		"severity":  true,
		"status":    true,
		"alertname": true,
		"instance":  true,
	}

	if !validGroupBy[groupBy] {
		h.response.BadRequest(c, "Invalid group_by parameter", gin.H{
			"valid_values": []string{"severity", "status", "alertname", "instance"},
		})
		return
	}

	stats, err := h.services.Stats.GetAlertStats(c.Request.Context(), startTime, endTime, groupBy)
	if err != nil {
		h.response.InternalServerError(c, "Failed to retrieve alert statistics", err.Error())
		return
	}

	h.response.Success(c, stats, "Alert statistics retrieved successfully")
}

// GetNotificationStats retrieves notification statistics
func (h *StatsHandler) GetNotificationStats(c *gin.Context) {
	// Get query parameters
	startTime := c.Query("start_time")
	endTime := c.Query("end_time")

	stats, err := h.services.Stats.GetNotificationStats(c.Request.Context(), startTime, endTime)
	if err != nil {
		h.response.InternalServerError(c, "Failed to retrieve notification statistics", err.Error())
		return
	}

	h.response.Success(c, stats, "Notification statistics retrieved successfully")
}

// GetSystemStats retrieves general system statistics
func (h *StatsHandler) GetSystemStats(c *gin.Context) {
	// Get basic system information
	systemStats := gin.H{
		"version":     "1.0.0",
		"uptime":      "calculated_uptime", // In a real implementation, track server start time
		"go_version":  "go1.21",
		"alerts_api":  "/api/v1/alerts",
		"rules_api":   "/api/v1/rules",
		"channels_api": "/api/v1/channels",
		"silences_api": "/api/v1/silences",
	}

	// Get counts of various entities
	// Note: In a production system, you might want to cache these counts
	
	// Count alerts
	_, totalAlerts, err := h.services.Alert.ListAlerts(c.Request.Context(), struct {
		Status    string `json:"status" form:"status"`
		Severity  string `json:"severity" form:"severity"`
		AlertName string `json:"alertname" form:"alertname"`
		Instance  string `json:"instance" form:"instance"`
		Page      int    `json:"page" form:"page"`
		Size      int    `json:"size" form:"size"`
		Sort      string `json:"sort" form:"sort"`
		Order     string `json:"order" form:"order"`
	}{Size: 1})
	
	if err != nil {
		h.response.InternalServerError(c, "Failed to get alert count", err.Error())
		return
	}

	// Count rules
	rules, err := h.services.RoutingRule.ListRules(c.Request.Context())
	if err != nil {
		h.response.InternalServerError(c, "Failed to get rules count", err.Error())
		return
	}

	// Count channels
	channels, err := h.services.NotificationChannel.ListChannels(c.Request.Context())
	if err != nil {
		h.response.InternalServerError(c, "Failed to get channels count", err.Error())
		return
	}

	// Count silences
	silences, err := h.services.Silence.ListSilences(c.Request.Context())
	if err != nil {
		h.response.InternalServerError(c, "Failed to get silences count", err.Error())
		return
	}

	// Combine system stats with counts
	systemStats["entities"] = gin.H{
		"total_alerts":    totalAlerts,
		"total_rules":     len(rules),
		"total_channels":  len(channels),
		"total_silences":  len(silences),
	}

	h.response.Success(c, systemStats, "System statistics retrieved successfully")
}

// GetHealthStatus retrieves system health status
func (h *StatsHandler) GetHealthStatus(c *gin.Context) {
	// Perform basic health checks
	healthStatus := gin.H{
		"status":    "healthy",
		"timestamp": gin.H{},
		"checks":    gin.H{},
	}

	checks := gin.H{}

	// Check database connectivity
	_, _, err := h.services.Alert.ListAlerts(c.Request.Context(), struct {
		Status    string `json:"status" form:"status"`
		Severity  string `json:"severity" form:"severity"`
		AlertName string `json:"alertname" form:"alertname"`
		Instance  string `json:"instance" form:"instance"`
		Page      int    `json:"page" form:"page"`
		Size      int    `json:"size" form:"size"`
		Sort      string `json:"sort" form:"sort"`
		Order     string `json:"order" form:"order"`
	}{Size: 1})

	if err != nil {
		checks["database"] = gin.H{
			"status": "unhealthy",
			"error":  err.Error(),
		}
		healthStatus["status"] = "unhealthy"
	} else {
		checks["database"] = gin.H{
			"status": "healthy",
		}
	}

	// Check notification channels (basic connectivity test)
	channels, err := h.services.NotificationChannel.ListChannels(c.Request.Context())
	if err != nil {
		checks["notification_channels"] = gin.H{
			"status": "unhealthy",
			"error":  err.Error(),
		}
		healthStatus["status"] = "degraded"
	} else {
		enabledChannels := 0
		for _, channel := range channels {
			if channel.Enabled {
				enabledChannels++
			}
		}
		checks["notification_channels"] = gin.H{
			"status":           "healthy",
			"total_channels":   len(channels),
			"enabled_channels": enabledChannels,
		}
	}

	healthStatus["checks"] = checks

	// Determine HTTP status based on health
	if healthStatus["status"] == "healthy" {
		h.response.Success(c, healthStatus, "System is healthy")
	} else {
		h.response.ServiceUnavailable(c, "System health check failed")
	}
}