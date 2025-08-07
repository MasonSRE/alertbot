package service

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"time"

	"alertbot/internal/models"
	"alertbot/internal/repository"

	"github.com/sirupsen/logrus"
)

type InhibitionService interface {
	// Inhibition Rules management
	ListInhibitionRules(ctx context.Context) ([]models.InhibitionRule, error)
	GetInhibitionRule(ctx context.Context, id uint) (*models.InhibitionRule, error)
	CreateInhibitionRule(ctx context.Context, rule *models.InhibitionRule) error
	UpdateInhibitionRule(ctx context.Context, id uint, data *models.InhibitionRule) error
	DeleteInhibitionRule(ctx context.Context, id uint) error

	// Testing
	TestInhibitionRule(ctx context.Context, rule *models.InhibitionRule, sourceAlert, targetAlert map[string]string) (bool, error)
}

type inhibitionService struct {
	inhibitionRepo repository.InhibitionRepository
	alertRepo      repository.AlertRepository
	logger         *logrus.Logger
}

func NewInhibitionService(
	inhibitionRepo repository.InhibitionRepository,
	alertRepo repository.AlertRepository,
	logger *logrus.Logger,
) InhibitionService {
	return &inhibitionService{
		inhibitionRepo: inhibitionRepo,
		alertRepo:      alertRepo,
		logger:         logger,
	}
}

func (s *inhibitionService) ListInhibitionRules(ctx context.Context) ([]models.InhibitionRule, error) {
	return s.inhibitionRepo.List()
}

func (s *inhibitionService) GetInhibitionRule(ctx context.Context, id uint) (*models.InhibitionRule, error) {
	return s.inhibitionRepo.GetByID(id)
}

func (s *inhibitionService) CreateInhibitionRule(ctx context.Context, rule *models.InhibitionRule) error {
	// Validate the rule
	if err := s.validateInhibitionRule(rule); err != nil {
		return fmt.Errorf("invalid inhibition rule: %w", err)
	}

	return s.inhibitionRepo.Create(rule)
}

func (s *inhibitionService) UpdateInhibitionRule(ctx context.Context, id uint, data *models.InhibitionRule) error {
	existingRule, err := s.inhibitionRepo.GetByID(id)
	if err != nil {
		return fmt.Errorf("rule not found: %w", err)
	}

	// Update fields
	existingRule.Name = data.Name
	existingRule.Description = data.Description
	existingRule.SourceMatchers = data.SourceMatchers
	existingRule.TargetMatchers = data.TargetMatchers
	existingRule.EqualLabels = data.EqualLabels
	existingRule.Duration = data.Duration
	existingRule.Priority = data.Priority
	existingRule.Enabled = data.Enabled

	// Validate the updated rule
	if err := s.validateInhibitionRule(existingRule); err != nil {
		return fmt.Errorf("invalid inhibition rule: %w", err)
	}

	return s.inhibitionRepo.Update(existingRule)
}

func (s *inhibitionService) DeleteInhibitionRule(ctx context.Context, id uint) error {
	return s.inhibitionRepo.Delete(id)
}

func (s *inhibitionService) ProcessAlertForInhibition(ctx context.Context, alert *models.Alert) error {
	// Skip if alert is not firing
	if alert.Status != "firing" {
		return nil
	}

	// Get active inhibition rules
	rules, err := s.inhibitionRepo.GetActiveInhibitionRules(ctx)
	if err != nil {
		return fmt.Errorf("failed to get inhibition rules: %w", err)
	}

	alertLabels := s.extractLabels(alert)

	for _, rule := range rules {
		// Check if this alert matches source matchers (can inhibit others)
		if s.alertMatchesMatchers(alertLabels, rule.SourceMatchers) {
			if err := s.applyInhibitionFromSource(ctx, alert, rule); err != nil {
				s.logger.WithError(err).WithFields(logrus.Fields{
					"alert_fingerprint": alert.Fingerprint,
					"rule_id":           rule.ID,
					"rule_name":         rule.Name,
				}).Error("Failed to apply inhibition from source alert")
			}
		}
	}

	return nil
}

