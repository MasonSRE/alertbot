package engine

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	"alertbot/internal/metrics"
	"alertbot/internal/models"
	"alertbot/internal/repository"

	"github.com/sirupsen/logrus"
)

// DeduplicationEngine handles alert deduplication and correlation
type DeduplicationEngine struct {
	repo   *repository.Repositories
	logger *logrus.Logger
	config DeduplicationConfig
}

// DeduplicationConfig holds configuration for deduplication
type DeduplicationConfig struct {
	// Time window for deduplication (alerts within this window are considered duplicates)
	DeduplicationWindow time.Duration
	
	// Labels to ignore when generating fingerprints
	IgnoreLabels []string
	
	// Labels that must match for correlation
	CorrelationLabels []string
	
	// Time window for correlation (alerts within this window can be correlated)
	CorrelationWindow time.Duration
	
	// Maximum number of related alerts to track
	MaxRelatedAlerts int
	
	// Enable time-based deduplication
	EnableTimeBasedDedup bool
	
	// Enable content-based deduplication
	EnableContentBasedDedup bool
	
	// Enable alert correlation
	EnableCorrelation bool
}

// DeduplicationResult contains the result of deduplication check
type DeduplicationResult struct {
	IsDuplicate      bool
	ExistingAlert    *models.Alert
	RelatedAlerts    []*models.Alert
	DeduplicationKey string
	CorrelationKey   string
	Action           string // "create", "update", "ignore"
}

// CorrelationRule defines how alerts should be correlated
type CorrelationRule struct {
	Name        string
	Description string
	Matchers    map[string]interface{}
	TimeWindow  time.Duration
	MaxAlerts   int
}

func NewDeduplicationEngine(repo *repository.Repositories, logger *logrus.Logger) *DeduplicationEngine {
	config := DeduplicationConfig{
		DeduplicationWindow:     5 * time.Minute,  // 5 minutes window for duplicates
		CorrelationWindow:      30 * time.Minute, // 30 minutes window for correlation
		MaxRelatedAlerts:       10,
		EnableTimeBasedDedup:   true,
		EnableContentBasedDedup: true,
		EnableCorrelation:      true,
		IgnoreLabels: []string{
			"__name__",
			"__tmp_",
			"timestamp",
			"receive_timestamp",
		},
		CorrelationLabels: []string{
			"instance",
			"job",
			"service",
			"cluster",
			"node",
		},
	}

	return &DeduplicationEngine{
		repo:   repo,
		logger: logger,
		config: config,
	}
}

// ProcessAlert performs deduplication and correlation for an incoming alert
func (de *DeduplicationEngine) ProcessAlert(ctx context.Context, alert *models.Alert) (*DeduplicationResult, error) {
	start := time.Now()
	defer func() {
		metrics.RecordDeduplicationDuration(time.Since(start).Seconds())
	}()

	result := &DeduplicationResult{
		RelatedAlerts: make([]*models.Alert, 0),
	}

	// Generate deduplication key
	dedupKey := de.generateDeduplicationKey(alert)
	result.DeduplicationKey = dedupKey

	// Generate correlation key
	corrKey := de.generateCorrelationKey(alert)
	result.CorrelationKey = corrKey

	de.logger.WithFields(logrus.Fields{
		"alert_fingerprint":  alert.Fingerprint,
		"deduplication_key":  dedupKey,
		"correlation_key":    corrKey,
	}).Debug("Processing alert for deduplication and correlation")

	// Check for existing duplicate
	if de.config.EnableTimeBasedDedup || de.config.EnableContentBasedDedup {
		existingAlert, err := de.findDuplicate(ctx, alert, dedupKey)
		if err != nil {
			return nil, fmt.Errorf("failed to check for duplicates: %w", err)
		}

		if existingAlert != nil {
			result.IsDuplicate = true
			result.ExistingAlert = existingAlert
			result.Action = de.determineUpdateAction(alert, existingAlert)
			
			// Record duplicate metrics
			if de.config.EnableTimeBasedDedup {
				metrics.RecordDeduplicationDuplicate("time_based")
			}
			if de.config.EnableContentBasedDedup {
				metrics.RecordDeduplicationDuplicate("content_based")
			}
			
			de.logger.WithFields(logrus.Fields{
				"existing_alert_id": existingAlert.ID,
				"action":           result.Action,
			}).Info("Duplicate alert detected")
		}
	}

	// Find related alerts for correlation
	if de.config.EnableCorrelation {
		relatedAlerts, err := de.findRelatedAlerts(ctx, alert, corrKey)
		if err != nil {
			de.logger.WithError(err).Warn("Failed to find related alerts")
		} else {
			result.RelatedAlerts = relatedAlerts
			
			if len(relatedAlerts) > 0 {
				// Record correlation metrics
				for range relatedAlerts {
					metrics.RecordDeduplicationCorrelation()
				}
				
				de.logger.WithFields(logrus.Fields{
					"related_count": len(relatedAlerts),
				}).Info("Found related alerts for correlation")
			}
		}
	}

	// Determine final action
	if !result.IsDuplicate {
		result.Action = "create"
	}

	// Record final metrics
	metrics.RecordDeduplicationProcessed(result.Action)

	return result, nil
}

