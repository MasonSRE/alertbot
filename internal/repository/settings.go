package repository

import (
	"alertbot/internal/models"

	"gorm.io/gorm"
)

type settingsRepository struct {
	db *gorm.DB
}

func NewSettingsRepository(db *gorm.DB) SettingsRepository {
	return &settingsRepository{db: db}
}

// System settings methods
func (r *settingsRepository) GetSystemConfig() (*models.SystemConfig, error) {
	var config models.SystemConfig
	err := r.db.First(&config).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// Return default settings if not found
			return &models.SystemConfig{
				SystemName:          "AlertBot",
				AdminEmail:          "admin@company.com",
				RetentionDays:       30,
				EnableNotifications: true,
				EnableWebhooks:      true,
				WebhookTimeout:      30,
			}, nil
		}
		return nil, err
	}
	return &config, nil
}

func (r *settingsRepository) UpdateSystemConfig(config *models.SystemConfig) error {
	var existingConfig models.SystemConfig
	err := r.db.First(&existingConfig).Error
	
	if err == gorm.ErrRecordNotFound {
		// Create new record
		config.ID = 1 // Ensure single record with ID 1
		return r.db.Create(config).Error
	} else if err != nil {
		return err
	}
	
	// Update existing record
	config.ID = existingConfig.ID
	return r.db.Save(config).Error
}

// Prometheus settings methods
func (r *settingsRepository) GetPrometheusConfig() (*models.PrometheusConfig, error) {
	var config models.PrometheusConfig
	err := r.db.First(&config).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// Return default settings if not found
			return &models.PrometheusConfig{
				Enabled:            true,
				URL:                "http://localhost:9090",
				Timeout:            30,
				QueryTimeout:       30,
				ScrapeInterval:     "15s",
				EvaluationInterval: "15s",
			}, nil
		}
		return nil, err
	}
	return &config, nil
}

func (r *settingsRepository) UpdatePrometheusConfig(config *models.PrometheusConfig) error {
	var existingConfig models.PrometheusConfig
	err := r.db.First(&existingConfig).Error
	
	if err == gorm.ErrRecordNotFound {
		// Create new record
		config.ID = 1 // Ensure single record with ID 1
		return r.db.Create(config).Error
	} else if err != nil {
		return err
	}
	
	// Update existing record
	config.ID = existingConfig.ID
	return r.db.Save(config).Error
}

// Notification settings methods
func (r *settingsRepository) GetNotificationConfig() (*models.NotificationConfig, error) {
	var config models.NotificationConfig
	err := r.db.First(&config).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// Return default settings if not found
			return &models.NotificationConfig{
				MaxRetries:    3,
				RetryInterval: 30,
				RateLimit:     100,
				BatchSize:     10,
			}, nil
		}
		return nil, err
	}
	return &config, nil
}

func (r *settingsRepository) UpdateNotificationConfig(config *models.NotificationConfig) error {
	var existingConfig models.NotificationConfig
	err := r.db.First(&existingConfig).Error
	
	if err == gorm.ErrRecordNotFound {
		// Create new record
		config.ID = 1 // Ensure single record with ID 1
		return r.db.Create(config).Error
	} else if err != nil {
		return err
	}
	
	// Update existing record
	config.ID = existingConfig.ID
	return r.db.Save(config).Error
}