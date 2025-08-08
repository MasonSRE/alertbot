package notification

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/smtp"
	"regexp"
	"strconv"
	"strings"
	"time"

	"alertbot/internal/models"

	"github.com/sirupsen/logrus"
)

// EmailChannel implements notification for Email
type EmailChannel struct {
	logger      *logrus.Logger
	emailRegex  *regexp.Regexp
}

// EmailConfig represents email configuration
type EmailConfig struct {
	SMTPHost     string   `json:"smtp_host" validate:"required"`
	SMTPPort     int      `json:"smtp_port" validate:"required,min=1,max=65535"`
	Username     string   `json:"username" validate:"required"`
	Password     string   `json:"password" validate:"required"`
	From         string   `json:"from" validate:"required,email"`
	To           []string `json:"to" validate:"required,min=1,dive,email"`
	CC           []string `json:"cc,omitempty" validate:"omitempty,dive,email"`
	BCC          []string `json:"bcc,omitempty" validate:"omitempty,dive,email"`
	UseTLS       bool     `json:"use_tls"`
	UseStartTLS  bool     `json:"use_starttls"`
	SkipVerify   bool     `json:"skip_verify"`
}

func NewEmailChannel(logger *logrus.Logger) *EmailChannel {
	// RFC 5322 compliant email regex
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9.!#$%&'*+/=?^_` + "`" + `{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$`)
	
	return &EmailChannel{
		logger:     logger,
		emailRegex: emailRegex,
	}
}

func (e *EmailChannel) GetType() models.NotificationChannelType {
	return models.ChannelTypeEmail
}

func (e *EmailChannel) Send(ctx context.Context, message *NotificationMessage) error {
	// Extract and validate email configuration
	config, err := e.extractConfig(message.ChannelConfig)
	if err != nil {
		return fmt.Errorf("invalid email configuration: %w", err)
	}

	// Validate all email addresses
	if err := e.validateConfig(config); err != nil {
		return fmt.Errorf("email configuration validation failed: %w", err)
	}

	// Build email content
	emailContent := e.buildEmail(config, message)

	// Send email with TLS support
	if err := e.sendEmailWithTLS(ctx, config, emailContent); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	e.logger.WithFields(logrus.Fields{
		"smtp_host": config.SMTPHost,
		"smtp_port": config.SMTPPort,
		"from":      config.From,
		"to_count":  len(config.To),
		"cc_count":  len(config.CC),
		"bcc_count": len(config.BCC),
		"use_tls":   config.UseTLS,
	}).Info("Email notification sent successfully")

	return nil
}

func (e *EmailChannel) Test(ctx context.Context, testMessage string) error {
	// Parse test configuration from JSON
	var testConfig map[string]interface{}
	if err := parseJSONConfig(testMessage, &testConfig); err != nil {
		return fmt.Errorf("invalid test configuration: %w", err)
	}

	// Extract and validate configuration
	config, err := e.extractConfig(testConfig)
	if err != nil {
		return fmt.Errorf("invalid email configuration: %w", err)
	}

	if err := e.validateConfig(config); err != nil {
		return fmt.Errorf("email configuration validation failed: %w", err)
	}

	// Test SMTP connection first
	if err := e.testConnection(ctx, config); err != nil {
		return fmt.Errorf("SMTP connection test failed: %w", err)
	}

	// Create test message
	message := &NotificationMessage{
		Title:         "AlertBot Email Test",
		Content:       "✅ This is a test email from AlertBot notification system.\n\nIf you receive this message, your email configuration is working correctly!",
		Level:         "info",
		ChannelConfig: testConfig,
	}

	return e.Send(ctx, message)
}