// generateDeduplicationKey creates a key for identifying duplicate alerts
func (de *DeduplicationEngine) generateDeduplicationKey(alert *models.Alert) string {
	// Start with alert name and critical labels
	var keyParts []string

	// Add alertname (most important for deduplication)
	if alertName, exists := alert.Labels["alertname"]; exists {
		if name, ok := alertName.(string); ok {
			keyParts = append(keyParts, fmt.Sprintf("alertname=%s", name))
		}
	}

	// Add instance information
	if instance, exists := alert.Labels["instance"]; exists {
		if inst, ok := instance.(string); ok {
			keyParts = append(keyParts, fmt.Sprintf("instance=%s", inst))
		}
	}

	// Add job information
	if job, exists := alert.Labels["job"]; exists {
		if j, ok := job.(string); ok {
			keyParts = append(keyParts, fmt.Sprintf("job=%s", j))
		}
	}

	// Add severity for more specific deduplication
	keyParts = append(keyParts, fmt.Sprintf("severity=%s", alert.Severity))

	// Add status to differentiate firing/resolved
	keyParts = append(keyParts, fmt.Sprintf("status=%s", alert.Status))

	// Add other significant labels (excluding ignored ones)
	var otherLabels []string
	for key, value := range alert.Labels {
		if de.shouldIgnoreLabel(key) {
			continue
		}
		
		// Skip already processed labels
		if key == "alertname" || key == "instance" || key == "job" {
			continue
		}
		
		if str, ok := value.(string); ok {
			otherLabels = append(otherLabels, fmt.Sprintf("%s=%s", key, str))
		}
	}
	
	// Sort for consistent key generation
	sort.Strings(otherLabels)
	keyParts = append(keyParts, otherLabels...)

	// Create hash of the key for consistent length
	keyString := strings.Join(keyParts, "|")
	hash := sha256.Sum256([]byte(keyString))
	return hex.EncodeToString(hash[:])[:16] // Use first 16 characters
}

// generateCorrelationKey creates a key for finding related alerts
func (de *DeduplicationEngine) generateCorrelationKey(alert *models.Alert) string {
	var keyParts []string

	// Use configured correlation labels
	for _, label := range de.config.CorrelationLabels {
		if value, exists := alert.Labels[label]; exists {
			if str, ok := value.(string); ok {
				keyParts = append(keyParts, fmt.Sprintf("%s=%s", label, str))
			}
		}
	}

	// Add severity for similar-impact correlation
	keyParts = append(keyParts, fmt.Sprintf("severity=%s", alert.Severity))

	if len(keyParts) == 0 {
		// Fallback to basic correlation
		keyParts = append(keyParts, "default")
	}

	// Sort for consistent key generation
	sort.Strings(keyParts)
	keyString := strings.Join(keyParts, "|")
	
	hash := sha256.Sum256([]byte(keyString))
	return hex.EncodeToString(hash[:])[:12] // Shorter key for correlation
}

