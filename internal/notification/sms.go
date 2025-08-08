package notification

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"

	"alertbot/internal/models"

	"github.com/sirupsen/logrus"
)

// SMSProvider interface for different SMS providers
type SMSProvider interface {
	SendSMS(ctx context.Context, phoneNumber, message string) error
	GetType() string
}

// SMSChannel implements notification for SMS
type SMSChannel struct {
	logger      *logrus.Logger
	providers   map[string]SMSProvider
	phoneRegex  *regexp.Regexp
	client      *http.Client
}

// SMSConfig represents SMS configuration for different providers
type SMSConfig struct {
	Provider     string   `json:"provider" validate:"required"`
	PhoneNumbers []string `json:"phone_numbers" validate:"required,min=1,dive,phone"`
	
	// Twilio configuration
	TwilioAccountSID string `json:"twilio_account_sid,omitempty"`
	TwilioAuthToken  string `json:"twilio_auth_token,omitempty"`
	TwilioFromNumber string `json:"twilio_from_number,omitempty"`
	
	// Aliyun SMS configuration
	AliyunAccessKeyID     string `json:"aliyun_access_key_id,omitempty"`
	AliyunAccessKeySecret string `json:"aliyun_access_key_secret,omitempty"`
	AliyunSignName        string `json:"aliyun_sign_name,omitempty"`
	AliyunTemplateCode    string `json:"aliyun_template_code,omitempty"`
	
	// Generic HTTP SMS configuration
	HTTPURL     string            `json:"http_url,omitempty"`
	HTTPMethod  string            `json:"http_method,omitempty"`
	HTTPHeaders map[string]string `json:"http_headers,omitempty"`
	HTTPParams  map[string]string `json:"http_params,omitempty"`
}

func NewSMSChannel(logger *logrus.Logger) *SMSChannel {
	// International phone number regex (E.164 format)
	phoneRegex := regexp.MustCompile(`^\+?[1-9]\d{1,14}$`)
	
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:       10,
			IdleConnTimeout:    30 * time.Second,
			DisableCompression: true,
		},
	}
	
	return &SMSChannel{
		logger:     logger,
		providers:  make(map[string]SMSProvider),
		phoneRegex: phoneRegex,
		client:     client,
	}
}

func (s *SMSChannel) GetType() models.NotificationChannelType {
	return models.ChannelTypeSMS
}

