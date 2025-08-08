package notification

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"alertbot/internal/models"

	"github.com/sirupsen/logrus"
)

// DingTalkChannel implements notification for DingTalk
type DingTalkChannel struct {
	logger *logrus.Logger
	client *http.Client
}

// DingTalkMessage represents a DingTalk message
type DingTalkMessage struct {
	MsgType  string                 `json:"msgtype"`
	Text     *DingTalkText          `json:"text,omitempty"`
	Markdown *DingTalkMarkdown      `json:"markdown,omitempty"`
	At       *DingTalkAt            `json:"at,omitempty"`
}

type DingTalkText struct {
	Content string `json:"content"`
}

type DingTalkMarkdown struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}

type DingTalkAt struct {
	AtMobiles []string `json:"atMobiles,omitempty"`
	IsAtAll   bool     `json:"isAtAll,omitempty"`
}

type DingTalkResponse struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}

func NewDingTalkChannel(logger *logrus.Logger) *DingTalkChannel {
	return &DingTalkChannel{
		logger: logger,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (d *DingTalkChannel) GetType() models.NotificationChannelType {
	return models.ChannelTypeDingTalk
}

func (d *DingTalkChannel) Send(ctx context.Context, message *NotificationMessage) error {
	webhookURL, ok := message.ChannelConfig["webhook_url"].(string)
	if !ok || webhookURL == "" {
		return fmt.Errorf("webhook_url is required for DingTalk notifications")
	}

	// Generate signature if secret is provided
	secret, _ := message.ChannelConfig["secret"].(string)
	if secret != "" {
		var err error
		webhookURL, err = d.signURL(webhookURL, secret)
		if err != nil {
			return fmt.Errorf("failed to sign DingTalk URL: %w", err)
		}
	}

	// Prepare the message
	dingMsg := d.formatMessage(message)

	// Send the message
	return d.sendMessage(ctx, webhookURL, dingMsg)
}

func (d *DingTalkChannel) Test(ctx context.Context, testMessage string) error {
	// DingTalk Test method should not be called directly from API layer
	// The old design expected testMessage to contain webhook URL, but API passes actual message
	// This method should only be called with proper configuration
	return fmt.Errorf("DingTalk Test method requires configuration. Use service layer with proper channel config instead")
}

func (d *DingTalkChannel) formatMessage(message *NotificationMessage) *DingTalkMessage {
	// Determine message color based on level
	var emoji string
	switch message.Level {
	case "error":
		emoji = "🔴"
	case "warning":
		emoji = "🟡"
	case "info":
		emoji = "🔵"
	default:
		emoji = "ℹ️"
	}

	// Check if we should use markdown format
	useMarkdown := true
	if val, ok := message.ChannelConfig["use_markdown"].(bool); ok {
		useMarkdown = val
	}

	dingMsg := &DingTalkMessage{}

	if useMarkdown {
		// Use markdown format for richer display
		dingMsg.MsgType = "markdown"
		dingMsg.Markdown = &DingTalkMarkdown{
			Title: fmt.Sprintf("%s %s", emoji, message.Title),
			Text:  fmt.Sprintf("## %s %s\n\n%s", emoji, message.Title, message.Content),
		}
	} else {
		// Use simple text format
		dingMsg.MsgType = "text"
		dingMsg.Text = &DingTalkText{
			Content: fmt.Sprintf("%s %s\n\n%s", emoji, message.Title, message.Content),
		}
	}

	// Configure @mentions
	dingMsg.At = &DingTalkAt{}
	
	if atMobiles, ok := message.ChannelConfig["at_mobiles"].([]interface{}); ok {
		for _, mobile := range atMobiles {
			if mobileStr, ok := mobile.(string); ok {
				dingMsg.At.AtMobiles = append(dingMsg.At.AtMobiles, mobileStr)
			}
		}
	}
	
	if atAll, ok := message.ChannelConfig["at_all"].(bool); ok {
		dingMsg.At.IsAtAll = atAll
	}

	return dingMsg
}

func (d *DingTalkChannel) signURL(webhookURL, secret string) (string, error) {
	timestamp := time.Now().UnixNano() / 1e6
	stringToSign := fmt.Sprintf("%d\n%s", timestamp, secret)
	
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(stringToSign))
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))
	
	// Add timestamp and signature to URL
	u, err := url.Parse(webhookURL)
	if err != nil {
		return "", err
	}
	
	q := u.Query()
	q.Set("timestamp", strconv.FormatInt(timestamp, 10))
	q.Set("sign", signature)
	u.RawQuery = q.Encode()
	
	return u.String(), nil
}

func (d *DingTalkChannel) sendMessage(ctx context.Context, webhookURL string, message *DingTalkMessage) error {
	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal DingTalk message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send DingTalk message: %w", err)
	}
	defer resp.Body.Close()

	var dingResp DingTalkResponse
	if err := json.NewDecoder(resp.Body).Decode(&dingResp); err != nil {
		return fmt.Errorf("failed to decode DingTalk response: %w", err)
	}

	if dingResp.ErrCode != 0 {
		return fmt.Errorf("DingTalk API error: %s (code: %d)", dingResp.ErrMsg, dingResp.ErrCode)
	}

	d.logger.WithFields(logrus.Fields{
		"webhook_url": webhookURL,
		"message_type": message.MsgType,
	}).Debug("DingTalk message sent successfully")

	return nil
}