// findDuplicate searches for existing duplicate alerts
func (de *DeduplicationEngine) findDuplicate(ctx context.Context, alert *models.Alert, dedupKey string) (*models.Alert, error) {
	// Build time window for search
	timeWindowStart := alert.StartsAt.Add(-de.config.DeduplicationWindow)
	timeWindowEnd := alert.StartsAt.Add(de.config.DeduplicationWindow)

	// Search for alerts with similar characteristics
	filters := models.AlertFilters{
		Status: "firing", // Only check against firing alerts
		Size:   100,      // Limit search scope
	}

	alerts, _, err := de.repo.Alert.List(filters)
	if err != nil {
		return nil, err
	}

	for _, existingAlert := range alerts {
		// Skip if outside time window
		if existingAlert.StartsAt.Before(timeWindowStart) || existingAlert.StartsAt.After(timeWindowEnd) {
			continue
		}

		// Skip self
		if existingAlert.Fingerprint == alert.Fingerprint {
			continue
		}

		// Check if deduplication keys match
		existingDedupKey := de.generateDeduplicationKey(&existingAlert)
		if existingDedupKey == dedupKey {
			return &existingAlert, nil
		}

		// Additional similarity checks
		if de.config.EnableContentBasedDedup && de.areAlertsSimilar(alert, &existingAlert) {
			return &existingAlert, nil
		}
	}

	return nil, nil
}

// findRelatedAlerts searches for alerts that might be related/correlated
func (de *DeduplicationEngine) findRelatedAlerts(ctx context.Context, alert *models.Alert, corrKey string) ([]*models.Alert, error) {
	// Build time window for correlation search
	timeWindowStart := alert.StartsAt.Add(-de.config.CorrelationWindow)
	timeWindowEnd := alert.StartsAt.Add(de.config.CorrelationWindow)

	// Search for alerts in correlation window
	filters := models.AlertFilters{
		Size: 200, // Broader search for correlation
	}

	alerts, _, err := de.repo.Alert.List(filters)
	if err != nil {
		return nil, err
	}

	var relatedAlerts []*models.Alert
	
	for _, existingAlert := range alerts {
		// Skip if outside time window
		if existingAlert.StartsAt.Before(timeWindowStart) || existingAlert.StartsAt.After(timeWindowEnd) {
			continue
		}

		// Skip self
		if existingAlert.Fingerprint == alert.Fingerprint {
			continue
		}

		// Check correlation
		if de.areAlertsCorrelated(alert, &existingAlert, corrKey) {
			relatedAlerts = append(relatedAlerts, &existingAlert)
			
			// Limit number of related alerts
			if len(relatedAlerts) >= de.config.MaxRelatedAlerts {
				break
			}
		}
	}

	return relatedAlerts, nil
}

// areAlertsSimilar checks if two alerts are similar enough to be considered duplicates
func (de *DeduplicationEngine) areAlertsSimilar(alert1, alert2 *models.Alert) bool {
	// Must have same alertname
	alertName1, ok1 := alert1.Labels["alertname"].(string)
	alertName2, ok2 := alert2.Labels["alertname"].(string)
	if !ok1 || !ok2 || alertName1 != alertName2 {
		return false
	}

	// Must have same instance (if present)
	instance1, hasInst1 := alert1.Labels["instance"].(string)
	instance2, hasInst2 := alert2.Labels["instance"].(string)
	if hasInst1 && hasInst2 && instance1 != instance2 {
		return false
	}

	// Must have same or similar severity
	if alert1.Severity != alert2.Severity {
		return false
	}

	// Check description similarity (if present)
	desc1, hasDesc1 := alert1.Annotations["description"].(string)
	desc2, hasDesc2 := alert2.Annotations["description"].(string)
	if hasDesc1 && hasDesc2 {
		// Simple similarity check - could be enhanced with fuzzy matching
		if de.stringSimilarity(desc1, desc2) < 0.8 {
			return false
		}
	}

	return true
}

// areAlertsCorrelated checks if alerts should be correlated
func (de *DeduplicationEngine) areAlertsCorrelated(alert1, alert2 *models.Alert, corrKey string) bool {
	// Generate correlation key for the second alert
	corrKey2 := de.generateCorrelationKey(alert2)
	
	// Basic correlation: same correlation key
	if corrKey == corrKey2 {
		return true
	}

	// Advanced correlation rules can be added here
	// For example: alerts from same service but different components
	service1, hasService1 := alert1.Labels["service"].(string)
	service2, hasService2 := alert2.Labels["service"].(string)
	if hasService1 && hasService2 && service1 == service2 {
		// Same service, different components might be related
		return true
	}

	// Check if alerts might be causally related
	return de.checkCausalRelation(alert1, alert2)
}

