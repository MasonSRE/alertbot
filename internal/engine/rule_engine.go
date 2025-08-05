package engine

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"alertbot/internal/errors"
	"alertbot/internal/models"
	"alertbot/internal/recovery"
	"alertbot/internal/repository"

	"github.com/sirupsen/logrus"
)

// RuleEngine handles alert routing and matching
type RuleEngine struct {
	repos          *repository.Repositories
	logger         *logrus.Logger
	rules          []models.RoutingRule
	rulesLock      sync.RWMutex
	lastUpdate     time.Time
	circuitBreaker *recovery.CircuitBreaker
	retryConfig    recovery.RetryConfig
}

// Matcher represents a condition matcher
type Matcher struct {
	Name    string `json:"name"`
	Value   string `json:"value"`
	IsRegex bool   `json:"is_regex"`
	Operator string `json:"operator"` // equals, not_equals, contains, not_contains, regex, not_regex
}

// RuleCondition represents rule matching conditions
type RuleCondition struct {
	Matchers []Matcher `json:"matchers"`
	Logic    string    `json:"logic"` // and, or
}

func NewRuleEngine(repos *repository.Repositories, logger *logrus.Logger) *RuleEngine {
	engine := &RuleEngine{
		repos:  repos,
		logger: logger,
		rules:  make([]models.RoutingRule, 0),
		circuitBreaker: recovery.NewCircuitBreaker(recovery.CircuitBreakerConfig{
			Name:         "rule_engine",
			MaxFailures:  3,
			ResetTimeout: 30 * time.Second,
			Logger:       logger,
		}),
		retryConfig: recovery.RetryConfig{
			MaxAttempts:   2,
			InitialDelay:  500 * time.Millisecond,
			MaxDelay:      5 * time.Second,
			BackoffFactor: 2.0,
			Jitter:        true,
			RetryCondition: func(err error) bool {
				return recovery.IsRetryable(err) && !errors.IsValidationError(err)
			},
			Logger: logger,
		},
	}
	
	// Load initial rules with retry
	if err := recovery.Retry(context.Background(), engine.retryConfig, func(ctx context.Context) error {
		return engine.LoadRules()
	}); err != nil {
		logger.WithError(err).Error("Failed to load initial rules after retries")
	}
	
	return engine
}

// LoadRules loads all active rules from database
func (re *RuleEngine) LoadRules() error {
	re.rulesLock.Lock()
	defer re.rulesLock.Unlock()

	rules, err := re.repos.RoutingRule.GetActiveRulesByPriority()
	if err != nil {
		return fmt.Errorf("failed to load rules: %w", err)
	}

	re.rules = rules
	re.lastUpdate = time.Now()
	
	re.logger.WithField("count", len(rules)).Info("Rules loaded successfully")
	return nil
}

// MatchAlert finds matching rules for an alert
func (re *RuleEngine) MatchAlert(ctx context.Context, alert *models.Alert) ([]models.RoutingRule, error) {
	re.rulesLock.RLock()
	defer re.rulesLock.RUnlock()

	var matchedRules []models.RoutingRule

	for _, rule := range re.rules {
		if !rule.Enabled {
			continue
		}

		matched, err := re.evaluateRule(rule, alert)
		if err != nil {
			re.logger.WithError(err).WithFields(logrus.Fields{
				"rule_id": rule.ID,
				"rule_name": rule.Name,
				"alert_fingerprint": alert.Fingerprint,
			}).Warn("Failed to evaluate rule")
			continue
		}

		if matched {
			matchedRules = append(matchedRules, rule)
			re.logger.WithFields(logrus.Fields{
				"rule_id": rule.ID,
				"rule_name": rule.Name,
				"alert_fingerprint": alert.Fingerprint,
			}).Debug("Rule matched alert")
		}
	}

	return matchedRules, nil
}

// evaluateRule checks if a rule matches an alert
func (re *RuleEngine) evaluateRule(rule models.RoutingRule, alert *models.Alert) (bool, error) {
	// Parse conditions from JSONB
	conditions, err := re.parseConditions(rule.Conditions)
	if err != nil {
		return false, fmt.Errorf("failed to parse rule conditions: %w", err)
	}

	return re.evaluateConditions(conditions, alert)
}

