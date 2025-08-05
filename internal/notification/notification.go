package notification

import (
	"context"
	"fmt"
	"time"

	"alertbot/internal/errors"
	"alertbot/internal/metrics"
	"alertbot/internal/models"
	"alertbot/internal/recovery"

	"github.com/sirupsen/logrus"
)

// NotificationManager manages all notification channels
type NotificationManager struct {
	channels       map[models.NotificationChannelType]NotificationChannel
	logger         *logrus.Logger
	circuitBreaker *recovery.CircuitBreaker
	retryConfig    recovery.RetryConfig
}

// NotificationChannel interface for all notification channels
type NotificationChannel interface {
	Send(ctx context.Context, message *NotificationMessage) error
	Test(ctx context.Context, testMessage string) error
	GetType() models.NotificationChannelType
}

// NotificationMessage represents a notification to be sent
type NotificationMessage struct {
	Title       string                 `json:"title"`
	Content     string                 `json:"content"`
	Level       string                 `json:"level"` // info, warning, error, critical
	Alert       *models.Alert          `json:"alert,omitempty"`
	ChannelConfig map[string]interface{} `json:"channel_config"`
	Template    string                 `json:"template,omitempty"`
}

func NewNotificationManager(logger *logrus.Logger) *NotificationManager {
	nm := &NotificationManager{
		channels: make(map[models.NotificationChannelType]NotificationChannel),
		logger:   logger,
		circuitBreaker: recovery.NewCircuitBreaker(recovery.CircuitBreakerConfig{
			Name:         "notification_manager",
			MaxFailures:  5,
			ResetTimeout: 60 * time.Second,
			Logger:       logger,
		}),
		retryConfig: recovery.RetryConfig{
			MaxAttempts:   3,
			InitialDelay:  time.Second,
			MaxDelay:      10 * time.Second,
			BackoffFactor: 2.0,
			Jitter:        true,
			RetryCondition: func(err error) bool {
				return recovery.IsRetryable(err) || recovery.IsTemporaryError(err)
			},
			Logger: logger,
		},
	}

	// Register notification channels
	nm.channels[models.ChannelTypeDingTalk] = NewDingTalkChannel(logger)
	nm.channels[models.ChannelTypeWeChatWork] = NewWeChatWorkChannel(logger)
	nm.channels[models.ChannelTypeEmail] = NewEmailChannel(logger)
	nm.channels[models.ChannelTypeSMS] = NewSMSChannel(logger)

	return nm
}

// SendNotification sends a notification through the specified channel with retry and circuit breaker
func (nm *NotificationManager) SendNotification(ctx context.Context, channelType models.NotificationChannelType, message *NotificationMessage) error {
	channel, exists := nm.channels[channelType]
	if !exists {
		return errors.NewNotFoundError("notification channel", string(channelType))
	}

	start := time.Now()
	
	// Use retry with circuit breaker
	err := recovery.RetryWithCircuitBreaker(ctx, nm.retryConfig, nm.circuitBreaker, func(ctx context.Context) error {
		return channel.Send(ctx, message)
	})
	
	duration := time.Since(start)

	if err != nil {
		nm.logger.WithFields(logrus.Fields{
			"channel_type": channelType,
			"duration":     duration,
			"error":        err.Error(),
		}).Error("Failed to send notification after retries")
		
		// Record metrics
		metrics.RecordNotificationSent(string(channelType), "failed", duration.Seconds())
		metrics.RecordNotificationError(string(channelType), "send_error")
		
		return errors.Wrap(err, "NOTIFICATION_FAILED", 
			fmt.Sprintf("Failed to send notification via %s", channelType),
			500)
	}

	nm.logger.WithFields(logrus.Fields{
		"channel_type": channelType,
		"duration":     duration,
	}).Info("Notification sent successfully")
	
	// Record success metrics
	metrics.RecordNotificationSent(string(channelType), "success", duration.Seconds())

	return nil
}