func (s *SMSChannel) Send(ctx context.Context, message *NotificationMessage) error {
	// Extract and validate SMS configuration
	config, err := s.extractConfig(message.ChannelConfig)
	if err != nil {
		return fmt.Errorf("invalid SMS configuration: %w", err)
	}

	// Validate configuration
	if err := s.validateConfig(config); err != nil {
		return fmt.Errorf("SMS configuration validation failed: %w", err)
	}

	// Get SMS provider
	provider, err := s.getProvider(config)
	if err != nil {
		return fmt.Errorf("failed to get SMS provider: %w", err)
	}

	// Format SMS content
	smsContent := s.formatSMSContent(message)

	// Send SMS to all phone numbers
	var errors []string
	for _, phoneNumber := range config.PhoneNumbers {
		if err := provider.SendSMS(ctx, phoneNumber, smsContent); err != nil {
			s.logger.WithFields(logrus.Fields{
				"phone_number": phoneNumber,
				"provider":     config.Provider,
				"error":        err.Error(),
			}).Error("Failed to send SMS")
			errors = append(errors, fmt.Sprintf("%s: %v", phoneNumber, err))
		} else {
			s.logger.WithFields(logrus.Fields{
				"phone_number": phoneNumber,
				"provider":     config.Provider,
			}).Info("SMS notification sent successfully")
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to send SMS to some numbers: %s", strings.Join(errors, "; "))
	}

	return nil
}

func (s *SMSChannel) Test(ctx context.Context, testMessage string) error {
	// Parse test configuration from JSON
	var testConfig map[string]interface{}
	if err := parseJSONConfig(testMessage, &testConfig); err != nil {
		return fmt.Errorf("invalid test configuration: %w", err)
	}

	// Extract and validate configuration
	config, err := s.extractConfig(testConfig)
	if err != nil {
		return fmt.Errorf("invalid SMS configuration: %w", err)
	}

	if err := s.validateConfig(config); err != nil {
		return fmt.Errorf("SMS configuration validation failed: %w", err)
	}

	// Test provider connection first
	_, err = s.getProvider(config)
	if err != nil {
		return fmt.Errorf("failed to get SMS provider: %w", err)
	}

	// Create test message
	message := &NotificationMessage{
		Title:         "AlertBot SMS Test",
		Content:       "✅ This is a test SMS from AlertBot notification system. If you receive this message, your SMS configuration is working correctly!",
		Level:         "info",
		ChannelConfig: testConfig,
	}

	return s.Send(ctx, message)
}

// extractConfig extracts and converts SMS configuration from generic map
func (s *SMSChannel) extractConfig(config map[string]interface{}) (*SMSConfig, error) {
	provider, ok := config["provider"].(string)
	if !ok || provider == "" {
		return nil, fmt.Errorf("provider is required")
	}

	phoneNumbers, err := s.getStringSlice(config, "phone_numbers")
	if err != nil {
		return nil, fmt.Errorf("invalid phone_numbers: %w", err)
	}
	if len(phoneNumbers) == 0 {
		return nil, fmt.Errorf("at least one phone number is required")
	}

	smsConfig := &SMSConfig{
		Provider:     provider,
		PhoneNumbers: phoneNumbers,
	}

	// Extract provider-specific configuration
	switch strings.ToLower(provider) {
	case "twilio":
		smsConfig.TwilioAccountSID, _ = config["twilio_account_sid"].(string)
		smsConfig.TwilioAuthToken, _ = config["twilio_auth_token"].(string)
		smsConfig.TwilioFromNumber, _ = config["twilio_from_number"].(string)
	
	case "aliyun":
		smsConfig.AliyunAccessKeyID, _ = config["aliyun_access_key_id"].(string)
		smsConfig.AliyunAccessKeySecret, _ = config["aliyun_access_key_secret"].(string)
		smsConfig.AliyunSignName, _ = config["aliyun_sign_name"].(string)
		smsConfig.AliyunTemplateCode, _ = config["aliyun_template_code"].(string)
	
	case "http":
		smsConfig.HTTPURL, _ = config["http_url"].(string)
		smsConfig.HTTPMethod, _ = config["http_method"].(string)
		if smsConfig.HTTPMethod == "" {
			smsConfig.HTTPMethod = "POST"
		}
		
		if headers, ok := config["http_headers"].(map[string]interface{}); ok {
			smsConfig.HTTPHeaders = make(map[string]string)
			for k, v := range headers {
				if str, ok := v.(string); ok {
					smsConfig.HTTPHeaders[k] = str
				}
			}
		}
		
		if params, ok := config["http_params"].(map[string]interface{}); ok {
			smsConfig.HTTPParams = make(map[string]string)
			for k, v := range params {
				if str, ok := v.(string); ok {
					smsConfig.HTTPParams[k] = str
				}
			}
		}
	}

	return smsConfig, nil
}

// getStringSlice converts interface{} to []string
func (s *SMSChannel) getStringSlice(config map[string]interface{}, key string) ([]string, error) {
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

// validateConfig validates SMS configuration
func (s *SMSChannel) validateConfig(config *SMSConfig) error {
	// Validate provider
	validProviders := []string{"twilio", "aliyun", "http"}
	isValidProvider := false
	for _, provider := range validProviders {
		if strings.ToLower(config.Provider) == provider {
			isValidProvider = true
			break
		}
	}
	if !isValidProvider {
		return fmt.Errorf("unsupported SMS provider: %s (supported: %s)", config.Provider, strings.Join(validProviders, ", "))
	}

	// Validate phone numbers
	for _, phoneNumber := range config.PhoneNumbers {
		if !s.isValidPhoneNumber(phoneNumber) {
			return fmt.Errorf("invalid phone number format: %s", phoneNumber)
		}
	}

	// Validate provider-specific configuration
	switch strings.ToLower(config.Provider) {
	case "twilio":
		if config.TwilioAccountSID == "" {
			return fmt.Errorf("twilio_account_sid is required for Twilio provider")
		}
		if config.TwilioAuthToken == "" {
			return fmt.Errorf("twilio_auth_token is required for Twilio provider")
		}
		if config.TwilioFromNumber == "" {
			return fmt.Errorf("twilio_from_number is required for Twilio provider")
		}
		if !s.isValidPhoneNumber(config.TwilioFromNumber) {
			return fmt.Errorf("invalid twilio_from_number format: %s", config.TwilioFromNumber)
		}
	
	case "aliyun":
		if config.AliyunAccessKeyID == "" {
			return fmt.Errorf("aliyun_access_key_id is required for Aliyun provider")
		}
		if config.AliyunAccessKeySecret == "" {
			return fmt.Errorf("aliyun_access_key_secret is required for Aliyun provider")
		}
		if config.AliyunSignName == "" {
			return fmt.Errorf("aliyun_sign_name is required for Aliyun provider")
		}
		if config.AliyunTemplateCode == "" {
			return fmt.Errorf("aliyun_template_code is required for Aliyun provider")
		}
	
	case "http":
		if config.HTTPURL == "" {
			return fmt.Errorf("http_url is required for HTTP provider")
		}
		if _, err := url.Parse(config.HTTPURL); err != nil {
			return fmt.Errorf("invalid http_url format: %w", err)
		}
	}

	return nil
}

// isValidPhoneNumber validates phone number format (E.164)
func (s *SMSChannel) isValidPhoneNumber(phoneNumber string) bool {
	if len(phoneNumber) < 8 || len(phoneNumber) > 16 {
		return false
	}
	return s.phoneRegex.MatchString(phoneNumber)
}

// getProvider returns the appropriate SMS provider
func (s *SMSChannel) getProvider(config *SMSConfig) (SMSProvider, error) {
	switch strings.ToLower(config.Provider) {
	case "twilio":
		return NewTwilioProvider(s.logger, s.client, config), nil
	case "aliyun":
		return NewAliyunProvider(s.logger, s.client, config), nil
	case "http":
		return NewHTTPProvider(s.logger, s.client, config), nil
	default:
		return nil, fmt.Errorf("unsupported SMS provider: %s", config.Provider)
	}
}

// formatSMSContent formats notification message for SMS (with character limits)
func (s *SMSChannel) formatSMSContent(message *NotificationMessage) string {
	var content string
	
	if message.Alert != nil {
		// Format alert-specific SMS
		alertName := s.getAlertLabel(message.Alert, "alertname", "Unknown")
		instance := s.getAlertLabel(message.Alert, "instance", "")
		
		// Use emoji indicators for different severity levels
		emoji := s.getSeverityEmoji(message.Alert.Severity)
		
		content = fmt.Sprintf("%s [%s] %s", emoji, message.Alert.Severity, alertName)
		
		if instance != "" {
			content += fmt.Sprintf(" on %s", instance)
		}
		
		content += fmt.Sprintf(" - %s at %s", message.Alert.Status, message.Alert.StartsAt.Format("15:04"))
		
		// Add description if short enough
		if description := s.getAlertAnnotation(message.Alert, "description", ""); description != "" && len(description) <= 50 {
			content += fmt.Sprintf(": %s", description)
		}
	} else {
		// Format generic notification
		emoji := s.getLevelEmoji(message.Level)
		content = fmt.Sprintf("%s [%s] %s", emoji, strings.ToUpper(message.Level), message.Title)
		
		if message.Content != "" && len(message.Content) <= 50 {
			content += fmt.Sprintf(": %s", message.Content)
		}
	}

	// Ensure SMS fits in single message (160 chars for GSM, 70 for Unicode)
	// Use 150 as safe limit to account for potential Unicode characters
	if len(content) > 150 {
		content = content[:147] + "..."
	}

	return content
}

// getSeverityEmoji returns emoji for alert severity
func (s *SMSChannel) getSeverityEmoji(severity string) string {
	switch strings.ToLower(severity) {
	case "critical":
		return "🔥" // Fire
	case "warning":
		return "⚠️"  // Warning
	case "info":
		return "ℹ️"  // Information
	default:
		return "🔔" // Bell
	}
}

// getLevelEmoji returns emoji for notification level
func (s *SMSChannel) getLevelEmoji(level string) string {
	switch strings.ToLower(level) {
	case "error", "critical":
		return "❌" // Cross mark
	case "warning":
		return "⚠️" // Warning
	case "info":
		return "ℹ️" // Information
	default:
		return "🔔" // Bell
	}
}

// getAlertLabel gets a label value from alert with fallback
func (s *SMSChannel) getAlertLabel(alert *models.Alert, key, fallback string) string {
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
func (s *SMSChannel) getAlertAnnotation(alert *models.Alert, key, fallback string) string {
	if alert.Annotations != nil {
		if value, exists := alert.Annotations[key]; exists {
			if str, ok := value.(string); ok {
				return str
			}
		}
	}
	return fallback
}

// TwilioProvider implements SMS sending via Twilio API
type TwilioProvider struct {
	logger     *logrus.Logger
	client     *http.Client
	accountSID string
	authToken  string
	fromNumber string
}

func NewTwilioProvider(logger *logrus.Logger, client *http.Client, config *SMSConfig) *TwilioProvider {
	return &TwilioProvider{
		logger:     logger,
		client:     client,
		accountSID: config.TwilioAccountSID,
		authToken:  config.TwilioAuthToken,
		fromNumber: config.TwilioFromNumber,
	}
}

func (t *TwilioProvider) GetType() string {
	return "twilio"
}

func (t *TwilioProvider) SendSMS(ctx context.Context, phoneNumber, message string) error {
	// Twilio REST API endpoint
	apiURL := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", t.accountSID)
	
	// Prepare form data
	data := url.Values{}
	data.Set("To", phoneNumber)
	data.Set("From", t.fromNumber)
	data.Set("Body", message)
	
	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	// Set headers
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(t.accountSID, t.authToken)
	
	// Send request
	resp, err := t.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}
	
	// Check status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		t.logger.WithFields(logrus.Fields{
			"status_code": resp.StatusCode,
			"response":    string(respBody),
		}).Error("Twilio API error")
		return fmt.Errorf("Twilio API error (status %d): %s", resp.StatusCode, string(respBody))
	}
	
	t.logger.WithFields(logrus.Fields{
		"phone_number": phoneNumber,
		"from_number":  t.fromNumber,
	}).Debug("SMS sent via Twilio successfully")
	
	return nil
}