// extractConfig extracts and converts email configuration from generic map
func (e *EmailChannel) extractConfig(config map[string]interface{}) (*EmailConfig, error) {
	smtpHost, ok := config["smtp_host"].(string)
	if !ok || smtpHost == "" {
		return nil, fmt.Errorf("smtp_host is required")
	}

	smtpPort := 587 // default
	if portVal, ok := config["smtp_port"]; ok {
		switch port := portVal.(type) {
		case float64:
			smtpPort = int(port)
		case int:
			smtpPort = port
		case string:
			if p, err := strconv.Atoi(port); err == nil {
				smtpPort = p
			}
		}
	}

	username, ok := config["username"].(string)
	if !ok || username == "" {
		return nil, fmt.Errorf("username is required")
	}

	password, ok := config["password"].(string)
	if !ok || password == "" {
		return nil, fmt.Errorf("password is required")
	}

	from, ok := config["from"].(string)
	if !ok || from == "" {
		return nil, fmt.Errorf("from address is required")
	}

	to, err := e.getStringSlice(config, "to")
	if err != nil {
		return nil, fmt.Errorf("invalid 'to' recipients: %w", err)
	}
	if len(to) == 0 {
		return nil, fmt.Errorf("at least one 'to' recipient is required")
	}

	cc, _ := e.getStringSlice(config, "cc")
	bcc, _ := e.getStringSlice(config, "bcc")

	// TLS settings
	useTLS, _ := config["use_tls"].(bool)
	useStartTLS, _ := config["use_starttls"].(bool)
	skipVerify, _ := config["skip_verify"].(bool)

	// If neither TLS nor StartTLS is specified, default to StartTLS for port 587
	if !useTLS && !useStartTLS {
		useStartTLS = (smtpPort == 587)
	}

	return &EmailConfig{
		SMTPHost:    smtpHost,
		SMTPPort:    smtpPort,
		Username:    username,
		Password:    password,
		From:        from,
		To:          to,
		CC:          cc,
		BCC:         bcc,
		UseTLS:      useTLS,
		UseStartTLS: useStartTLS,
		SkipVerify:  skipVerify,
	}, nil
}

// getStringSlice converts interface{} to []string for recipient lists
func (e *EmailChannel) getStringSlice(config map[string]interface{}, key string) ([]string, error) {
	val, ok := config[key]
	if !ok {
		return []string{}, nil
	}

	switch v := val.(type) {
	case []interface{}:
		var result []string
		for _, item := range v {
			if str, ok := item.(string); ok {
				result = append(result, str)
			}
		}
		return result, nil
	case []string:
		return v, nil
	case string:
		return []string{v}, nil
	default:
		return nil, fmt.Errorf("invalid %s format", key)
	}
}

// validateConfig validates email configuration and email addresses
func (e *EmailChannel) validateConfig(config *EmailConfig) error {
	// Validate SMTP port
	if config.SMTPPort < 1 || config.SMTPPort > 65535 {
		return fmt.Errorf("invalid SMTP port: %d", config.SMTPPort)
	}

	// Validate from email
	if !e.isValidEmail(config.From) {
		return fmt.Errorf("invalid from email address: %s", config.From)
	}

	// Validate to emails
	for _, email := range config.To {
		if !e.isValidEmail(email) {
			return fmt.Errorf("invalid to email address: %s", email)
		}
	}

	// Validate CC emails
	for _, email := range config.CC {
		if !e.isValidEmail(email) {
			return fmt.Errorf("invalid CC email address: %s", email)
		}
	}

	// Validate BCC emails
	for _, email := range config.BCC {
		if !e.isValidEmail(email) {
			return fmt.Errorf("invalid BCC email address: %s", email)
		}
	}

	return nil
}

// isValidEmail validates email address format
func (e *EmailChannel) isValidEmail(email string) bool {
	if len(email) > 254 {
		return false
	}
	return e.emailRegex.MatchString(email)
}

// buildEmail builds the complete email message with headers and body
func (e *EmailChannel) buildEmail(config *EmailConfig, message *NotificationMessage) string {
	var email strings.Builder

	// Email headers
	email.WriteString(fmt.Sprintf("From: AlertBot <%s>\r\n", config.From))
	email.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(config.To, ", ")))
	
	if len(config.CC) > 0 {
		email.WriteString(fmt.Sprintf("Cc: %s\r\n", strings.Join(config.CC, ", ")))
	}
	
	email.WriteString(fmt.Sprintf("Subject: %s\r\n", message.Title))
	email.WriteString("MIME-Version: 1.0\r\n")
	email.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	email.WriteString("Content-Transfer-Encoding: 8bit\r\n")
	email.WriteString(fmt.Sprintf("Date: %s\r\n", time.Now().Format(time.RFC1123Z)))
	email.WriteString(fmt.Sprintf("Message-ID: <%d.%s@alertbot>\r\n", time.Now().Unix(), generateMessageID()))
	email.WriteString("\r\n")

	// HTML body
	html := e.formatHTMLContent(message)
	email.WriteString(html)

	return email.String()
}