// TestChannel tests a notification channel
func (nm *NotificationManager) TestChannel(ctx context.Context, channelType models.NotificationChannelType, testMessage string) error {
	channel, exists := nm.channels[channelType]
	if !exists {
		return errors.NewNotFoundError("notification channel", string(channelType))
	}

	// Use simple retry for testing (no circuit breaker)
	testRetryConfig := recovery.RetryConfig{
		MaxAttempts:   2,
		InitialDelay:  500 * time.Millisecond,
		MaxDelay:      2 * time.Second,
		BackoffFactor: 1.5,
		Jitter:        false,
		RetryCondition: func(err error) bool {
			return recovery.IsRetryable(err)
		},
		Logger: nm.logger,
	}

	return recovery.Retry(ctx, testRetryConfig, func(ctx context.Context) error {
		return channel.Test(ctx, testMessage)
	})
}

// SendAlertNotification sends an alert notification with proper formatting
func (nm *NotificationManager) SendAlertNotification(ctx context.Context, alert *models.Alert, channelConfig models.JSONB, channelType models.NotificationChannelType) error {
	message := nm.formatAlertMessage(alert, channelConfig)
	return nm.SendNotification(ctx, channelType, message)
}

// formatAlertMessage formats an alert into a notification message
func (nm *NotificationManager) formatAlertMessage(alert *models.Alert, channelConfig models.JSONB) *NotificationMessage {
	// Extract alert information
	alertName := nm.getAlertLabel(alert, "alertname", "Unknown Alert")
	instance := nm.getAlertLabel(alert, "instance", "unknown")
	description := nm.getAlertAnnotation(alert, "description", "No description available")
	summary := nm.getAlertAnnotation(alert, "summary", "")

	// Determine level based on severity
	level := "info"
	switch alert.Severity {
	case string(models.AlertSeverityCritical):
		level = "error"
	case string(models.AlertSeverityWarning):
		level = "warning"
	default:
		level = "info"
	}

	// Format title
	title := fmt.Sprintf("[%s] %s", alert.Status, alertName)

	// Format content
	content := fmt.Sprintf("**Alert**: %s\n", alertName)
	content += fmt.Sprintf("**Status**: %s\n", alert.Status)
	content += fmt.Sprintf("**Severity**: %s\n", alert.Severity)
	content += fmt.Sprintf("**Instance**: %s\n", instance)
	
	if summary != "" {
		content += fmt.Sprintf("**Summary**: %s\n", summary)
	}
	
	content += fmt.Sprintf("**Description**: %s\n", description)
	content += fmt.Sprintf("**Started At**: %s\n", alert.StartsAt.Format("2006-01-02 15:04:05"))
	
	if alert.EndsAt != nil {
		content += fmt.Sprintf("**Ended At**: %s\n", alert.EndsAt.Format("2006-01-02 15:04:05"))
	}

	return &NotificationMessage{
		Title:         title,
		Content:       content,
		Level:         level,
		Alert:         alert,
		ChannelConfig: channelConfig,
	}
}

// getAlertLabel gets a label value from alert with fallback
func (nm *NotificationManager) getAlertLabel(alert *models.Alert, key, fallback string) string {
	if alert.Labels != nil {
		if value, exists := alert.Labels[key]; exists {
			if str, ok := value.(string); ok {
				return str
			}
		}
	}
	return fallback
}

// getAlertAnnotation gets an annotation value from alert with fallback
func (nm *NotificationManager) getAlertAnnotation(alert *models.Alert, key, fallback string) string {
	if alert.Annotations != nil {
		if value, exists := alert.Annotations[key]; exists {
			if str, ok := value.(string); ok {
				return str
			}
		}
	}
	return fallback
}

// GetSupportedChannels returns list of supported notification channel types
func (nm *NotificationManager) GetSupportedChannels() []models.NotificationChannelType {
	channels := make([]models.NotificationChannelType, 0, len(nm.channels))
	for channelType := range nm.channels {
		channels = append(channels, channelType)
	}
	return channels
}