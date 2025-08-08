package service

import (
	"alertbot/internal/models"
	"alertbot/internal/repository"
	"net/url"
	"time"
)

type settingsService struct {
	repo repository.SettingsRepository
}

func NewSettingsService(repo repository.SettingsRepository) SettingsService {
	return &settingsService{
		repo: repo,
	}
}

func (s *settingsService) GetSystemConfig() (*models.SystemConfig, error) {
	return s.repo.GetSystemConfig()
}

func (s *settingsService) UpdateSystemConfig(config *models.SystemConfig) error {
	// Validate system config
	if config.SystemName == "" {
		return &ValidationError{Field: "system_name", Message: "System name is required"}
	}
	
	if config.AdminEmail == "" {
		return &ValidationError{Field: "admin_email", Message: "Admin email is required"}
	}
	
	if config.RetentionDays < 1 || config.RetentionDays > 365 {
		return &ValidationError{Field: "retention_days", Message: "Retention days must be between 1 and 365"}
	}
	
	if config.WebhookTimeout < 1 || config.WebhookTimeout > 300 {
		return &ValidationError{Field: "webhook_timeout", Message: "Webhook timeout must be between 1 and 300 seconds"}
	}
	
	return s.repo.UpdateSystemConfig(config)
}

func (s *settingsService) GetPrometheusConfig() (*models.PrometheusConfig, error) {
	return s.repo.GetPrometheusConfig()
}

func (s *settingsService) UpdatePrometheusConfig(config *models.PrometheusConfig) error {
	// Validate Prometheus config
	if config.URL == "" {
		return &ValidationError{Field: "url", Message: "Prometheus URL is required"}
	}
	
	// Validate URL format
	if _, err := url.Parse(config.URL); err != nil {
		return &ValidationError{Field: "url", Message: "Invalid URL format"}
	}
	
	if config.Timeout < 5 || config.Timeout > 300 {
		return &ValidationError{Field: "timeout", Message: "Timeout must be between 5 and 300 seconds"}
	}
	
	if config.QueryTimeout < 5 || config.QueryTimeout > 300 {
		return &ValidationError{Field: "query_timeout", Message: "Query timeout must be between 5 and 300 seconds"}
	}
	
	// Validate interval formats
	if config.ScrapeInterval != "" {
		if _, err := time.ParseDuration(config.ScrapeInterval); err != nil {
			return &ValidationError{Field: "scrape_interval", Message: "Invalid scrape interval format"}
		}
	}
	
	if config.EvaluationInterval != "" {
		if _, err := time.ParseDuration(config.EvaluationInterval); err != nil {
			return &ValidationError{Field: "evaluation_interval", Message: "Invalid evaluation interval format"}
		}
	}
	
	return s.repo.UpdatePrometheusConfig(config)
}

func (s *settingsService) GetNotificationConfig() (*models.NotificationConfig, error) {
	return s.repo.GetNotificationConfig()
}

func (s *settingsService) UpdateNotificationConfig(config *models.NotificationConfig) error {
	// Validate notification config
	if config.MaxRetries < 0 || config.MaxRetries > 10 {
		return &ValidationError{Field: "max_retries", Message: "Max retries must be between 0 and 10"}
	}
	
	if config.RetryInterval < 1 || config.RetryInterval > 3600 {
		return &ValidationError{Field: "retry_interval", Message: "Retry interval must be between 1 and 3600 seconds"}
	}
	
	if config.RateLimit < 1 || config.RateLimit > 1000 {
		return &ValidationError{Field: "rate_limit", Message: "Rate limit must be between 1 and 1000"}
	}
	
	if config.BatchSize < 1 || config.BatchSize > 100 {
		return &ValidationError{Field: "batch_size", Message: "Batch size must be between 1 and 100"}
	}
	
	return s.repo.UpdateNotificationConfig(config)
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}