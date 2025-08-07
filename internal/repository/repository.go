package repository

import (
	"context"

	"alertbot/internal/models"

	"gorm.io/gorm"
)

type Repositories struct {
	Alert               AlertRepository
	RoutingRule         RoutingRuleRepository
	NotificationChannel NotificationChannelRepository
	Silence             SilenceRepository
	AlertHistory        AlertHistoryRepository
	AlertGroup          AlertGroupRepository
	Inhibition          InhibitionRepository
}

type AlertRepository interface {
	Create(alert *models.Alert) error
	GetByFingerprint(fingerprint string) (*models.Alert, error)
	List(filters models.AlertFilters) ([]models.Alert, int64, error)
	Update(alert *models.Alert) error
	Delete(fingerprint string) error
}

type RoutingRuleRepository interface {
	Create(rule *models.RoutingRule) error
	GetByID(id uint) (*models.RoutingRule, error)
	List() ([]models.RoutingRule, error)
	Update(rule *models.RoutingRule) error
	Delete(id uint) error
	GetActiveRulesByPriority() ([]models.RoutingRule, error)
}

type NotificationChannelRepository interface {
	Create(channel *models.NotificationChannel) error
	GetByID(id uint) (*models.NotificationChannel, error)
	List() ([]models.NotificationChannel, error)
	Update(channel *models.NotificationChannel) error
	Delete(id uint) error
	GetActiveChannels() ([]models.NotificationChannel, error)
}

type SilenceRepository interface {
	Create(silence *models.Silence) error
	GetByID(id uint) (*models.Silence, error)
	List() ([]models.Silence, error)
	Delete(id uint) error
	GetActiveSilences() ([]models.Silence, error)
}

type AlertHistoryRepository interface {
	Create(history *models.AlertHistory) error
	GetByAlertFingerprint(fingerprint string) ([]models.AlertHistory, error)
	GetByFingerprint(fingerprint string) ([]models.AlertHistory, error)
	List(filters models.AlertHistoryFilters) ([]models.AlertHistory, int64, error)
}

type InhibitionRepository interface {
	Create(rule *models.InhibitionRule) error
	GetByID(id uint) (*models.InhibitionRule, error)
	List() ([]models.InhibitionRule, error)
	Update(rule *models.InhibitionRule) error
	Delete(id uint) error
	
	// Service methods
	ListInhibitionRules(ctx context.Context) ([]*models.InhibitionRule, error)
	GetInhibitionRule(ctx context.Context, id uint) (*models.InhibitionRule, error)
	CreateInhibitionRule(ctx context.Context, rule *models.InhibitionRule) error
	UpdateInhibitionRule(ctx context.Context, rule *models.InhibitionRule) error
	DeleteInhibitionRule(ctx context.Context, id uint) error
	GetActiveInhibitionRules(ctx context.Context) ([]*models.InhibitionRule, error)
	
	// Inhibition status methods
	CreateInhibitionStatus(ctx context.Context, status *models.InhibitionStatus) error
	DeleteInhibitionStatus(ctx context.Context, id uint) error
	GetInhibitionsByTarget(ctx context.Context, targetFingerprint string) ([]*models.InhibitionStatus, error)
	GetInhibitionsBySource(ctx context.Context, sourceFingerprint string) ([]*models.InhibitionStatus, error)
	CleanupExpiredInhibitions(ctx context.Context) error
	GetActiveInhibitions(ctx context.Context) ([]*models.InhibitionStatus, error)
}

func NewRepositories(db *gorm.DB) *Repositories {
	return &Repositories{
		Alert:               NewAlertRepository(db),
		RoutingRule:         NewRoutingRuleRepository(db),
		NotificationChannel: NewNotificationChannelRepository(db),
		Silence:             NewSilenceRepository(db),
		AlertHistory:        NewAlertHistoryRepository(db),
		AlertGroup:          NewAlertGroupRepository(db),
		Inhibition:          NewInhibitionRepository(db),
	}
}