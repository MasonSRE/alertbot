package api

import (
	"net/http"
	"time"

	"alertbot/internal/models"
	"alertbot/internal/service"

	"github.com/gin-gonic/gin"
)

type SettingsHandler struct {
	response *ResponseHelper
	settings service.SettingsService
}

func NewSettingsHandler(settings service.SettingsService) *SettingsHandler {
	return &SettingsHandler{
		response: NewResponseHelper(),
		settings: settings,
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
	config, err := h.settings.GetSystemConfig()
	if err != nil {
		h.response.InternalServerError(c, "Failed to get system settings", err.Error())
		return
	}

	// Convert to response format
	settings := SystemSettings{
		SystemName:          config.SystemName,
		AdminEmail:          config.AdminEmail,
		RetentionDays:       config.RetentionDays,
		EnableNotifications: config.EnableNotifications,
		EnableWebhooks:      config.EnableWebhooks,
		WebhookTimeout:      config.WebhookTimeout,
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

	// Convert to model
	config := &models.SystemConfig{
		SystemName:          settings.SystemName,
		AdminEmail:          settings.AdminEmail,
		RetentionDays:       settings.RetentionDays,
		EnableNotifications: settings.EnableNotifications,
		EnableWebhooks:      settings.EnableWebhooks,
		WebhookTimeout:      settings.WebhookTimeout,
	}

	if err := h.settings.UpdateSystemConfig(config); err != nil {
		h.response.BadRequest(c, "Failed to update system settings", err.Error())
		return
	}
	
	h.response.Success(c, settings, "System settings updated successfully")
}

// GetPrometheusSettings retrieves Prometheus settings
func (h *SettingsHandler) GetPrometheusSettings(c *gin.Context) {
	config, err := h.settings.GetPrometheusConfig()
	if err != nil {
		h.response.InternalServerError(c, "Failed to get Prometheus settings", err.Error())
		return
	}

	// Convert to response format
	settings := PrometheusSettings{
		Enabled:            config.Enabled,
		URL:                config.URL,
		Timeout:            config.Timeout,
		QueryTimeout:       config.QueryTimeout,
		ScrapeInterval:     config.ScrapeInterval,
		EvaluationInterval: config.EvaluationInterval,
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

	// Convert to model
	config := &models.PrometheusConfig{
		Enabled:            settings.Enabled,
		URL:                settings.URL,
		Timeout:            settings.Timeout,
		QueryTimeout:       settings.QueryTimeout,
		ScrapeInterval:     settings.ScrapeInterval,
		EvaluationInterval: settings.EvaluationInterval,
	}

	if err := h.settings.UpdatePrometheusConfig(config); err != nil {
		h.response.BadRequest(c, "Failed to update Prometheus settings", err.Error())
		return
	}
	
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
	config, err := h.settings.GetNotificationConfig()
	if err != nil {
		h.response.InternalServerError(c, "Failed to get notification settings", err.Error())
		return
	}

	// Convert to response format
	settings := NotificationSettings{
		MaxRetries:    config.MaxRetries,
		RetryInterval: config.RetryInterval,
		RateLimit:     config.RateLimit,
		BatchSize:     config.BatchSize,
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

	// Convert to model
	config := &models.NotificationConfig{
		MaxRetries:    settings.MaxRetries,
		RetryInterval: settings.RetryInterval,
		RateLimit:     settings.RateLimit,
		BatchSize:     settings.BatchSize,
	}

	if err := h.settings.UpdateNotificationConfig(config); err != nil {
		h.response.BadRequest(c, "Failed to update notification settings", err.Error())
		return
	}
	
	h.response.Success(c, settings, "Notification settings updated successfully")
}