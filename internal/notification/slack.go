package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"alertbot/internal/models"

	"github.com/sirupsen/logrus"
)

// SlackChannel implements Slack notification channel using webhooks
type SlackChannel struct {
	logger *logrus.Logger
	client *http.Client
}

// SlackConfig represents Slack channel configuration
type SlackConfig struct {
	WebhookURL string `json:"webhook_url" validate:"required,url"`
	Channel    string `json:"channel" validate:"required"`
	Username   string `json:"username,omitempty"`
	IconEmoji  string `json:"icon_emoji,omitempty"`
	IconURL    string `json:"icon_url,omitempty"`
}

// SlackMessage represents a message to be sent to Slack
type SlackMessage struct {
	Text        string            `json:"text,omitempty"`
	Channel     string            `json:"channel,omitempty"`
	Username    string            `json:"username,omitempty"`
	IconEmoji   string            `json:"icon_emoji,omitempty"`
	IconURL     string            `json:"icon_url,omitempty"`
	Attachments []SlackAttachment `json:"attachments,omitempty"`
}

// SlackAttachment represents a Slack message attachment with rich formatting
type SlackAttachment struct {
	Color      string       `json:"color,omitempty"`
	Pretext    string       `json:"pretext,omitempty"`
	AuthorName string       `json:"author_name,omitempty"`
	AuthorIcon string       `json:"author_icon,omitempty"`
	Title      string       `json:"title,omitempty"`
	TitleLink  string       `json:"title_link,omitempty"`
	Text       string       `json:"text,omitempty"`
	Fields     []SlackField `json:"fields,omitempty"`
	Timestamp  int64        `json:"ts,omitempty"`
	Footer     string       `json:"footer,omitempty"`
	FooterIcon string       `json:"footer_icon,omitempty"`
}

// SlackField represents a field in a Slack attachment
type SlackField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

// SlackResponse represents Slack webhook response
type SlackResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

func NewSlackChannel(logger *logrus.Logger) *SlackChannel {
	return &SlackChannel{
		logger: logger,
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:       10,
				IdleConnTimeout:    30 * time.Second,
				DisableCompression: true,
			},
		},
	}
}

func (s *SlackChannel) Send(ctx context.Context, message *NotificationMessage) error {
	// Extract Slack configuration from channel config
	config, err := s.extractConfig(message.ChannelConfig)
	if err != nil {
		return fmt.Errorf("invalid Slack configuration: %w", err)
	}

	// Create Slack message with rich formatting
	slackMessage := s.formatMessage(message, config)

	// Send message
	if err := s.sendMessage(ctx, config.WebhookURL, slackMessage); err != nil {
		return fmt.Errorf("failed to send Slack message: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"channel": config.Channel,
		"level":   message.Level,
	}).Info("Slack notification sent successfully")

	return nil
}

func (s *SlackChannel) Test(ctx context.Context, testMessage string) error {
	// Parse test configuration from JSON
	var testConfig map[string]interface{}
	if err := parseJSONConfig(testMessage, &testConfig); err != nil {
		return fmt.Errorf("invalid test configuration: %w", err)
	}

	// Create test notification message
	message := &NotificationMessage{
		Title:         "AlertBot Slack Test",
		Content:       "✅ This is a test message from AlertBot. If you receive this, your Slack integration is working correctly!",
		Level:         "info",
		ChannelConfig: testConfig,
	}

	return s.Send(ctx, message)
}

func (s *SlackChannel) GetType() models.NotificationChannelType {
	return models.ChannelTypeSlack
}

// extractConfig extracts and validates Slack configuration
func (s *SlackChannel) extractConfig(config map[string]interface{}) (*SlackConfig, error) {
	webhookURL, ok := config["webhook_url"].(string)
	if !ok || webhookURL == "" {
		return nil, fmt.Errorf("webhook_url is required and must be a string")
	}

	// Validate webhook URL format
	if _, err := url.Parse(webhookURL); err != nil {
		return nil, fmt.Errorf("invalid webhook_url format: %w", err)
	}

	// Check if it's a Slack webhook URL
	if !strings.Contains(webhookURL, "hooks.slack.com") {
		return nil, fmt.Errorf("webhook_url must be a valid Slack webhook URL")
	}

	channel, ok := config["channel"].(string)
	if !ok || channel == "" {
		return nil, fmt.Errorf("channel is required and must be a string")
	}

	// Validate channel format (should start with # for public channels or @ for DMs)
	if !strings.HasPrefix(channel, "#") && !strings.HasPrefix(channel, "@") {
		return nil, fmt.Errorf("channel must start with # (public channel) or @ (direct message)")
	}

	slackConfig := &SlackConfig{
		WebhookURL: webhookURL,
		Channel:    channel,
		Username:   "AlertBot", // Default username
		IconEmoji:  ":warning:", // Default icon
	}

	// Optional fields
	if username, ok := config["username"].(string); ok && username != "" {
		slackConfig.Username = username
	}

	if iconEmoji, ok := config["icon_emoji"].(string); ok && iconEmoji != "" {
		slackConfig.IconEmoji = iconEmoji
	}

	if iconURL, ok := config["icon_url"].(string); ok && iconURL != "" {
		slackConfig.IconURL = iconURL
	}

	return slackConfig, nil
}

