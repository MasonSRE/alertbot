package service

import (
	"alertbot/internal/models"
	"context"
)

type AlertService interface {
	ReceiveAlerts(ctx context.Context, alerts []models.PrometheusAlert) error
	GetAlert(ctx context.Context, fingerprint string) (*models.Alert, error)
	ListAlerts(ctx context.Context, filters models.AlertFilters) ([]models.Alert, int64, error)
	SilenceAlert(ctx context.Context, fingerprint string, duration string, comment string) error
	AcknowledgeAlert(ctx context.Context, fingerprint string, comment string) error
	ResolveAlert(ctx context.Context, fingerprint string, comment string) error
	BatchSilenceAlerts(ctx context.Context, fingerprints []string, duration string, comment string) error
	BatchAcknowledgeAlerts(ctx context.Context, fingerprints []string, comment string) error
	BatchResolveAlerts(ctx context.Context, fingerprints []string, comment string) error
	GetAlertHistory(ctx context.Context, fingerprint string) ([]models.AlertHistory, error)
	ListAlertHistory(ctx context.Context, filters models.AlertHistoryFilters) ([]models.AlertHistory, int64, error)
	GetAlertRelations(ctx context.Context, fingerprint string) (*models.AlertRelations, error)
	UpdateDeduplicationConfig(ctx context.Context, config models.DeduplicationConfig) error
}

type RoutingRuleService interface {
	CreateRule(ctx context.Context, rule *models.RoutingRule) error
	GetRule(ctx context.Context, id uint) (*models.RoutingRule, error)
	ListRules(ctx context.Context) ([]models.RoutingRule, error)
	UpdateRule(ctx context.Context, rule *models.RoutingRule) error
	DeleteRule(ctx context.Context, id uint) error
	TestRule(ctx context.Context, conditions map[string]interface{}, sampleAlert models.Alert) (bool, []models.RoutingRule, error)
}

type NotificationChannelService interface {
	CreateChannel(ctx context.Context, channel *models.NotificationChannel) error
	GetChannel(ctx context.Context, id uint) (*models.NotificationChannel, error)
	ListChannels(ctx context.Context) ([]models.NotificationChannel, error)
	UpdateChannel(ctx context.Context, channel *models.NotificationChannel) error
	DeleteChannel(ctx context.Context, id uint) error
	TestChannel(ctx context.Context, id uint, message string) error
}

type SilenceService interface {
	CreateSilence(ctx context.Context, silence *models.Silence) error
	GetSilence(ctx context.Context, id uint) (*models.Silence, error)
	ListSilences(ctx context.Context) ([]models.Silence, error)
	DeleteSilence(ctx context.Context, id uint) error
}

type StatsService interface {
	GetAlertStats(ctx context.Context, startTime, endTime string, groupBy string) (*models.Stats, error)
	GetNotificationStats(ctx context.Context, startTime, endTime string) (interface{}, error)
}

type SettingsService interface {
	GetSystemConfig() (*models.SystemConfig, error)
	UpdateSystemConfig(config *models.SystemConfig) error
	
	GetPrometheusConfig() (*models.PrometheusConfig, error)
	UpdatePrometheusConfig(config *models.PrometheusConfig) error
	
	GetNotificationConfig() (*models.NotificationConfig, error)
	UpdateNotificationConfig(config *models.NotificationConfig) error
}