func (s *inhibitionService) IsAlertInhibited(ctx context.Context, alert *models.Alert) (bool, []*models.InhibitionStatus, error) {
	inhibitions, err := s.inhibitionRepo.GetInhibitionsByTarget(ctx, alert.Fingerprint)
	if err != nil {
		return false, nil, err
	}

	if len(inhibitions) > 0 {
		return true, inhibitions, nil
	}

	return false, nil, nil
}

func (s *inhibitionService) RemoveInhibitionsForAlert(ctx context.Context, alertFingerprint string) error {
	// Remove inhibitions where this alert is the source (alert resolved, so stop inhibiting others)
	inhibitions, err := s.inhibitionRepo.GetInhibitionsBySource(ctx, alertFingerprint)
	if err != nil {
		return err
	}

	for _, inhibition := range inhibitions {
		if err := s.inhibitionRepo.DeleteInhibitionStatus(ctx, inhibition.ID); err != nil {
			s.logger.WithError(err).WithField("inhibition_id", inhibition.ID).Error("Failed to remove inhibition")
		}
	}

	return nil
}

func (s *inhibitionService) CleanupExpiredInhibitions(ctx context.Context) error {
	return s.inhibitionRepo.CleanupExpiredInhibitions(ctx)
}

func (s *inhibitionService) TestInhibitionRule(ctx context.Context, rule *models.InhibitionRule, sourceAlert, targetAlert map[string]string) (bool, error) {
	// Check if source alert matches source matchers
	if !s.alertMatchesMatchers(sourceAlert, rule.SourceMatchers) {
		return false, nil
	}

	// Check if target alert matches target matchers
	if !s.alertMatchesMatchers(targetAlert, rule.TargetMatchers) {
		return false, nil
	}

	// Check equal labels if specified
	if rule.EqualLabels != nil {
		equalLabels, ok := rule.EqualLabels["labels"].([]interface{})
		if ok {
			for _, labelInterface := range equalLabels {
				label, ok := labelInterface.(string)
				if !ok {
					continue
				}

				sourceValue, sourceExists := sourceAlert[label]
				targetValue, targetExists := targetAlert[label]

				if !sourceExists || !targetExists || sourceValue != targetValue {
					return false, nil
				}
			}
		}
	}

	return true, nil
}

// Helper methods

func (s *inhibitionService) validateInhibitionRule(rule *models.InhibitionRule) error {
	if rule.Name == "" {
		return fmt.Errorf("rule name cannot be empty")
	}

	if rule.SourceMatchers == nil || len(rule.SourceMatchers) == 0 {
		return fmt.Errorf("source_matchers cannot be empty")
	}

	if rule.TargetMatchers == nil || len(rule.TargetMatchers) == 0 {
		return fmt.Errorf("target_matchers cannot be empty")
	}

	if rule.Duration < 0 || rule.Duration > 86400 {
		return fmt.Errorf("duration must be between 0 and 86400 seconds")
	}

	// Validate matcher formats
	if err := s.validateMatchers(rule.SourceMatchers); err != nil {
		return fmt.Errorf("invalid source matchers: %w", err)
	}

	if err := s.validateMatchers(rule.TargetMatchers); err != nil {
		return fmt.Errorf("invalid target matchers: %w", err)
	}

	return nil
}

func (s *inhibitionService) validateMatchers(matchers models.JSONB) error {
	matchersArray, ok := matchers["matchers"].([]interface{})
	if !ok {
		return fmt.Errorf("matchers must be an array")
	}

	for _, matcherInterface := range matchersArray {
		matcher, ok := matcherInterface.(map[string]interface{})
		if !ok {
			return fmt.Errorf("each matcher must be an object")
		}

		name, nameOk := matcher["name"].(string)
		value, valueOk := matcher["value"].(string)
		isRegex, _ := matcher["is_regex"].(bool)

		if !nameOk || name == "" {
			return fmt.Errorf("matcher name is required")
		}

		if !valueOk || value == "" {
			return fmt.Errorf("matcher value is required")
		}

		// If it's a regex matcher, validate the regex
		if isRegex {
			if _, err := regexp.Compile(value); err != nil {
				return fmt.Errorf("invalid regex pattern '%s': %v", value, err)
			}
		}
	}

	return nil
}