// AliyunProvider implements SMS sending via Aliyun SMS API
type AliyunProvider struct {
	logger          *logrus.Logger
	client          *http.Client
	accessKeyID     string
	accessKeySecret string
	signName        string
	templateCode    string
}

func NewAliyunProvider(logger *logrus.Logger, client *http.Client, config *SMSConfig) *AliyunProvider {
	return &AliyunProvider{
		logger:          logger,
		client:          client,
		accessKeyID:     config.AliyunAccessKeyID,
		accessKeySecret: config.AliyunAccessKeySecret,
		signName:        config.AliyunSignName,
		templateCode:    config.AliyunTemplateCode,
	}
}

func (a *AliyunProvider) GetType() string {
	return "aliyun"
}

func (a *AliyunProvider) SendSMS(ctx context.Context, phoneNumber, message string) error {
	// Aliyun SMS API endpoint
	apiURL := "https://dysmsapi.aliyuncs.com/"
	
	// Prepare parameters
	params := map[string]string{
		"Action":        "SendSms",
		"Version":       "2017-05-25",
		"RegionId":      "cn-hangzhou",
		"PhoneNumbers":  phoneNumber,
		"SignName":      a.signName,
		"TemplateCode":  a.templateCode,
		"TemplateParam": fmt.Sprintf(`{"message":"%s"}`, message),
		"Format":        "JSON",
		"Timestamp":     time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		"SignatureMethod": "HMAC-SHA1",
		"SignatureVersion": "1.0",
		"SignatureNonce": fmt.Sprintf("%d", time.Now().UnixNano()),
		"AccessKeyId":   a.accessKeyID,
	}
	
	// Generate signature
	signature := a.generateSignature("POST", params)
	params["Signature"] = signature
	
	// Prepare form data
	data := url.Values{}
	for k, v := range params {
		data.Set(k, v)
	}
	
	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	// Set headers
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	
	// Send request
	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}
	
	// Parse JSON response
	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}
	
	// Check for errors
	if code, ok := result["Code"].(string); ok && code != "OK" {
		messageStr, _ := result["Message"].(string)
		a.logger.WithFields(logrus.Fields{
			"error_code": code,
			"message":    messageStr,
		}).Error("Aliyun SMS API error")
		return fmt.Errorf("Aliyun SMS API error (code %s): %s", code, messageStr)
	}
	
	a.logger.WithFields(logrus.Fields{
		"phone_number": phoneNumber,
		"sign_name":    a.signName,
	}).Debug("SMS sent via Aliyun successfully")
	
	return nil
}

