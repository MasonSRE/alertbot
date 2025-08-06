package api

import (
	"fmt"
	"net/http"
	"strings"

	"alertbot/internal/models"
	"alertbot/internal/service"

	"github.com/gin-gonic/gin"
)

type NotificationChannelHandler struct {
	services *service.Services
	response *ResponseHelper
}

func NewNotificationChannelHandler(services *service.Services) *NotificationChannelHandler {
	return &NotificationChannelHandler{
		services: services,
		response: NewResponseHelper(),
	}
}

// ListChannels retrieves all notification channels
func (h *NotificationChannelHandler) ListChannels(c *gin.Context) {
	channels, err := h.services.NotificationChannel.ListChannels(c.Request.Context())
	if err != nil {
		h.response.InternalServerError(c, "Failed to retrieve notification channels", err.Error())
		return
	}

	h.response.Success(c, gin.H{
		"items": channels,
		"total": len(channels),
	}, "Notification channels retrieved successfully")
}

// CreateChannel creates a new notification channel
func (h *NotificationChannelHandler) CreateChannel(c *gin.Context) {
	var channel models.NotificationChannel
	if !h.response.BindAndValidate(c, &channel) {
		return
	}

	// Validate required fields
	if channel.Name == "" {
		h.response.ValidationError(c, "Channel name is required", nil)
		return
	}

	if channel.Type == "" {
		h.response.ValidationError(c, "Channel type is required", nil)
		return
	}

	// Validate channel type
	supportedTypes := []models.NotificationChannelType{
		models.ChannelTypeDingTalk,
		models.ChannelTypeWeChatWork,
		models.ChannelTypeEmail,
		models.ChannelTypeSMS,
	}

	validType := false
	for _, t := range supportedTypes {
		if models.NotificationChannelType(channel.Type) == t {
			validType = true
			break
		}
	}

	if !validType {
		h.response.ValidationError(c, "Unsupported channel type", gin.H{
			"supported_types": supportedTypes,
		})
		return
	}

	// Validate configuration based on type
	if err := h.validateChannelConfig(models.NotificationChannelType(channel.Type), channel.Config); err != nil {
		h.response.ValidationError(c, "Invalid channel configuration", err.Error())
		return
	}

	if err := h.services.NotificationChannel.CreateChannel(c.Request.Context(), &channel); err != nil {
		h.response.InternalServerError(c, "Failed to create notification channel", err.Error())
		return
	}

	h.response.SuccessWithStatus(c, http.StatusCreated, channel, "Notification channel created successfully")
}

// GetChannel retrieves a notification channel by ID
func (h *NotificationChannelHandler) GetChannel(c *gin.Context) {
	id, ok := h.response.ParseUintParam(c, "id")
	if !ok {
		return
	}

	channel, err := h.services.NotificationChannel.GetChannel(c.Request.Context(), id)
	if err != nil {
		h.response.NotFound(c, "Notification channel")
		return
	}

	// Mask sensitive information in config
	maskedChannel := h.maskSensitiveConfig(*channel)

	h.response.Success(c, maskedChannel, "Notification channel retrieved successfully")
}

// UpdateChannel updates a notification channel
func (h *NotificationChannelHandler) UpdateChannel(c *gin.Context) {
	id, ok := h.response.ParseUintParam(c, "id")
	if !ok {
		return
	}

	// Check if channel exists
	existingChannel, err := h.services.NotificationChannel.GetChannel(c.Request.Context(), id)
	if err != nil {
		h.response.NotFound(c, "Notification channel")
		return
	}

	var channel models.NotificationChannel
	if !h.response.BindAndValidate(c, &channel) {
		return
	}

	// Set ID and preserve created_at
	channel.ID = id
	channel.CreatedAt = existingChannel.CreatedAt

	// Validate required fields
	if channel.Name == "" {
		h.response.ValidationError(c, "Channel name is required", nil)
		return
	}

	if channel.Type == "" {
		h.response.ValidationError(c, "Channel type is required", nil)
		return
	}

	// Validate configuration based on type
	if err := h.validateChannelConfig(models.NotificationChannelType(channel.Type), channel.Config); err != nil {
		h.response.ValidationError(c, "Invalid channel configuration", err.Error())
		return
	}

	if err := h.services.NotificationChannel.UpdateChannel(c.Request.Context(), &channel); err != nil {
		h.response.InternalServerError(c, "Failed to update notification channel", err.Error())
		return
	}

	// Mask sensitive information in response
	maskedChannel := h.maskSensitiveConfig(channel)

	h.response.Success(c, maskedChannel, "Notification channel updated successfully")
}

// DeleteChannel deletes a notification channel
func (h *NotificationChannelHandler) DeleteChannel(c *gin.Context) {
	id, ok := h.response.ParseUintParam(c, "id")
	if !ok {
		return
	}

	// Check if channel exists
	_, err := h.services.NotificationChannel.GetChannel(c.Request.Context(), id)
	if err != nil {
		h.response.NotFound(c, "Notification channel")
		return
	}

	if err := h.services.NotificationChannel.DeleteChannel(c.Request.Context(), id); err != nil {
		h.response.InternalServerError(c, "Failed to delete notification channel", err.Error())
		return
	}

	h.response.Success(c, nil, "Notification channel deleted successfully")
}

