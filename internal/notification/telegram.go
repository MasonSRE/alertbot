package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"alertbot/internal/models"

	"github.com/sirupsen/logrus"
)

// TelegramChannel implements Telegram Bot API notification channel
type TelegramChannel struct {
	logger *logrus.Logger
	client *http.Client
}

// TelegramConfig represents Telegram channel configuration
type TelegramConfig struct {
	BotToken string `json:"bot_token" validate:"required"`
	ChatID   string `json:"chat_id" validate:"required"`
}

// TelegramMessage represents a message to be sent to Telegram
type TelegramMessage struct {
	ChatID                string `json:"chat_id"`
	Text                  string `json:"text"`
	ParseMode             string `json:"parse_mode,omitempty"`
	DisableWebPagePreview bool   `json:"disable_web_page_preview,omitempty"`
	DisableNotification   bool   `json:"disable_notification,omitempty"`
}

// TelegramResponse represents Telegram API response
type TelegramResponse struct {
	OK          bool        `json:"ok"`
	Result      interface{} `json:"result,omitempty"`
	ErrorCode   int         `json:"error_code,omitempty"`
	Description string      `json:"description,omitempty"`
}

func NewTelegramChannel(logger *logrus.Logger) *TelegramChannel {
	return &TelegramChannel{
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

func (t *TelegramChannel) Send(ctx context.Context, message *NotificationMessage) error {
	// Extract Telegram configuration from channel config
	config, err := t.extractConfig(message.ChannelConfig)
	if err != nil {
		return fmt.Errorf("invalid Telegram configuration: %w", err)
	}

	// Format message for Telegram
	telegramText := t.formatMessage(message)

	// Create Telegram message
	telegramMessage := TelegramMessage{
		ChatID:                config.ChatID,
		Text:                  telegramText,
		ParseMode:             "Markdown",
		DisableWebPagePreview: true,
		DisableNotification:   message.Level == "info", // Don't notify for info messages
	}

	// Send message
	if err := t.sendMessage(ctx, config.BotToken, &telegramMessage); err != nil {
		return fmt.Errorf("failed to send Telegram message: %w", err)
	}

	t.logger.WithFields(logrus.Fields{
		"chat_id": config.ChatID,
		"level":   message.Level,
	}).Info("Telegram notification sent successfully")

	return nil
}

func (t *TelegramChannel) Test(ctx context.Context, testMessage string) error {
	// Create test notification message
	message := &NotificationMessage{
		Title:   "AlertBot Test Message",
		Content: testMessage,
		Level:   "info",
		ChannelConfig: map[string]interface{}{
			"bot_token": "test_token", // This should be provided by the frontend
			"chat_id":   "test_chat",  // This should be provided by the frontend
		},
	}

	return t.Send(ctx, message)
}

func (t *TelegramChannel) GetType() models.NotificationChannelType {
	return models.ChannelTypeTelegram
}

// extractConfig extracts and validates Telegram configuration
func (t *TelegramChannel) extractConfig(config map[string]interface{}) (*TelegramConfig, error) {
	botToken, ok := config["bot_token"].(string)
	if !ok || botToken == "" {
		return nil, fmt.Errorf("bot_token is required and must be a string")
	}

	chatID, ok := config["chat_id"].(string)
	if !ok || chatID == "" {
		return nil, fmt.Errorf("chat_id is required and must be a string")
	}

	// Validate bot token format (should be something like 123456789:ABCdefGHIjklMNOpqrSTUvwxYZ)
	if !strings.Contains(botToken, ":") {
		return nil, fmt.Errorf("invalid bot token format")
	}

	// Validate chat_id (can be numeric ID or channel username starting with @)
	if !strings.HasPrefix(chatID, "@") {
		if _, err := strconv.ParseInt(chatID, 10, 64); err != nil {
			return nil, fmt.Errorf("chat_id must be a numeric ID or channel username starting with @")
		}
	}

	return &TelegramConfig{
		BotToken: botToken,
		ChatID:   chatID,
	}, nil
}

// formatMessage formats the notification message for Telegram with Markdown
func (t *TelegramChannel) formatMessage(message *NotificationMessage) string {
	var builder strings.Builder

	// Title with emoji based on level
	emoji := t.getLevelEmoji(message.Level)
	builder.WriteString(fmt.Sprintf("%s *%s*\n\n", emoji, escapeMarkdown(message.Title)))

	// Content with proper Markdown formatting
	content := t.formatContent(message.Content, message.Alert)
	builder.WriteString(content)

	// Add footer with timestamp
	builder.WriteString(fmt.Sprintf("\n\n🕒 *Time:* %s", time.Now().Format("2006-01-02 15:04:05")))

	return builder.String()
}

// formatContent formats alert content with better structure for Telegram
func (t *TelegramChannel) formatContent(content string, alert *models.Alert) string {
	if alert == nil {
		return escapeMarkdown(content)
	}

	var builder strings.Builder

	// Alert details in a more readable format
	if alertName := t.getAlertLabel(alert, "alertname", ""); alertName != "" {
		builder.WriteString(fmt.Sprintf("📊 *Alert:* `%s`\n", escapeMarkdown(alertName)))
	}

	builder.WriteString(fmt.Sprintf("🚨 *Status:* `%s`\n", escapeMarkdown(alert.Status)))
	builder.WriteString(fmt.Sprintf("⚠️ *Severity:* `%s`\n", escapeMarkdown(alert.Severity)))

	if instance := t.getAlertLabel(alert, "instance", ""); instance != "" {
		builder.WriteString(fmt.Sprintf("🖥️ *Instance:* `%s`\n", escapeMarkdown(instance)))
	}

	if job := t.getAlertLabel(alert, "job", ""); job != "" {
		builder.WriteString(fmt.Sprintf("💼 *Job:* `%s`\n", escapeMarkdown(job)))
	}

	// Description from annotations
	if description := t.getAlertAnnotation(alert, "description", ""); description != "" {
		builder.WriteString(fmt.Sprintf("\n📝 *Description:*\n%s\n", escapeMarkdown(description)))
	}

	// Summary from annotations
	if summary := t.getAlertAnnotation(alert, "summary", ""); summary != "" {
		builder.WriteString(fmt.Sprintf("\n📋 *Summary:*\n%s\n", escapeMarkdown(summary)))
	}

	builder.WriteString(fmt.Sprintf("\n⏰ *Started:* `%s`", escapeMarkdown(alert.StartsAt.Format("2006-01-02 15:04:05"))))

	if alert.EndsAt != nil {
		builder.WriteString(fmt.Sprintf("\n🏁 *Ended:* `%s`", escapeMarkdown(alert.EndsAt.Format("2006-01-02 15:04:05"))))
	}

	return builder.String()
}

// getLevelEmoji returns appropriate emoji for alert level
func (t *TelegramChannel) getLevelEmoji(level string) string {
	switch strings.ToLower(level) {
	case "critical", "error":
		return "🔴"
	case "warning":
		return "🟡"
	case "info":
		return "🔵"
	default:
		return "ℹ️"
	}
}

// getAlertLabel gets a label value from alert with fallback
func (t *TelegramChannel) getAlertLabel(alert *models.Alert, key, fallback string) string {
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
func (t *TelegramChannel) getAlertAnnotation(alert *models.Alert, key, fallback string) string {
	if alert.Annotations != nil {
		if value, exists := alert.Annotations[key]; exists {
			if str, ok := value.(string); ok {
				return str
			}
		}
	}
	return fallback
}

// escapeMarkdown escapes special Markdown characters for Telegram
func escapeMarkdown(text string) string {
	// Telegram Markdown v2 requires escaping these characters: _*[]()~`>#+-=|{}.!
	replacer := strings.NewReplacer(
		"_", "\\_",
		"*", "\\*",
		"[", "\\[",
		"]", "\\]",
		"(", "\\(",
		")", "\\)",
		"~", "\\~",
		"`", "\\`",
		">", "\\>",
		"#", "\\#",
		"+", "\\+",
		"-", "\\-",
		"=", "\\=",
		"|", "\\|",
		"{", "\\{",
		"}", "\\}",
		".", "\\.",
		"!", "\\!",
	)
	return replacer.Replace(text)
}

// sendMessage sends a message to Telegram Bot API
func (t *TelegramChannel) sendMessage(ctx context.Context, botToken string, message *TelegramMessage) error {
	// Construct API URL
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)

	// Marshal message to JSON
	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "AlertBot/1.0")

	// Send request
	resp, err := t.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	var telegramResp TelegramResponse
	if err := json.NewDecoder(resp.Body).Decode(&telegramResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	// Check if request was successful
	if !telegramResp.OK {
		return fmt.Errorf("Telegram API error (code %d): %s", telegramResp.ErrorCode, telegramResp.Description)
	}

	return nil
}

// ValidateTelegramConfig validates Telegram configuration by sending a test message
func ValidateTelegramConfig(ctx context.Context, botToken, chatID string) error {
	client := &http.Client{Timeout: 10 * time.Second}
	
	// First, validate bot token by calling getMe
	getMeURL := fmt.Sprintf("https://api.telegram.org/bot%s/getMe", botToken)
	
	req, err := http.NewRequestWithContext(ctx, "GET", getMeURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to Telegram API: %w", err)
	}
	defer resp.Body.Close()
	
	var telegramResp TelegramResponse
	if err := json.NewDecoder(resp.Body).Decode(&telegramResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}
	
	if !telegramResp.OK {
		return fmt.Errorf("invalid bot token: %s", telegramResp.Description)
	}
	
	// Try to send a test message to validate chat_id
	testMessage := TelegramMessage{
		ChatID: chatID,
		Text:   "🤖 AlertBot configuration test - this message confirms your Telegram integration is working correctly!",
		ParseMode: "Markdown",
	}
	
	sendURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)
	jsonData, _ := json.Marshal(testMessage)
	
	req, err = http.NewRequestWithContext(ctx, "POST", sendURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create test message request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	
	resp, err = client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send test message: %w", err)
	}
	defer resp.Body.Close()
	
	if err := json.NewDecoder(resp.Body).Decode(&telegramResp); err != nil {
		return fmt.Errorf("failed to decode test message response: %w", err)
	}
	
	if !telegramResp.OK {
		return fmt.Errorf("failed to send message to chat %s: %s", chatID, telegramResp.Description)
	}
	
	return nil
}