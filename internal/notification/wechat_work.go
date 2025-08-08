package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"alertbot/internal/models"

	"github.com/sirupsen/logrus"
)

// WeChatWorkChannel implements notification for WeChat Work
type WeChatWorkChannel struct {
	logger *logrus.Logger
	client *http.Client
}

// WeChatWorkMessage represents a WeChat Work message
type WeChatWorkMessage struct {
	MsgType  string                   `json:"msgtype"`
	Text     *WeChatWorkText          `json:"text,omitempty"`
	Markdown *WeChatWorkMarkdown      `json:"markdown,omitempty"`
}

type WeChatWorkText struct {
	Content             string   `json:"content"`
	MentionedList       []string `json:"mentioned_list,omitempty"`
	MentionedMobileList []string `json:"mentioned_mobile_list,omitempty"`
}

type WeChatWorkMarkdown struct {
	Content string `json:"content"`
}

type WeChatWorkResponse struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}

func NewWeChatWorkChannel(logger *logrus.Logger) *WeChatWorkChannel {
	return &WeChatWorkChannel{
		logger: logger,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (w *WeChatWorkChannel) GetType() models.NotificationChannelType {
	return models.ChannelTypeWeChatWork
}

func (w *WeChatWorkChannel) Send(ctx context.Context, message *NotificationMessage) error {
	webhookURL, ok := message.ChannelConfig["webhook_url"].(string)
	if !ok || webhookURL == "" {
		return fmt.Errorf("webhook_url is required for WeChat Work notifications")
	}

	// Prepare the message
	wechatMsg := w.formatMessage(message)

	// Send the message
	return w.sendMessage(ctx, webhookURL, wechatMsg)
}

func (w *WeChatWorkChannel) Test(ctx context.Context, testMessage string) error {
	// WeChat Work Test method should not be called directly from API layer
	// The old design expected testMessage to contain webhook URL, but API passes actual message
	// This method should only be called with proper configuration
	return fmt.Errorf("WeChat Work Test method requires configuration. Use service layer with proper channel config instead")
}

func (w *WeChatWorkChannel) TestWithConfig(ctx context.Context, testMessage string, config models.JSONB) error {
	// Create a test notification message
	message := &NotificationMessage{
		Title:         "AlertBot Test Notification",
		Content:       testMessage,
		Level:         "info",
		ChannelConfig: config,
	}

	return w.Send(ctx, message)
}

func (w *WeChatWorkChannel) formatMessage(message *NotificationMessage) *WeChatWorkMessage {
	// Determine message color based on level
	var colorTag string
	switch message.Level {
	case "error":
		colorTag = `<font color="warning">`
	case "warning":
		colorTag = `<font color="warning">`
	case "info":
		colorTag = `<font color="info">`
	default:
		colorTag = `<font color="comment">`
	}

	// Check if we should use markdown format
	useMarkdown := true
	if val, ok := message.ChannelConfig["use_markdown"].(bool); ok {
		useMarkdown = val
	}

	wechatMsg := &WeChatWorkMessage{}

	if useMarkdown {
		// Use markdown format for richer display
		wechatMsg.MsgType = "markdown"
		
		content := fmt.Sprintf("## %s%s</font>\n\n%s", colorTag, message.Title, message.Content)
		
		// Add mentions for markdown
		if mentionedList, ok := message.ChannelConfig["mentioned_list"].([]interface{}); ok {
			for _, user := range mentionedList {
				if userStr, ok := user.(string); ok {
					content += fmt.Sprintf("\n<@%s>", userStr)
				}
			}
		}
		
		wechatMsg.Markdown = &WeChatWorkMarkdown{
			Content: content,
		}
	} else {
		// Use simple text format
		wechatMsg.MsgType = "text"
		
		textContent := fmt.Sprintf("%s\n\n%s", message.Title, message.Content)
		
		textMsg := &WeChatWorkText{
			Content: textContent,
		}
		
		// Configure mentions for text
		if mentionedList, ok := message.ChannelConfig["mentioned_list"].([]interface{}); ok {
			for _, user := range mentionedList {
				if userStr, ok := user.(string); ok {
					textMsg.MentionedList = append(textMsg.MentionedList, userStr)
				}
			}
		}
		
		if mentionedMobileList, ok := message.ChannelConfig["mentioned_mobile_list"].([]interface{}); ok {
			for _, mobile := range mentionedMobileList {
				if mobileStr, ok := mobile.(string); ok {
					textMsg.MentionedMobileList = append(textMsg.MentionedMobileList, mobileStr)
				}
			}
		}
		
		wechatMsg.Text = textMsg
	}

	return wechatMsg
}

func (w *WeChatWorkChannel) sendMessage(ctx context.Context, webhookURL string, message *WeChatWorkMessage) error {
	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal WeChat Work message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send WeChat Work message: %w", err)
	}
	defer resp.Body.Close()

	var wechatResp WeChatWorkResponse
	if err := json.NewDecoder(resp.Body).Decode(&wechatResp); err != nil {
		return fmt.Errorf("failed to decode WeChat Work response: %w", err)
	}

	if wechatResp.ErrCode != 0 {
		return fmt.Errorf("WeChat Work API error: %s (code: %d)", wechatResp.ErrMsg, wechatResp.ErrCode)
	}

	w.logger.WithFields(logrus.Fields{
		"webhook_url": webhookURL,
		"message_type": message.MsgType,
	}).Debug("WeChat Work message sent successfully")

	return nil
}