func (s *inhibitionService) extractLabels(alert *models.Alert) map[string]string {
	labels := make(map[string]string)
	if labelData, err := json.Marshal(alert.Labels); err == nil {
		json.Unmarshal(labelData, &labels)
	}
	return labels
}

func (s *inhibitionService) alertMatchesMatchers(alertLabels map[string]string, matchers models.JSONB) bool {
	matchersArray, ok := matchers["matchers"].([]interface{})
	if !ok {
		return false
	}

	for _, matcherInterface := range matchersArray {
		matcher, ok := matcherInterface.(map[string]interface{})
		if !ok {
			continue
		}

		name, nameOk := matcher["name"].(string)
		value, valueOk := matcher["value"].(string)
		isRegex, _ := matcher["is_regex"].(bool)

		if !nameOk || !valueOk {
			continue
		}

		alertValue, exists := alertLabels[name]
		if !exists {
			return false
		}

		var matched bool
		if isRegex {
			regex, err := regexp.Compile(value)
			if err != nil {
				s.logger.WithError(err).WithField("pattern", value).Error("Invalid regex in inhibition matcher")
				continue
			}
			matched = regex.MatchString(alertValue)
		} else {
			matched = alertValue == value
		}

		if !matched {
			return false
		}
	}

	return true
}

func (s *inhibitionService) applyInhibitionFromSource(ctx context.Context, sourceAlert *models.Alert, rule *models.InhibitionRule) error {
	// Find all alerts that match target matchers
	// For now, we'll get all firing alerts and filter them
	// In production, you'd want a more efficient query
	allAlerts, _, err := s.alertRepo.List(models.AlertFilters{Status: "firing", Size: 1000})
	if err != nil {
		return fmt.Errorf("failed to get alerts for inhibition check: %w", err)
	}

	sourceLabels := s.extractLabels(sourceAlert)

	for _, targetAlert := range allAlerts {
		// Skip self
		if targetAlert.Fingerprint == sourceAlert.Fingerprint {
			continue
		}

		targetLabels := s.extractLabels(&targetAlert)

		// Check if target alert matches target matchers
		if !s.alertMatchesMatchers(targetLabels, rule.TargetMatchers) {
			continue
		}

		// Check equal labels constraint
		if rule.EqualLabels != nil {
			equalLabels, ok := rule.EqualLabels["labels"].([]interface{})
			if ok {
				allEqual := true
				for _, labelInterface := range equalLabels {
					label, ok := labelInterface.(string)
					if !ok {
						continue
					}

					sourceValue, sourceExists := sourceLabels[label]
					targetValue, targetExists := targetLabels[label]

					if !sourceExists || !targetExists || sourceValue != targetValue {
						allEqual = false
						break
					}
				}

				if !allEqual {
					continue
				}
			}
		}

		// Create inhibition status
		inhibition := &models.InhibitionStatus{
			SourceFingerprint: sourceAlert.Fingerprint,
			TargetFingerprint: targetAlert.Fingerprint,
			RuleID:            rule.ID,
			InhibitedAt:       time.Now(),
		}

		// Set expiration if duration is specified
		if rule.Duration > 0 {
			expiresAt := time.Now().Add(time.Duration(rule.Duration) * time.Second)
			inhibition.ExpiresAt = &expiresAt
		}

		if err := s.inhibitionRepo.CreateInhibitionStatus(ctx, inhibition); err != nil {
			s.logger.WithError(err).WithFields(logrus.Fields{
				"source_fingerprint": sourceAlert.Fingerprint,
				"target_fingerprint": targetAlert.Fingerprint,
				"rule_id":            rule.ID,
			}).Error("Failed to create inhibition status")
		}
	}

	return nil
}