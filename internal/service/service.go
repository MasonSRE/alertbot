package service

import (
	"alertbot/internal/config"
	"alertbot/internal/engine"
	"alertbot/internal/notification"
	"alertbot/internal/repository"
	"alertbot/internal/websocket"

	"github.com/sirupsen/logrus"
)

type Services struct {
	Alert            AlertService
	RoutingRule      RoutingRuleService
	NotificationChannel NotificationChannelService
	Silence          SilenceService
	Stats            StatsService
	AlertGroup       AlertGroupService
	Inhibition       InhibitionService
}

type ServiceDependencies struct {
	Repositories        *repository.Repositories
	Logger              *logrus.Logger
	Config              *config.Config
	RuleEngine          *engine.RuleEngine
	NotificationManager *notification.NotificationManager
	WebSocketHub        *websocket.Hub
}

func NewServices(deps ServiceDependencies) *Services {
	// Initialize rule engine if not provided
	if deps.RuleEngine == nil {
		deps.RuleEngine = engine.NewRuleEngine(deps.Repositories, deps.Logger)
	}
	
	// Initialize notification manager if not provided
	if deps.NotificationManager == nil {
		deps.NotificationManager = notification.NewNotificationManager(deps.Logger)
	}
	
	return &Services{
		Alert:               NewAlertService(deps),
		RoutingRule:         NewRoutingRuleService(deps),
		NotificationChannel: NewNotificationChannelService(deps),
		Silence:             NewSilenceService(deps),
		Stats:               NewStatsService(deps), // Implemented in stats_service.go
		AlertGroup:          NewAlertGroupService(deps.Repositories.AlertGroup, deps.Repositories.Alert, deps.Logger),
		Inhibition:          NewInhibitionService(deps.Repositories.Inhibition, deps.Repositories.Alert, deps.Logger),
	}
}