// checkCausalRelation checks if alerts might have a causal relationship
func (de *DeduplicationEngine) checkCausalRelation(alert1, alert2 *models.Alert) bool {
	// Infrastructure dependency patterns
	patterns := map[string][]string{
		"node":     {"instance", "job"},
		"service":  {"instance"},
		"database": {"service", "instance"},
		"network":  {"instance", "cluster"},
	}

	// Check if alerts follow dependency patterns
	for parentType, childLabels := range patterns {
		if de.alertMatchesType(alert1, parentType) {
			for _, childLabel := range childLabels {
				if value1, exists1 := alert1.Labels[childLabel].(string); exists1 {
					if value2, exists2 := alert2.Labels[childLabel].(string); exists2 {
						if value1 == value2 && alert1.StartsAt.Before(alert2.StartsAt) {
							// Alert1 might be causing alert2
							return true
						}
					}
				}
			}
		}
	}

	return false
}

// alertMatchesType checks if alert matches a specific type pattern
func (de *DeduplicationEngine) alertMatchesType(alert *models.Alert, alertType string) bool {
	alertName, ok := alert.Labels["alertname"].(string)
	if !ok {
		return false
	}
	
	// Simple pattern matching - could be enhanced
	return strings.Contains(strings.ToLower(alertName), strings.ToLower(alertType))
}

// determineUpdateAction determines what action to take for a duplicate alert
func (de *DeduplicationEngine) determineUpdateAction(newAlert, existingAlert *models.Alert) string {
	// If severity increased, update
	severityOrder := map[string]int{
		"info":     1,
		"warning":  2,
		"critical": 3,
	}
	
	newSeverityLevel := severityOrder[newAlert.Severity]
	existingSeverityLevel := severityOrder[existingAlert.Severity]
	
	if newSeverityLevel > existingSeverityLevel {
		return "update_severity"
	}
	
	// If status changed, update
	if newAlert.Status != existingAlert.Status {
		return "update_status"
	}
	
	// If it's been a while since last update, refresh
	if time.Since(existingAlert.UpdatedAt) > time.Hour {
		return "refresh"
	}
	
	// Otherwise, just ignore
	return "ignore"
}

// shouldIgnoreLabel checks if a label should be ignored in deduplication
func (de *DeduplicationEngine) shouldIgnoreLabel(label string) bool {
	for _, ignoreLabel := range de.config.IgnoreLabels {
		if strings.HasPrefix(label, ignoreLabel) {
			return true
		}
	}
	return false
}

// stringSimilarity calculates basic string similarity (Jaccard similarity)
func (de *DeduplicationEngine) stringSimilarity(s1, s2 string) float64 {
	if s1 == s2 {
		return 1.0
	}
	
	words1 := strings.Fields(strings.ToLower(s1))
	words2 := strings.Fields(strings.ToLower(s2))
	
	if len(words1) == 0 && len(words2) == 0 {
		return 1.0
	}
	
	if len(words1) == 0 || len(words2) == 0 {
		return 0.0
	}
	
	// Create sets
	set1 := make(map[string]bool)
	set2 := make(map[string]bool)
	
	for _, word := range words1 {
		set1[word] = true
	}
	
	for _, word := range words2 {
		set2[word] = true
	}
	
	// Calculate intersection
	intersection := 0
	for word := range set1 {
		if set2[word] {
			intersection++
		}
	}
	
	// Calculate union
	union := len(set1) + len(set2) - intersection
	
	if union == 0 {
		return 1.0
	}
	
	return float64(intersection) / float64(union)
}

// UpdateDeduplicationConfig updates the deduplication configuration
func (de *DeduplicationEngine) UpdateDeduplicationConfig(config DeduplicationConfig) {
	de.config = config
	de.logger.Info("Deduplication configuration updated")
}