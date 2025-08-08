package middleware

import (
	"fmt"
	"net/http"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// ValidationMiddleware provides input validation for API requests
type ValidationMiddleware struct {
	logger *logrus.Logger
}

// NewValidationMiddleware creates a new validation middleware
func NewValidationMiddleware(logger *logrus.Logger) *ValidationMiddleware {
	return &ValidationMiddleware{
		logger: logger,
	}
}

// ValidateJSON validates JSON request body against validation rules
func (v *ValidationMiddleware) ValidateJSON() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip validation for GET requests
		if c.Request.Method == "GET" {
			c.Next()
			return
		}

		// Get the request body data that was bound
		if value, exists := c.Get("requestData"); exists {
			if err := v.validateStruct(value); err != nil {
				v.logger.WithFields(logrus.Fields{
					"path":   c.Request.URL.Path,
					"method": c.Request.Method,
					"error":  err.Error(),
				}).Warn("Request validation failed")

				c.JSON(http.StatusBadRequest, gin.H{
					"success": false,
					"error":   "Validation failed",
					"details": err.Error(),
				})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// ValidateNotificationChannel validates notification channel configuration
func (v *ValidationMiddleware) ValidateNotificationChannel() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method != "POST" && c.Request.Method != "PUT" {
			c.Next()
			return
		}

		var req struct {
			Type   string                 `json:"type" binding:"required"`
			Config map[string]interface{} `json:"config" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "Invalid request format",
				"details": err.Error(),
			})
			c.Abort()
			return
		}

		// Validate channel configuration based on type
		if err := v.validateChannelConfig(req.Type, req.Config); err != nil {
			v.logger.WithFields(logrus.Fields{
				"channel_type": req.Type,
				"error":        err.Error(),
			}).Warn("Channel configuration validation failed")

			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "Channel configuration validation failed",
				"details": err.Error(),
			})
			c.Abort()
			return
		}

		// Store validated data for handlers
		c.Set("validatedChannelData", req)
		c.Next()
	}
}

// ValidateAlertFilters validates alert filtering parameters
func (v *ValidationMiddleware) ValidateAlertFilters() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Validate page and size parameters
		if pageStr := c.Query("page"); pageStr != "" {
			if page, err := strconv.Atoi(pageStr); err != nil || page < 1 {
				c.JSON(http.StatusBadRequest, gin.H{
					"success": false,
					"error":   "Invalid page parameter",
					"details": "Page must be a positive integer",
				})
				c.Abort()
				return
			}
		}

		if sizeStr := c.Query("size"); sizeStr != "" {
			if size, err := strconv.Atoi(sizeStr); err != nil || size < 1 || size > 1000 {
				c.JSON(http.StatusBadRequest, gin.H{
					"success": false,
					"error":   "Invalid size parameter",
					"details": "Size must be between 1 and 1000",
				})
				c.Abort()
				return
			}
		}

		// Validate sort parameter
		if sort := c.Query("sort"); sort != "" {
			validSortFields := []string{"created_at", "updated_at", "severity", "status", "starts_at"}
			isValid := false
			for _, field := range validSortFields {
				if sort == field {
					isValid = true
					break
				}
			}
			if !isValid {
				c.JSON(http.StatusBadRequest, gin.H{
					"success": false,
					"error":   "Invalid sort parameter",
					"details": fmt.Sprintf("Sort must be one of: %s", strings.Join(validSortFields, ", ")),
				})
				c.Abort()
				return
			}
		}

		// Validate order parameter
		if order := c.Query("order"); order != "" {
			if order != "asc" && order != "desc" {
				c.JSON(http.StatusBadRequest, gin.H{
					"success": false,
					"error":   "Invalid order parameter",
					"details": "Order must be 'asc' or 'desc'",
				})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// validateStruct validates a struct using reflection and validation tags
func (v *ValidationMiddleware) validateStruct(data interface{}) error {
	value := reflect.ValueOf(data)
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	}

	if value.Kind() != reflect.Struct {
		return nil // Not a struct, skip validation
	}

	structType := value.Type()
	for i := 0; i < value.NumField(); i++ {
		field := value.Field(i)
		fieldType := structType.Field(i)

		// Get validation tag
		validateTag := fieldType.Tag.Get("validate")
		if validateTag == "" {
			continue
		}

		// Parse validation rules
		rules := strings.Split(validateTag, ",")
		for _, rule := range rules {
			if err := v.validateField(field, fieldType.Name, rule); err != nil {
				return err
			}
		}
	}

	return nil
}

// validateField validates a single field against a validation rule
func (v *ValidationMiddleware) validateField(field reflect.Value, fieldName, rule string) error {
	rule = strings.TrimSpace(rule)

	switch {
	case rule == "required":
		if field.Kind() == reflect.String && field.String() == "" {
			return fmt.Errorf("field '%s' is required", fieldName)
		}
		if field.Kind() == reflect.Slice && field.Len() == 0 {
			return fmt.Errorf("field '%s' is required", fieldName)
		}

	case strings.HasPrefix(rule, "min="):
		minStr := strings.TrimPrefix(rule, "min=")
		min, err := strconv.Atoi(minStr)
		if err != nil {
			return fmt.Errorf("invalid min validation rule: %s", rule)
		}

		if field.Kind() == reflect.String && len(field.String()) < min {
			return fmt.Errorf("field '%s' must be at least %d characters", fieldName, min)
		}
		if field.Kind() == reflect.Slice && field.Len() < min {
			return fmt.Errorf("field '%s' must have at least %d items", fieldName, min)
		}
		if field.Kind() == reflect.Int && field.Int() < int64(min) {
			return fmt.Errorf("field '%s' must be at least %d", fieldName, min)
		}

	case strings.HasPrefix(rule, "max="):
		maxStr := strings.TrimPrefix(rule, "max=")
		max, err := strconv.Atoi(maxStr)
		if err != nil {
			return fmt.Errorf("invalid max validation rule: %s", rule)
		}

		if field.Kind() == reflect.String && len(field.String()) > max {
			return fmt.Errorf("field '%s' must be at most %d characters", fieldName, max)
		}
		if field.Kind() == reflect.Slice && field.Len() > max {
			return fmt.Errorf("field '%s' must have at most %d items", fieldName, max)
		}
		if field.Kind() == reflect.Int && field.Int() > int64(max) {
			return fmt.Errorf("field '%s' must be at most %d", fieldName, max)
		}

	case rule == "email":
		if field.Kind() == reflect.String {
			if !isValidEmail(field.String()) {
				return fmt.Errorf("field '%s' must be a valid email address", fieldName)
			}
		}

	case rule == "url":
		if field.Kind() == reflect.String {
			if !isValidURL(field.String()) {
				return fmt.Errorf("field '%s' must be a valid URL", fieldName)
			}
		}

	case rule == "phone":
		if field.Kind() == reflect.String {
			if !isValidPhoneNumber(field.String()) {
				return fmt.Errorf("field '%s' must be a valid phone number", fieldName)
			}
		}
	}

	return nil
}

// validateChannelConfig validates notification channel configuration
func (v *ValidationMiddleware) validateChannelConfig(channelType string, config map[string]interface{}) error {
	switch strings.ToLower(channelType) {
	case "email":
		return v.validateEmailConfig(config)
	case "sms":
		return v.validateSMSConfig(config)
	case "dingtalk":
		return v.validateDingTalkConfig(config)
	case "wechat_work":
		return v.validateWeChatWorkConfig(config)
	case "telegram":
		return v.validateTelegramConfig(config)
	default:
		return fmt.Errorf("unsupported channel type: %s", channelType)
	}
}

// validateEmailConfig validates email channel configuration
func (v *ValidationMiddleware) validateEmailConfig(config map[string]interface{}) error {
	required := []string{"smtp_host", "username", "password", "from", "to"}
	for _, field := range required {
		if val, ok := config[field]; !ok || val == "" {
			return fmt.Errorf("field '%s' is required for email channel", field)
		}
	}

	// Validate email addresses
	if from, ok := config["from"].(string); ok {
		if !isValidEmail(from) {
			return fmt.Errorf("invalid from email address")
		}
	}

	// Validate to addresses
	if to, ok := config["to"]; ok {
		emails, err := convertToStringSlice(to)
		if err != nil {
			return fmt.Errorf("invalid to addresses format")
		}
		for _, email := range emails {
			if !isValidEmail(email) {
				return fmt.Errorf("invalid to email address: %s", email)
			}
		}
	}

	return nil
}

// validateSMSConfig validates SMS channel configuration
func (v *ValidationMiddleware) validateSMSConfig(config map[string]interface{}) error {
	provider, ok := config["provider"].(string)
	if !ok || provider == "" {
		return fmt.Errorf("provider is required for SMS channel")
	}

	phoneNumbers, ok := config["phone_numbers"]
	if !ok {
		return fmt.Errorf("phone_numbers is required for SMS channel")
	}

	phones, err := convertToStringSlice(phoneNumbers)
	if err != nil {
		return fmt.Errorf("invalid phone_numbers format")
	}

	for _, phone := range phones {
		if !isValidPhoneNumber(phone) {
			return fmt.Errorf("invalid phone number: %s", phone)
		}
	}

	// Validate provider-specific configuration
	switch strings.ToLower(provider) {
	case "twilio":
		required := []string{"twilio_account_sid", "twilio_auth_token", "twilio_from_number"}
		for _, field := range required {
			if val, ok := config[field]; !ok || val == "" {
				return fmt.Errorf("field '%s' is required for Twilio SMS", field)
			}
		}
	case "aliyun":
		required := []string{"aliyun_access_key_id", "aliyun_access_key_secret", "aliyun_sign_name", "aliyun_template_code"}
		for _, field := range required {
			if val, ok := config[field]; !ok || val == "" {
				return fmt.Errorf("field '%s' is required for Aliyun SMS", field)
			}
		}
	case "http":
		if val, ok := config["http_url"]; !ok || val == "" {
			return fmt.Errorf("http_url is required for HTTP SMS provider")
		}
	}

	return nil
}

// validateDingTalkConfig validates DingTalk channel configuration
func (v *ValidationMiddleware) validateDingTalkConfig(config map[string]interface{}) error {
	if val, ok := config["webhook_url"]; !ok || val == "" {
		return fmt.Errorf("webhook_url is required for DingTalk channel")
	}

	if webhookURL, ok := config["webhook_url"].(string); ok {
		if !isValidURL(webhookURL) {
			return fmt.Errorf("invalid webhook_url for DingTalk channel")
		}
	}

	return nil
}

// validateWeChatWorkConfig validates WeChat Work channel configuration
func (v *ValidationMiddleware) validateWeChatWorkConfig(config map[string]interface{}) error {
	if val, ok := config["webhook_url"]; !ok || val == "" {
		return fmt.Errorf("webhook_url is required for WeChat Work channel")
	}

	if webhookURL, ok := config["webhook_url"].(string); ok {
		if !isValidURL(webhookURL) {
			return fmt.Errorf("invalid webhook_url for WeChat Work channel")
		}
	}

	return nil
}

// validateTelegramConfig validates Telegram channel configuration
func (v *ValidationMiddleware) validateTelegramConfig(config map[string]interface{}) error {
	required := []string{"bot_token", "chat_id"}
	for _, field := range required {
		if val, ok := config[field]; !ok || val == "" {
			return fmt.Errorf("field '%s' is required for Telegram channel", field)
		}
	}

	// Validate bot token format
	if botToken, ok := config["bot_token"].(string); ok {
		if !strings.Contains(botToken, ":") {
			return fmt.Errorf("invalid bot_token format for Telegram channel")
		}
	}

	return nil
}

// Utility functions for validation

func isValidEmail(email string) bool {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return len(email) <= 254 && emailRegex.MatchString(email)
}

func isValidURL(url string) bool {
	urlRegex := regexp.MustCompile(`^https?://[a-zA-Z0-9.-]+(\.[a-zA-Z]{2,})(:[0-9]+)?(/.*)?$`)
	return urlRegex.MatchString(url)
}

func isValidPhoneNumber(phone string) bool {
	phoneRegex := regexp.MustCompile(`^\+?[1-9]\d{1,14}$`)
	return len(phone) >= 8 && len(phone) <= 16 && phoneRegex.MatchString(phone)
}

func convertToStringSlice(value interface{}) ([]string, error) {
	switch v := value.(type) {
	case []interface{}:
		var result []string
		for _, item := range v {
			if str, ok := item.(string); ok {
				result = append(result, str)
			} else {
				return nil, fmt.Errorf("invalid item type in slice")
			}
		}
		return result, nil
	case []string:
		return v, nil
	case string:
		return []string{v}, nil
	default:
		return nil, fmt.Errorf("invalid type for string slice")
	}
}