// TestChannel tests a notification channel
func (h *NotificationChannelHandler) TestChannel(c *gin.Context) {
	id, ok := h.response.ParseUintParam(c, "id")
	if !ok {
		return
	}

	var req struct {
		Message string `json:"message" binding:"required"`
	}

	if !h.response.BindAndValidate(c, &req) {
		return
	}

	// Check if channel exists and is enabled
	channel, err := h.services.NotificationChannel.GetChannel(c.Request.Context(), id)
	if err != nil {
		h.response.NotFound(c, "Notification channel")
		return
	}

	if !channel.Enabled {
		h.response.BadRequest(c, "Notification channel is disabled", nil)
		return
	}

	if err := h.services.NotificationChannel.TestChannel(c.Request.Context(), id, req.Message); err != nil {
		h.response.InternalServerError(c, "Failed to test notification channel", err.Error())
		return
	}

	h.response.Success(c, gin.H{
		"sent":     true,
		"message":  "Test notification sent successfully",
		"channel":  channel.Name,
		"type":     channel.Type,
	}, "Test notification sent successfully")
}

// validateChannelConfig validates channel configuration based on type
func (h *NotificationChannelHandler) validateChannelConfig(channelType models.NotificationChannelType, config models.JSONB) error {
	if config == nil {
		return fmt.Errorf("configuration is required")
	}

	switch channelType {
	case models.ChannelTypeDingTalk:
		return h.validateDingTalkConfig(config)
	case models.ChannelTypeWeChatWork:
		return h.validateWeChatWorkConfig(config)
	case models.ChannelTypeEmail:
		return h.validateEmailConfig(config)
	case models.ChannelTypeSMS:
		return h.validateSMSConfig(config)
	default:
		return fmt.Errorf("unsupported channel type: %s", channelType)
	}
}

// validateDingTalkConfig validates DingTalk channel configuration
func (h *NotificationChannelHandler) validateDingTalkConfig(config models.JSONB) error {
	webhookURL, ok := config["webhook_url"].(string)
	if !ok || webhookURL == "" {
		return fmt.Errorf("webhook_url is required for DingTalk channels")
	}

	// Basic URL validation
	if !strings.HasPrefix(webhookURL, "https://oapi.dingtalk.com/robot/send") {
		return fmt.Errorf("invalid DingTalk webhook URL")
	}

	return nil
}

// validateWeChatWorkConfig validates WeChat Work channel configuration
func (h *NotificationChannelHandler) validateWeChatWorkConfig(config models.JSONB) error {
	webhookURL, ok := config["webhook_url"].(string)
	if !ok || webhookURL == "" {
		return fmt.Errorf("webhook_url is required for WeChat Work channels")
	}

	// Basic URL validation
	if !strings.HasPrefix(webhookURL, "https://qyapi.weixin.qq.com/cgi-bin/webhook/send") {
		return fmt.Errorf("invalid WeChat Work webhook URL")
	}

	return nil
}

// validateEmailConfig validates Email channel configuration
func (h *NotificationChannelHandler) validateEmailConfig(config models.JSONB) error {
	requiredFields := []string{"smtp_host", "username", "password", "from"}
	
	for _, field := range requiredFields {
		if value, ok := config[field].(string); !ok || value == "" {
			return fmt.Errorf("%s is required for email channels", field)
		}
	}

	// Validate that at least one recipient is provided
	if toInterface, ok := config["to"]; ok {
		switch to := toInterface.(type) {
		case []interface{}:
			if len(to) == 0 {
				return fmt.Errorf("at least one recipient is required")
			}
		case []string:
			if len(to) == 0 {
				return fmt.Errorf("at least one recipient is required")
			}
		case string:
			if to == "" {
				return fmt.Errorf("at least one recipient is required")
			}
		default:
			return fmt.Errorf("invalid 'to' field format")
		}
	} else {
		return fmt.Errorf("'to' field is required for email channels")
	}

	return nil
}

// validateSMSConfig validates SMS channel configuration
func (h *NotificationChannelHandler) validateSMSConfig(config models.JSONB) error {
	// Check for phone numbers
	if phoneNumbersInterface, ok := config["phone_numbers"]; ok {
		switch phoneNumbers := phoneNumbersInterface.(type) {
		case []interface{}:
			if len(phoneNumbers) == 0 {
				return fmt.Errorf("at least one phone number is required")
			}
		case []string:
			if len(phoneNumbers) == 0 {
				return fmt.Errorf("at least one phone number is required")
			}
		case string:
			if phoneNumbers == "" {
				return fmt.Errorf("at least one phone number is required")
			}
		default:
			return fmt.Errorf("invalid 'phone_numbers' field format")
		}
	} else {
		return fmt.Errorf("'phone_numbers' field is required for SMS channels")
	}

	return nil
}

// maskSensitiveConfig masks sensitive information in channel config
func (h *NotificationChannelHandler) maskSensitiveConfig(channel models.NotificationChannel) models.NotificationChannel {
	if channel.Config == nil {
		return channel
	}

	// Create a copy of the config
	maskedConfig := make(models.JSONB)
	for k, v := range channel.Config {
		maskedConfig[k] = v
	}

	// Mask sensitive fields
	sensitiveFields := []string{"password", "secret", "auth_token", "access_key", "api_key"}
	
	for _, field := range sensitiveFields {
		if _, exists := maskedConfig[field]; exists {
			maskedConfig[field] = "***MASKED***"
		}
	}

	channel.Config = maskedConfig
	return channel
}