// formatMessage formats the notification message for Slack with rich attachments
func (s *SlackChannel) formatMessage(message *NotificationMessage, config *SlackConfig) *SlackMessage {
	slackMessage := &SlackMessage{
		Channel:   config.Channel,
		Username:  config.Username,
		IconEmoji: config.IconEmoji,
		IconURL:   config.IconURL,
	}

	// Create main attachment with rich formatting
	attachment := SlackAttachment{
		Color:     s.getLevelColor(message.Level),
		Title:     message.Title,
		Text:      message.Content,
		Timestamp: time.Now().Unix(),
		Footer:    "AlertBot",
		FooterIcon: "https://via.placeholder.com/16x16/007ACC/ffffff.png?text=A",
	}

	// Add alert-specific fields if alert is present
	if message.Alert != nil {
		attachment.Fields = s.formatAlertFields(message.Alert)
		
		// Set pretext with emoji based on severity
		emoji := s.getSeverityEmoji(message.Alert.Severity)
		attachment.Pretext = fmt.Sprintf("%s *%s Alert*", emoji, strings.Title(message.Alert.Severity))
		
		// Override color based on alert severity
		attachment.Color = s.getSeverityColor(message.Alert.Severity)
	}

	slackMessage.Attachments = []SlackAttachment{attachment}

	return slackMessage
}

// formatAlertFields formats alert information as Slack attachment fields
func (s *SlackChannel) formatAlertFields(alert *models.Alert) []SlackField {
	var fields []SlackField

	// Basic alert information
	if alertName := s.getAlertLabel(alert, "alertname", ""); alertName != "" {
		fields = append(fields, SlackField{
			Title: "Alert Name",
			Value: alertName,
			Short: true,
		})
	}

	fields = append(fields, SlackField{
		Title: "Status",
		Value: strings.Title(alert.Status),
		Short: true,
	})

	fields = append(fields, SlackField{
		Title: "Severity",
		Value: strings.Title(alert.Severity),
		Short: true,
	})

	if instance := s.getAlertLabel(alert, "instance", ""); instance != "" {
		fields = append(fields, SlackField{
			Title: "Instance",
			Value: instance,
			Short: true,
		})
	}

	if job := s.getAlertLabel(alert, "job", ""); job != "" {
		fields = append(fields, SlackField{
			Title: "Job",
			Value: job,
			Short: true,
		})
	}

	// Started time
	fields = append(fields, SlackField{
		Title: "Started",
		Value: alert.StartsAt.Format("2006-01-02 15:04:05"),
		Short: true,
	})

	// Description from annotations
	if description := s.getAlertAnnotation(alert, "description", ""); description != "" {
		fields = append(fields, SlackField{
			Title: "Description",
			Value: description,
			Short: false,
		})
	}

	return fields
}

// getLevelColor returns appropriate color for message level
func (s *SlackChannel) getLevelColor(level string) string {
	switch strings.ToLower(level) {
	case "critical", "error":
		return "danger"
	case "warning":
		return "warning"
	case "info":
		return "good"
	default:
		return "#36a64f"
	}
}

// getSeverityColor returns appropriate color for alert severity
func (s *SlackChannel) getSeverityColor(severity string) string {
	switch strings.ToLower(severity) {
	case "critical":
		return "danger"
	case "warning":
		return "warning"
	case "info":
		return "good"
	default:
		return "#36a64f"
	}
}

// getSeverityEmoji returns appropriate emoji for alert severity
func (s *SlackChannel) getSeverityEmoji(severity string) string {
	switch strings.ToLower(severity) {
	case "critical":
		return ":fire:"
	case "warning":
		return ":warning:"
	case "info":
		return ":information_source:"
	default:
		return ":bell:"
	}
}

// getAlertLabel gets a label value from alert with fallback
func (s *SlackChannel) getAlertLabel(alert *models.Alert, key, fallback string) string {
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
func (s *SlackChannel) getAlertAnnotation(alert *models.Alert, key, fallback string) string {
	if alert.Annotations != nil {
		if value, exists := alert.Annotations[key]; exists {
			if str, ok := value.(string); ok {
				return str
			}
		}
	}
	return fallback
}

// sendMessage sends a message to Slack webhook
func (s *SlackChannel) sendMessage(ctx context.Context, webhookURL string, message *SlackMessage) error {
	// Marshal message to JSON
	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "AlertBot/1.0")

	// Send request
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Slack webhook returned status %d", resp.StatusCode)
	}

	// Slack responds with "ok" for successful webhook calls
	var buf bytes.Buffer
	buf.ReadFrom(resp.Body)
	responseText := strings.TrimSpace(buf.String())
	
	if responseText != "ok" {
		return fmt.Errorf("Slack webhook error: %s", responseText)
	}

	return nil
}

// ValidateSlackConfig validates Slack configuration by sending a test message
func ValidateSlackConfig(ctx context.Context, webhookURL, channel string) error {
	client := &http.Client{Timeout: 10 * time.Second}
	
	// Create test message
	testMessage := SlackMessage{
		Channel:   channel,
		Username:  "AlertBot",
		IconEmoji: ":robot_face:",
		Text:      "🤖 AlertBot configuration test - this message confirms your Slack integration is working correctly!",
	}
	
	// Marshal to JSON
	jsonData, err := json.Marshal(testMessage)
	if err != nil {
		return fmt.Errorf("failed to marshal test message: %w", err)
	}
	
	// Send test message
	req, err := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "AlertBot/1.0")
	
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to Slack webhook: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Slack webhook returned status %d", resp.StatusCode)
	}
	
	// Check response
	var buf bytes.Buffer
	buf.ReadFrom(resp.Body)
	responseText := strings.TrimSpace(buf.String())
	
	if responseText != "ok" {
		return fmt.Errorf("Slack webhook test failed: %s", responseText)
	}
	
	return nil
}