// generateSignature generates Aliyun API signature
func (a *AliyunProvider) generateSignature(method string, params map[string]string) string {
	// Sort parameters
	keys := make([]string, 0, len(params))
	for k := range params {
		if k != "Signature" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	
	// Build canonical query string
	var canonicalQuery strings.Builder
	for i, key := range keys {
		if i > 0 {
			canonicalQuery.WriteString("&")
		}
		canonicalQuery.WriteString(url.QueryEscape(key))
		canonicalQuery.WriteString("=")
		canonicalQuery.WriteString(url.QueryEscape(params[key]))
	}
	
	// Build string to sign
	stringToSign := method + "&" + url.QueryEscape("/") + "&" + url.QueryEscape(canonicalQuery.String())
	
	// Generate HMAC-SHA1 signature
	h := hmac.New(sha256.New, []byte(a.accessKeySecret+"&"))
	h.Write([]byte(stringToSign))
	return hex.EncodeToString(h.Sum(nil))
}

// HTTPProvider implements generic HTTP SMS provider
type HTTPProvider struct {
	logger  *logrus.Logger
	client  *http.Client
	url     string
	method  string
	headers map[string]string
	params  map[string]string
}

func NewHTTPProvider(logger *logrus.Logger, client *http.Client, config *SMSConfig) *HTTPProvider {
	return &HTTPProvider{
		logger:  logger,
		client:  client,
		url:     config.HTTPURL,
		method:  config.HTTPMethod,
		headers: config.HTTPHeaders,
		params:  config.HTTPParams,
	}
}

func (h *HTTPProvider) GetType() string {
	return "http"
}

func (h *HTTPProvider) SendSMS(ctx context.Context, phoneNumber, message string) error {
	// Replace placeholders in parameters
	data := url.Values{}
	for key, value := range h.params {
		// Replace common placeholders
		value = strings.ReplaceAll(value, "{{phone_number}}", phoneNumber)
		value = strings.ReplaceAll(value, "{{message}}", message)
		data.Set(key, value)
	}
	
	// Create request
	var req *http.Request
	var err error
	
	if strings.ToUpper(h.method) == "GET" {
		// For GET requests, add parameters to URL
		u, _ := url.Parse(h.url)
		u.RawQuery = data.Encode()
		req, err = http.NewRequestWithContext(ctx, h.method, u.String(), nil)
	} else {
		// For POST/PUT requests, add parameters to body
		req, err = http.NewRequestWithContext(ctx, h.method, h.url, strings.NewReader(data.Encode()))
		if err == nil {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
	}
	
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	// Set custom headers
	for key, value := range h.headers {
		// Replace placeholders in headers
		value = strings.ReplaceAll(value, "{{phone_number}}", phoneNumber)
		value = strings.ReplaceAll(value, "{{message}}", message)
		req.Header.Set(key, value)
	}
	
	// Send request
	resp, err := h.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}
	
	// Check status code
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		h.logger.WithFields(logrus.Fields{
			"status_code": resp.StatusCode,
			"response":    string(respBody),
		}).Error("HTTP SMS provider error")
		return fmt.Errorf("HTTP SMS provider error (status %d): %s", resp.StatusCode, string(respBody))
	}
	
	h.logger.WithFields(logrus.Fields{
		"phone_number": phoneNumber,
		"provider_url": h.url,
	}).Debug("SMS sent via HTTP provider successfully")
	
	return nil
}