// parseConditions parses rule conditions from JSONB
func (re *RuleEngine) parseConditions(conditionsJSON models.JSONB) (*RuleCondition, error) {
	condition := &RuleCondition{
		Logic: "and", // default
	}

	// Handle different JSON structures
	if matchers, ok := conditionsJSON["matchers"].([]interface{}); ok {
		for _, m := range matchers {
			if matcherMap, ok := m.(map[string]interface{}); ok {
				matcher := Matcher{
					Operator: "equals", // default
				}
				
				if name, ok := matcherMap["name"].(string); ok {
					matcher.Name = name
				}
				if value, ok := matcherMap["value"].(string); ok {
					matcher.Value = value
				}
				if isRegex, ok := matcherMap["is_regex"].(bool); ok {
					matcher.IsRegex = isRegex
				}
				if operator, ok := matcherMap["operator"].(string); ok {
					matcher.Operator = operator
				}
				
				condition.Matchers = append(condition.Matchers, matcher)
			}
		}
	} else {
		// Simple key-value conditions
		for key, value := range conditionsJSON {
			if key == "logic" {
				if logic, ok := value.(string); ok {
					condition.Logic = logic
				}
				continue
			}

			matcher := Matcher{
				Name:     key,
				Operator: "equals",
			}

			switch v := value.(type) {
			case string:
				matcher.Value = v
			case []interface{}:
				// Handle array values (e.g., severity: ["warning", "critical"])
				var values []string
				for _, item := range v {
					if str, ok := item.(string); ok {
						values = append(values, str)
					}
				}
				matcher.Value = strings.Join(values, "|")
				matcher.Operator = "in"
			default:
				matcher.Value = fmt.Sprintf("%v", v)
			}

			condition.Matchers = append(condition.Matchers, matcher)
		}
	}

	if logic, ok := conditionsJSON["logic"].(string); ok {
		condition.Logic = logic
	}

	return condition, nil
}

// evaluateConditions evaluates rule conditions against an alert
func (re *RuleEngine) evaluateConditions(condition *RuleCondition, alert *models.Alert) (bool, error) {
	if len(condition.Matchers) == 0 {
		return true, nil // Empty conditions match everything
	}

	results := make([]bool, len(condition.Matchers))
	
	for i, matcher := range condition.Matchers {
		result, err := re.evaluateMatcher(matcher, alert)
		if err != nil {
			return false, err
		}
		results[i] = result
	}

	// Apply logic operator
	switch strings.ToLower(condition.Logic) {
	case "or":
		for _, result := range results {
			if result {
				return true, nil
			}
		}
		return false, nil
	case "and", "":
		for _, result := range results {
			if !result {
				return false, nil
			}
		}
		return true, nil
	default:
		return false, fmt.Errorf("unsupported logic operator: %s", condition.Logic)
	}
}

// evaluateMatcher evaluates a single matcher against an alert
func (re *RuleEngine) evaluateMatcher(matcher Matcher, alert *models.Alert) (bool, error) {
	alertValue := re.getAlertValue(matcher.Name, alert)
	
	switch strings.ToLower(matcher.Operator) {
	case "equals", "eq", "":
		if matcher.IsRegex {
			return re.matchRegex(matcher.Value, alertValue)
		}
		return alertValue == matcher.Value, nil
		
	case "not_equals", "ne":
		if matcher.IsRegex {
			matched, err := re.matchRegex(matcher.Value, alertValue)
			return !matched, err
		}
		return alertValue != matcher.Value, nil
		
	case "contains":
		return strings.Contains(alertValue, matcher.Value), nil
		
	case "not_contains":
		return !strings.Contains(alertValue, matcher.Value), nil
		
	case "regex":
		return re.matchRegex(matcher.Value, alertValue)
		
	case "not_regex":
		matched, err := re.matchRegex(matcher.Value, alertValue)
		return !matched, err
		
	case "in":
		values := strings.Split(matcher.Value, "|")
		for _, v := range values {
			if strings.TrimSpace(v) == alertValue {
				return true, nil
			}
		}
		return false, nil
		
	case "not_in":
		values := strings.Split(matcher.Value, "|")
		for _, v := range values {
			if strings.TrimSpace(v) == alertValue {
				return false, nil
			}
		}
		return true, nil
		
	case "gt":
		return re.compareNumeric(alertValue, matcher.Value, ">")
		
	case "gte":
		return re.compareNumeric(alertValue, matcher.Value, ">=")
		
	case "lt":
		return re.compareNumeric(alertValue, matcher.Value, "<")
		
	case "lte":
		return re.compareNumeric(alertValue, matcher.Value, "<=")
		
	default:
		return false, fmt.Errorf("unsupported matcher operator: %s", matcher.Operator)
	}
}

