package service

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"alertbot/internal/models"
	"alertbot/internal/repository"

	"github.com/sirupsen/logrus"
)

type AlertGroupService interface {
	// Group management
	ListAlertGroups(ctx context.Context, filters *models.AlertFilters) ([]*models.AlertGroup, error)
	GetAlertGroup(ctx context.Context, id uint) (*models.AlertGroup, error)
	
	// Group rules management
	ListAlertGroupRules(ctx context.Context) ([]*models.AlertGroupRule, error)
	GetAlertGroupRule(ctx context.Context, id uint) (*models.AlertGroupRule, error)
	CreateAlertGroupRule(ctx context.Context, rule *models.AlertGroupRule) error
	UpdateAlertGroupRule(ctx context.Context, id uint, data *models.AlertGroupRule) error
	DeleteAlertGroupRule(ctx context.Context, id uint) error
	
	// Alert grouping logic
	ProcessAlertForGrouping(ctx context.Context, alert *models.Alert) (*models.AlertGroup, error)
	UpdateGroupFromAlert(ctx context.Context, group *models.AlertGroup, alert *models.Alert) error
}

type alertGroupService struct {
	groupRepo repository.AlertGroupRepository
	alertRepo repository.AlertRepository
	logger    *logrus.Logger
}

func NewAlertGroupService(
	groupRepo repository.AlertGroupRepository, 
	alertRepo repository.AlertRepository,
	logger *logrus.Logger,
) AlertGroupService {
	return &alertGroupService{
		groupRepo: groupRepo,
		alertRepo: alertRepo,
		logger:    logger,
	}
}

func (s *alertGroupService) ListAlertGroups(ctx context.Context, filters *models.AlertFilters) ([]*models.AlertGroup, error) {
	return s.groupRepo.ListAlertGroups(ctx, filters)
}

func (s *alertGroupService) GetAlertGroup(ctx context.Context, id uint) (*models.AlertGroup, error) {
	return s.groupRepo.GetAlertGroup(ctx, id)
}

func (s *alertGroupService) ListAlertGroupRules(ctx context.Context) ([]*models.AlertGroupRule, error) {
	return s.groupRepo.ListAlertGroupRules(ctx)
}

func (s *alertGroupService) GetAlertGroupRule(ctx context.Context, id uint) (*models.AlertGroupRule, error) {
	return s.groupRepo.GetAlertGroupRule(ctx, id)
}

func (s *alertGroupService) CreateAlertGroupRule(ctx context.Context, rule *models.AlertGroupRule) error {
	// Validate the rule
	if err := s.validateGroupRule(rule); err != nil {
		return fmt.Errorf("invalid group rule: %w", err)
	}
	
	return s.groupRepo.CreateAlertGroupRule(ctx, rule)
}

func (s *alertGroupService) UpdateAlertGroupRule(ctx context.Context, id uint, data *models.AlertGroupRule) error {
	existingRule, err := s.groupRepo.GetAlertGroupRule(ctx, id)
	if err != nil {
		return fmt.Errorf("rule not found: %w", err)
	}
	
	// Update fields
	existingRule.Name = data.Name
	existingRule.Description = data.Description
	existingRule.GroupBy = data.GroupBy
	existingRule.GroupWait = data.GroupWait
	existingRule.GroupInterval = data.GroupInterval
	existingRule.RepeatInterval = data.RepeatInterval
	existingRule.Matchers = data.Matchers
	existingRule.Priority = data.Priority
	existingRule.Enabled = data.Enabled
	
	// Validate the updated rule
	if err := s.validateGroupRule(existingRule); err != nil {
		return fmt.Errorf("invalid group rule: %w", err)
	}
	
	return s.groupRepo.UpdateAlertGroupRule(ctx, existingRule)
}

func (s *alertGroupService) DeleteAlertGroupRule(ctx context.Context, id uint) error {
	return s.groupRepo.DeleteAlertGroupRule(ctx, id)
}

func (s *alertGroupService) ProcessAlertForGrouping(ctx context.Context, alert *models.Alert) (*models.AlertGroup, error) {
	// Get active group rules
	rules, err := s.groupRepo.GetActiveAlertGroupRules(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get group rules: %w", err)
	}
	
	// Find the first matching rule (rules are ordered by priority)
	var matchingRule *models.AlertGroupRule
	for _, rule := range rules {
		if s.alertMatchesRule(alert, rule) {
			matchingRule = rule
			break
		}
	}
	
	// If no rule matches, create a default group by alertname
	if matchingRule == nil {
		matchingRule = s.getDefaultGroupRule()
	}
	
	// Generate group key based on the rule
	groupKey := s.generateGroupKey(alert, matchingRule)
	
	// Try to find existing group
	existingGroup, err := s.groupRepo.GetAlertGroupByKey(ctx, groupKey)
	if err == nil {
		// Update existing group
		err = s.UpdateGroupFromAlert(ctx, existingGroup, alert)
		return existingGroup, err
	}
	
	// Create new group
	group := &models.AlertGroup{
		GroupKey:     groupKey,
		GroupBy:      matchingRule.GroupBy,
		CommonLabels: s.extractCommonLabels(alert, matchingRule),
		AlertCount:   1,
		Status:       alert.Status,
		Severity:     alert.Severity,
		FirstAlertAt: alert.StartsAt,
		LastAlertAt:  alert.StartsAt,
	}
	
	err = s.groupRepo.CreateAlertGroup(ctx, group)
	if err != nil {
		return nil, fmt.Errorf("failed to create alert group: %w", err)
	}
	
	return group, nil
}

