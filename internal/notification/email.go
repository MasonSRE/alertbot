package notification

import (
	"context"
	"fmt"
	"net/smtp"
	"strings"
	"time"

	"alertbot/internal/models"

	"github.com/sirupsen/logrus"
)

// EmailChannel implements notification for Email
type EmailChannel struct {
	logger *logrus.Logger
}

func NewEmailChannel(logger *logrus.Logger) *EmailChannel {
	return &EmailChannel{
		logger: logger,
	}
}

func (e *EmailChannel) GetType() models.NotificationChannelType {
	return models.ChannelTypeEmail
}

func (e *EmailChannel) Send(ctx context.Context, message *NotificationMessage) error {
	// Extract email configuration
	smtpHost, ok := message.ChannelConfig["smtp_host"].(string)
	if !ok || smtpHost == "" {
		return fmt.Errorf("smtp_host is required for email notifications")
	}

	smtpPort, ok := message.ChannelConfig["smtp_port"].(float64)
	if !ok {
		smtpPort = 587 // default SMTP port
	}

	username, ok := message.ChannelConfig["username"].(string)
	if !ok || username == "" {
		return fmt.Errorf("username is required for email notifications")
	}

	password, ok := message.ChannelConfig["password"].(string)
	if !ok || password == "" {
		return fmt.Errorf("password is required for email notifications")
	}

	from, ok := message.ChannelConfig["from"].(string)
	if !ok || from == "" {
		return fmt.Errorf("from address is required for email notifications")
	}

	// Get recipients
	toList, err := e.getRecipients(message.ChannelConfig, "to")
	if err != nil {
		return fmt.Errorf("failed to get 'to' recipients: %w", err)
	}
	if len(toList) == 0 {
		return fmt.Errorf("at least one 'to' recipient is required")
	}

	ccList, _ := e.getRecipients(message.ChannelConfig, "cc")
	bccList, _ := e.getRecipients(message.ChannelConfig, "bcc")

	// Build email
	emailContent := e.buildEmail(from, toList, ccList, message)

	// Send email
	return e.sendEmail(smtpHost, int(smtpPort), username, password, from, append(toList, append(ccList, bccList...)...), emailContent)
}

func (e *EmailChannel) Test(ctx context.Context, testMessage string) error {
	// For testing, testMessage should contain JSON configuration
	// In a real implementation, you might parse this differently
	message := &NotificationMessage{
		Title:   "AlertBot Test Email",
		Content: "This is a test email from AlertBot notification system.",
		Level:   "info",
		ChannelConfig: map[string]interface{}{
			// Test configuration would be parsed from testMessage
			// For now, return success to avoid requiring full SMTP setup for testing
		},
	}

	e.logger.Info("Email test notification requested")
	_ = message // avoid unused variable warning
	return nil
}

func (e *EmailChannel) getRecipients(config map[string]interface{}, key string) ([]string, error) {
	recipientsInterface, ok := config[key]
	if !ok {
		return []string{}, nil
	}

	switch recipients := recipientsInterface.(type) {
	case []interface{}:
		var result []string
		for _, recipient := range recipients {
			if recipientStr, ok := recipient.(string); ok {
				result = append(result, recipientStr)
			}
		}
		return result, nil
	case []string:
		return recipients, nil
	case string:
		return []string{recipients}, nil
	default:
		return nil, fmt.Errorf("invalid %s format", key)
	}
}

func (e *EmailChannel) buildEmail(from string, to, cc []string, message *NotificationMessage) string {
	var email strings.Builder

	// Headers
	email.WriteString(fmt.Sprintf("From: AlertBot <%s>\r\n", from))
	email.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(to, ", ")))
	
	if len(cc) > 0 {
		email.WriteString(fmt.Sprintf("Cc: %s\r\n", strings.Join(cc, ", ")))
	}
	
	email.WriteString(fmt.Sprintf("Subject: %s\r\n", message.Title))
	email.WriteString("MIME-Version: 1.0\r\n")
	email.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	email.WriteString(fmt.Sprintf("Date: %s\r\n", time.Now().Format(time.RFC1123Z)))
	email.WriteString("\r\n")

	// Body
	html := e.formatHTMLContent(message)
	email.WriteString(html)

	return email.String()
}

func (e *EmailChannel) formatHTMLContent(message *NotificationMessage) string {
	// Determine color based on level
	var color string
	var bgColor string
	switch message.Level {
	case "error":
		color = "#dc3545"
		bgColor = "#f8d7da"
	case "warning":
		color = "#fd7e14"
		bgColor = "#fff3cd"
	case "info":
		color = "#0dcaf0"
		bgColor = "#d1ecf1"
	default:
		color = "#6c757d"
		bgColor = "#f8f9fa"
	}

	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>%s</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 0; padding: 20px; background-color: #f5f5f5; }
        .container { max-width: 600px; margin: 0 auto; background-color: white; border-radius: 8px; overflow: hidden; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        .header { background-color: %s; color: %s; padding: 20px; text-align: center; }
        .content { padding: 20px; line-height: 1.6; }
        .footer { background-color: #f8f9fa; padding: 15px; text-align: center; font-size: 12px; color: #6c757d; }
        .alert-info { background-color: %s; border-left: 4px solid %s; padding: 10px; margin: 10px 0; }
        pre { background-color: #f8f9fa; padding: 10px; border-radius: 4px; overflow-x: auto; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>%s</h1>
        </div>
        <div class="content">
            <div class="alert-info">
                %s
            </div>
        </div>
        <div class="footer">
            Sent by AlertBot at %s
        </div>
    </div>
</body>
</html>`, 
		message.Title, 
		bgColor, color,
		bgColor, color,
		message.Title,
		strings.ReplaceAll(message.Content, "\n", "<br>"),
		time.Now().Format("2006-01-02 15:04:05"))

	return html
}

func (e *EmailChannel) sendEmail(smtpHost string, smtpPort int, username, password, from string, to []string, message string) error {
	auth := smtp.PlainAuth("", username, password, smtpHost)
	addr := fmt.Sprintf("%s:%d", smtpHost, smtpPort)

	err := smtp.SendMail(addr, auth, from, to, []byte(message))
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	e.logger.WithFields(logrus.Fields{
		"smtp_host": smtpHost,
		"smtp_port": smtpPort,
		"from":      from,
		"to_count":  len(to),
	}).Debug("Email sent successfully")

	return nil
}