// generateMessageID generates a unique message ID
func generateMessageID() string {
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), time.Now().Unix())
}

// parseJSONConfig parses JSON string into configuration map
func parseJSONConfig(jsonStr string, config interface{}) error {
	if jsonStr == "" {
		return fmt.Errorf("empty configuration")
	}
	return json.Unmarshal([]byte(jsonStr), config)
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

// sendEmailWithTLS sends email with TLS/StartTLS support
func (e *EmailChannel) sendEmailWithTLS(ctx context.Context, config *EmailConfig, message string) error {
	addr := fmt.Sprintf("%s:%d", config.SMTPHost, config.SMTPPort)
	
	// Collect all recipients
	allRecipients := make([]string, 0, len(config.To)+len(config.CC)+len(config.BCC))
	allRecipients = append(allRecipients, config.To...)
	allRecipients = append(allRecipients, config.CC...)
	allRecipients = append(allRecipients, config.BCC...)

	// Use different approaches based on TLS configuration
	if config.UseTLS {
		return e.sendWithTLS(ctx, config, addr, allRecipients, message)
	} else if config.UseStartTLS {
		return e.sendWithStartTLS(ctx, config, addr, allRecipients, message)
	} else {
		// Plain SMTP (not recommended for production)
		return e.sendPlain(ctx, config, addr, allRecipients, message)
	}
}

// sendWithTLS sends email using direct TLS connection
func (e *EmailChannel) sendWithTLS(ctx context.Context, config *EmailConfig, addr string, recipients []string, message string) error {
	tlsConfig := &tls.Config{
		ServerName:         config.SMTPHost,
		InsecureSkipVerify: config.SkipVerify,
	}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("failed to connect with TLS: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, config.SMTPHost)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer client.Close()

	return e.authenticateAndSend(client, config, recipients, message)
}

// sendWithStartTLS sends email using StartTLS
func (e *EmailChannel) sendWithStartTLS(ctx context.Context, config *EmailConfig, addr string, recipients []string, message string) error {
	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer client.Close()

	// Start TLS if supported
	if ok, _ := client.Extension("STARTTLS"); ok {
		tlsConfig := &tls.Config{
			ServerName:         config.SMTPHost,
			InsecureSkipVerify: config.SkipVerify,
		}
		if err := client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("failed to start TLS: %w", err)
		}
	}

	return e.authenticateAndSend(client, config, recipients, message)
}

// sendPlain sends email without encryption (not recommended)
func (e *EmailChannel) sendPlain(ctx context.Context, config *EmailConfig, addr string, recipients []string, message string) error {
	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer client.Close()

	return e.authenticateAndSend(client, config, recipients, message)
}

// authenticateAndSend handles authentication and message sending
func (e *EmailChannel) authenticateAndSend(client *smtp.Client, config *EmailConfig, recipients []string, message string) error {
	// Authenticate
	auth := smtp.PlainAuth("", config.Username, config.Password, config.SMTPHost)
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	// Set sender
	if err := client.Mail(config.From); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	// Set recipients
	for _, recipient := range recipients {
		if err := client.Rcpt(recipient); err != nil {
			return fmt.Errorf("failed to set recipient %s: %w", recipient, err)
		}
	}

	// Send message
	wc, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to start data transmission: %w", err)
	}
	defer wc.Close()

	if _, err := wc.Write([]byte(message)); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	return nil
}

// testConnection tests SMTP connection without sending email
func (e *EmailChannel) testConnection(ctx context.Context, config *EmailConfig) error {
	addr := fmt.Sprintf("%s:%d", config.SMTPHost, config.SMTPPort)
	
	// Test connection based on TLS configuration
	if config.UseTLS {
		tlsConfig := &tls.Config{
			ServerName:         config.SMTPHost,
			InsecureSkipVerify: config.SkipVerify,
		}
		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return fmt.Errorf("TLS connection failed: %w", err)
		}
		conn.Close()
	} else {
		// Test plain connection
		conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
		if err != nil {
			return fmt.Errorf("connection failed: %w", err)
		}
		conn.Close()
	}

	e.logger.WithFields(logrus.Fields{
		"smtp_host": config.SMTPHost,
		"smtp_port": config.SMTPPort,
		"use_tls":   config.UseTLS,
	}).Debug("SMTP connection test successful")

	return nil
}