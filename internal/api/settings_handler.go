package api

import (
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
)

type SettingsHandler struct {
	response *ResponseHelper
}

func NewSettingsHandler() *SettingsHandler {
	return &SettingsHandler{
		response: NewResponseHelper(),
	}
}

type SystemSettings struct {
	SystemName          string `json:"system_name" binding:"required"`
	AdminEmail          string `json:"admin_email" binding:"required,email"`
	RetentionDays       int    `json:"retention_days" binding:"required,min=1,max=365"`
	EnableNotifications bool   `json:"enable_notifications"`
	EnableWebhooks      bool   `json:"enable_webhooks"`
	WebhookTimeout      int    `json:"webhook_timeout" binding:"min=1,max=300"`
}

type PrometheusSettings struct {
	Enabled            bool   `json:"enabled"`
	URL                string `json:"url" binding:"required,url"`
	Timeout            int    `json:"timeout" binding:"required,min=5,max=300"`
	QueryTimeout       int    `json:"query_timeout" binding:"required,min=5,max=300"`
	ScrapeInterval     string `json:"scrape_interval"`
	EvaluationInterval string `json:"evaluation_interval"`
}

type NotificationSettings struct {
	MaxRetries    int `json:"max_retries" binding:"min=0,max=10"`
	RetryInterval int `json:"retry_interval" binding:"min=1,max=3600"`
	RateLimit     int `json:"rate_limit" binding:"min=1,max=1000"`
	BatchSize     int `json:"batch_size" binding:"min=1,max=100"`
}

// GetSystemSettings retrieves system settings
func (h *SettingsHandler) GetSystemSettings(c *gin.Context) {
	// In a real implementation, these would be loaded from database or config
	settings := SystemSettings{
		SystemName:          "AlertBot",
		AdminEmail:          "admin@company.com",
		RetentionDays:       30,
		EnableNotifications: true,
		EnableWebhooks:      true,
		WebhookTimeout:      30,
	}

	h.response.Success(c, settings, "System settings retrieved successfully")
}

// UpdateSystemSettings updates system settings
func (h *SettingsHandler) UpdateSystemSettings(c *gin.Context) {
	var settings SystemSettings
	if err := c.ShouldBindJSON(&settings); err != nil {
		h.response.BadRequest(c, "Invalid settings data", err.Error())
		return
	}

	// In a real implementation, save to database or config file
	// For now, just return success
	
	h.response.Success(c, settings, "System settings updated successfully")
}

// GetPrometheusSettings retrieves Prometheus settings
func (h *SettingsHandler) GetPrometheusSettings(c *gin.Context) {
	// In a real implementation, these would be loaded from database or config
	settings := PrometheusSettings{
		Enabled:            true,
		URL:                "http://localhost:9090",
		Timeout:            30,
		QueryTimeout:       30,
		ScrapeInterval:     "15s",
		EvaluationInterval: "15s",
	}

	h.response.Success(c, settings, "Prometheus settings retrieved successfully")
}

// UpdatePrometheusSettings updates Prometheus settings
func (h *SettingsHandler) UpdatePrometheusSettings(c *gin.Context) {
	var settings PrometheusSettings
	if err := c.ShouldBindJSON(&settings); err != nil {
		h.response.BadRequest(c, "Invalid Prometheus settings", err.Error())
		return
	}

	// Validate URL format
	if settings.URL != "" {
		if _, err := url.Parse(settings.URL); err != nil {
			h.response.BadRequest(c, "Invalid Prometheus URL format", err.Error())
			return
		}
	}

	// Validate interval formats
	if settings.ScrapeInterval != "" {
		if _, err := time.ParseDuration(settings.ScrapeInterval); err != nil {
			h.response.BadRequest(c, "Invalid scrape_interval format", err.Error())
			return
		}
	}

	if settings.EvaluationInterval != "" {
		if _, err := time.ParseDuration(settings.EvaluationInterval); err != nil {
			h.response.BadRequest(c, "Invalid evaluation_interval format", err.Error())
			return
		}
	}

	// In a real implementation, save to database or config file
	// Also should restart Prometheus connection if URL changed
	
	h.response.Success(c, settings, "Prometheus settings updated successfully")
}

// TestPrometheusConnection tests connection to Prometheus server
func (h *SettingsHandler) TestPrometheusConnection(c *gin.Context) {
	var testRequest struct {
		URL     string `json:"url" binding:"required,url"`
		Timeout int    `json:"timeout" binding:"required,min=5,max=300"`
	}

	if err := c.ShouldBindJSON(&testRequest); err != nil {
		h.response.BadRequest(c, "Invalid test request", err.Error())
		return
	}

	// Test connection to Prometheus
	client := &http.Client{
		Timeout: time.Duration(testRequest.Timeout) * time.Second,
	}

	// Try to access Prometheus API - using a simple query endpoint
	resp, err := client.Get(testRequest.URL + "/api/v1/query?query=up")
	if err != nil {
		h.response.ServiceUnavailable(c, "Failed to connect to Prometheus: "+err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		h.response.ServiceUnavailable(c, "Prometheus returned non-200 status code: "+resp.Status)
		return
	}

	result := gin.H{
		"connected": true,
		"status":    "healthy",
		"url":       testRequest.URL,
		"response_time": "calculated_response_time", // In real implementation, measure this
	}

	h.response.Success(c, result, "Prometheus connection test successful")
}

// GetNotificationSettings retrieves notification settings
func (h *SettingsHandler) GetNotificationSettings(c *gin.Context) {
	// In a real implementation, these would be loaded from database or config
	settings := NotificationSettings{
		MaxRetries:    3,
		RetryInterval: 30,
		RateLimit:     100,
		BatchSize:     10,
	}

	h.response.Success(c, settings, "Notification settings retrieved successfully")
}

// UpdateNotificationSettings updates notification settings
func (h *SettingsHandler) UpdateNotificationSettings(c *gin.Context) {
	var settings NotificationSettings
	if err := c.ShouldBindJSON(&settings); err != nil {
		h.response.BadRequest(c, "Invalid notification settings", err.Error())
		return
	}

	// In a real implementation, save to database or config file
	// Also should update notification service configuration
	
	h.response.Success(c, settings, "Notification settings updated successfully")
}