// getAlertValue gets a value from alert labels or annotations
func (re *RuleEngine) getAlertValue(fieldName string, alert *models.Alert) string {
	// Check special fields first
	switch fieldName {
	case "status":
		return alert.Status
	case "severity":
		return alert.Severity
	case "fingerprint":
		return alert.Fingerprint
	}

	// Check labels
	if alert.Labels != nil {
		if value, exists := alert.Labels[fieldName]; exists {
			if str, ok := value.(string); ok {
				return str
			}
			return fmt.Sprintf("%v", value)
		}
	}

	// Check annotations
	if alert.Annotations != nil {
		if value, exists := alert.Annotations[fieldName]; exists {
			if str, ok := value.(string); ok {
				return str
			}
			return fmt.Sprintf("%v", value)
		}
	}

	return ""
}

// matchRegex performs regex matching with caching
func (re *RuleEngine) matchRegex(pattern, text string) (bool, error) {
	if pattern == "" {
		return text == "", nil
	}
	
	regex, err := regexp.Compile(pattern)
	if err != nil {
		return false, fmt.Errorf("invalid regex pattern '%s': %w", pattern, err)
	}
	
	return regex.MatchString(text), nil
}

// compareNumeric compares numeric values
func (re *RuleEngine) compareNumeric(alertValue, ruleValue, operator string) (bool, error) {
	alertNum, err := strconv.ParseFloat(alertValue, 64)
	if err != nil {
		return false, fmt.Errorf("alert value '%s' is not numeric", alertValue)
	}
	
	ruleNum, err := strconv.ParseFloat(ruleValue, 64)
	if err != nil {
		return false, fmt.Errorf("rule value '%s' is not numeric", ruleValue)
	}
	
	switch operator {
	case ">":
		return alertNum > ruleNum, nil
	case ">=":
		return alertNum >= ruleNum, nil
	case "<":
		return alertNum < ruleNum, nil
	case "<=":
		return alertNum <= ruleNum, nil
	default:
		return false, fmt.Errorf("unsupported numeric operator: %s", operator)
	}
}

// TestRule tests if a rule matches a sample alert
func (re *RuleEngine) TestRule(ctx context.Context, conditions map[string]interface{}, sampleAlert models.Alert) (bool, []models.RoutingRule, error) {
	// Convert conditions to JSONB
	conditionsJSON := models.JSONB(conditions)
	
	// Create a temporary rule for testing
	testRule := models.RoutingRule{
		ID:         0,
		Name:       "test_rule",
		Conditions: conditionsJSON,
		Enabled:    true,
	}
	
	matched, err := re.evaluateRule(testRule, &sampleAlert)
	if err != nil {
		return false, nil, err
	}
	
	var matchedRules []models.RoutingRule
	if matched {
		matchedRules = append(matchedRules, testRule)
	}
	
	return matched, matchedRules, nil
}

// RefreshRules reloads rules from database if they've been updated
func (re *RuleEngine) RefreshRules() error {
	return re.LoadRules()
}

// GetActiveRules returns currently loaded rules
func (re *RuleEngine) GetActiveRules() []models.RoutingRule {
	re.rulesLock.RLock()
	defer re.rulesLock.RUnlock()
	
	rules := make([]models.RoutingRule, len(re.rules))
	copy(rules, re.rules)
	return rules
}

// ValidateRule validates rule conditions syntax
func (re *RuleEngine) ValidateRule(conditions models.JSONB) error {
	_, err := re.parseConditions(conditions)
	if err != nil {
		return fmt.Errorf("invalid rule conditions: %w", err)
	}
	
	// Additional validation can be added here
	return nil
}