func (s *alertGroupService) UpdateGroupFromAlert(ctx context.Context, group *models.AlertGroup, alert *models.Alert) error {
	// Update group statistics
	group.AlertCount++
	group.LastAlertAt = alert.StartsAt
	
	// Update severity to highest
	if s.severityLevel(alert.Severity) > s.severityLevel(group.Severity) {
		group.Severity = alert.Severity
	}
	
	// Update status - if any alert is firing, group is firing
	if alert.Status == "firing" {
		group.Status = "firing"
	}
	
	return s.groupRepo.UpdateAlertGroup(ctx, group)
}

// Helper methods

func (s *alertGroupService) validateGroupRule(rule *models.AlertGroupRule) error {
	if rule.Name == "" {
		return fmt.Errorf("rule name cannot be empty")
	}
	
	if rule.GroupBy == nil || len(rule.GroupBy) == 0 {
		return fmt.Errorf("group_by cannot be empty")
	}
	
	if rule.GroupWait < 0 || rule.GroupWait > 3600 {
		return fmt.Errorf("group_wait must be between 0 and 3600 seconds")
	}
	
	if rule.GroupInterval < 60 || rule.GroupInterval > 86400 {
		return fmt.Errorf("group_interval must be between 60 and 86400 seconds")
	}
	
	if rule.RepeatInterval < 300 || rule.RepeatInterval > 604800 {
		return fmt.Errorf("repeat_interval must be between 300 and 604800 seconds")
	}
	
	return nil
}

func (s *alertGroupService) alertMatchesRule(alert *models.Alert, rule *models.AlertGroupRule) bool {
	// If rule has no matchers, it matches all alerts
	if rule.Matchers == nil || len(rule.Matchers) == 0 {
		return true
	}
	
	// Check if alert labels match rule matchers
	matchers, ok := rule.Matchers["matchers"].([]interface{})
	if !ok {
		return true // If matchers format is invalid, match all
	}
	
	alertLabels := make(map[string]string)
	if labelData, err := json.Marshal(alert.Labels); err == nil {
		json.Unmarshal(labelData, &alertLabels)
	}
	
	for _, matcherInterface := range matchers {
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
		
		if isRegex {
			// For now, simple contains check - in production, use proper regex
			if !strings.Contains(alertValue, value) {
				return false
			}
		} else {
			if alertValue != value {
				return false
			}
		}
	}
	
	return true
}

func (s *alertGroupService) generateGroupKey(alert *models.Alert, rule *models.AlertGroupRule) string {
	// Extract grouping labels from the rule
	groupByLabels, ok := rule.GroupBy["labels"].([]interface{})
	if !ok {
		groupByLabels = []interface{}{"alertname"} // Default to alertname
	}
	
	// Build sorted key from alert labels
	var keyParts []string
	alertLabels := make(map[string]string)
	if labelData, err := json.Marshal(alert.Labels); err == nil {
		json.Unmarshal(labelData, &alertLabels)
	}
	
	for _, labelInterface := range groupByLabels {
		label, ok := labelInterface.(string)
		if !ok {
			continue
		}
		
		if value, exists := alertLabels[label]; exists {
			keyParts = append(keyParts, fmt.Sprintf("%s=%s", label, value))
		}
	}
	
	sort.Strings(keyParts)
	keyString := strings.Join(keyParts, ",")
	
	// Hash the key for consistent length
	hash := sha256.Sum256([]byte(keyString))
	return fmt.Sprintf("group-%x", hash[:8]) // Use first 8 bytes of hash
}

func (s *alertGroupService) extractCommonLabels(alert *models.Alert, rule *models.AlertGroupRule) models.JSONB {
	alertLabels := make(map[string]string)
	if labelData, err := json.Marshal(alert.Labels); err == nil {
		json.Unmarshal(labelData, &alertLabels)
	}
	
	commonLabels := make(models.JSONB)
	groupByLabels, ok := rule.GroupBy["labels"].([]interface{})
	if !ok {
		return commonLabels
	}
	
	for _, labelInterface := range groupByLabels {
		label, ok := labelInterface.(string)
		if !ok {
			continue
		}
		
		if value, exists := alertLabels[label]; exists {
			commonLabels[label] = value
		}
	}
	
	return commonLabels
}

func (s *alertGroupService) getDefaultGroupRule() *models.AlertGroupRule {
	return &models.AlertGroupRule{
		Name: "default",
		GroupBy: models.JSONB{
			"labels": []string{"alertname"},
		},
		GroupWait:      10,
		GroupInterval:  300,
		RepeatInterval: 3600,
	}
}

func (s *alertGroupService) severityLevel(severity string) int {
	switch severity {
	case "critical":
		return 3
	case "warning":
		return 2
	case "info":
		return 1
	default:
		return 0
	}
}