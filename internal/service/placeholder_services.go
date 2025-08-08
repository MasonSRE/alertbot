package service

import (
	"context"
	"fmt"

	"alertbot/internal/models"
)

// 占位符服务实现，后续会实现具体功能

type routingRuleService struct {
	deps ServiceDependencies
}

func NewRoutingRuleService(deps ServiceDependencies) RoutingRuleService {
	return &routingRuleService{deps: deps}
}

func (s *routingRuleService) CreateRule(ctx context.Context, rule *models.RoutingRule) error {
	return s.deps.Repositories.RoutingRule.Create(rule)
}

func (s *routingRuleService) GetRule(ctx context.Context, id uint) (*models.RoutingRule, error) {
	return s.deps.Repositories.RoutingRule.GetByID(id)
}

func (s *routingRuleService) ListRules(ctx context.Context) ([]models.RoutingRule, error) {
	return s.deps.Repositories.RoutingRule.List()
}

func (s *routingRuleService) UpdateRule(ctx context.Context, rule *models.RoutingRule) error {
	return s.deps.Repositories.RoutingRule.Update(rule)
}

func (s *routingRuleService) DeleteRule(ctx context.Context, id uint) error {
	return s.deps.Repositories.RoutingRule.Delete(id)
}

func (s *routingRuleService) TestRule(ctx context.Context, conditions map[string]interface{}, sampleAlert models.Alert) (bool, []models.RoutingRule, error) {
	// Use rule engine for testing
	if s.deps.RuleEngine == nil {
		return false, []models.RoutingRule{}, fmt.Errorf("rule engine not available")
	}
	return s.deps.RuleEngine.TestRule(ctx, conditions, sampleAlert)
}

type notificationChannelService struct {
	deps ServiceDependencies
}

func NewNotificationChannelService(deps ServiceDependencies) NotificationChannelService {
	return &notificationChannelService{deps: deps}
}

func (s *notificationChannelService) CreateChannel(ctx context.Context, channel *models.NotificationChannel) error {
	return s.deps.Repositories.NotificationChannel.Create(channel)
}

func (s *notificationChannelService) GetChannel(ctx context.Context, id uint) (*models.NotificationChannel, error) {
	return s.deps.Repositories.NotificationChannel.GetByID(id)
}

func (s *notificationChannelService) ListChannels(ctx context.Context) ([]models.NotificationChannel, error) {
	return s.deps.Repositories.NotificationChannel.List()
}

func (s *notificationChannelService) UpdateChannel(ctx context.Context, channel *models.NotificationChannel) error {
	return s.deps.Repositories.NotificationChannel.Update(channel)
}

func (s *notificationChannelService) DeleteChannel(ctx context.Context, id uint) error {
	return s.deps.Repositories.NotificationChannel.Delete(id)
}

func (s *notificationChannelService) TestChannel(ctx context.Context, id uint, message string) error {
	// Get channel configuration
	channel, err := s.deps.Repositories.NotificationChannel.GetByID(id)
	if err != nil {
		return fmt.Errorf("failed to get notification channel: %w", err)
	}

	if !channel.Enabled {
		return fmt.Errorf("notification channel is disabled")
	}

	// Test the channel using notification manager
	if s.deps.NotificationManager == nil {
		return fmt.Errorf("notification manager not available")
	}

	// For channels that require configuration (WeChat Work, DingTalk), use TestChannelWithConfig
	channelType := models.NotificationChannelType(channel.Type)
	if channelType == models.ChannelTypeWeChatWork || channelType == models.ChannelTypeDingTalk {
		return s.deps.NotificationManager.TestChannelWithConfig(ctx, channelType, message, channel.Config)
	}

	// For other channel types, use the regular TestChannel method
	return s.deps.NotificationManager.TestChannel(ctx, channelType, message)
}

type silenceService struct {
	deps ServiceDependencies
}

func NewSilenceService(deps ServiceDependencies) SilenceService {
	return &silenceService{deps: deps}
}

func (s *silenceService) CreateSilence(ctx context.Context, silence *models.Silence) error {
	return s.deps.Repositories.Silence.Create(silence)
}

func (s *silenceService) GetSilence(ctx context.Context, id uint) (*models.Silence, error) {
	return s.deps.Repositories.Silence.GetByID(id)
}

func (s *silenceService) ListSilences(ctx context.Context) ([]models.Silence, error) {
	return s.deps.Repositories.Silence.List()
}

func (s *silenceService) DeleteSilence(ctx context.Context, id uint) error {
	return s.deps.Repositories.Silence.Delete(id)
}

// Stats service implementation is now in stats_service.go