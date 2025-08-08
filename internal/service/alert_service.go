package service

import (
	"context"
	"crypto/md5"
	"fmt"
	"sort"
	"strings"
	"time"

	"alertbot/internal/engine"
	"alertbot/internal/metrics"
	"alertbot/internal/models"
	"github.com/sirupsen/logrus"
)

type alertService struct {
	deps ServiceDependencies
}

func NewAlertService(deps ServiceDependencies) AlertService {
	return &alertService{deps: deps}
}

func (s *alertService) ReceiveAlerts(ctx context.Context, prometheusAlerts []models.PrometheusAlert) error {
	start := time.Now()
	defer func() {
		metrics.RecordAlertProcessingDuration("receive_alerts", time.Since(start).Seconds())
	}()

	for _, promAlert := range prometheusAlerts {
		// Record alert received metric
		status := "firing"
		if !promAlert.EndsAt.IsZero() && promAlert.EndsAt.Before(time.Now()) {
			status = "resolved"
		}
		severity := promAlert.Labels["severity"]
		if severity == "" {
			severity = "warning"
		}
		metrics.RecordAlertReceived(status, severity)
		alert := s.convertPrometheusAlert(promAlert)
		
		// Process alert through deduplication engine
		var dedupResult *engine.DeduplicationResult
		if s.deps.DeduplicationEngine != nil {
			var err error
			dedupResult, err = s.deps.DeduplicationEngine.ProcessAlert(ctx, alert)
			if err != nil {
				s.deps.Logger.WithError(err).WithField("alert_fingerprint", alert.Fingerprint).Error("Failed to process alert deduplication")
				// Continue without deduplication if there's an error
			}
		}
		
		// Handle deduplication result
		if dedupResult != nil && dedupResult.IsDuplicate {
			s.handleDuplicateAlert(ctx, alert, dedupResult)
			continue
		}
		
		// 检查是否已存在
		existingAlert, err := s.deps.Repositories.Alert.GetByFingerprint(alert.Fingerprint)
		if err == nil {
			// 更新现有告警
			existingAlert.Status = alert.Status
			existingAlert.Annotations = alert.Annotations
			existingAlert.EndsAt = alert.EndsAt
			existingAlert.UpdatedAt = time.Now()
			
			if err := s.deps.Repositories.Alert.Update(existingAlert); err != nil {
				s.deps.Logger.WithError(err).Error("Failed to update alert")
				continue
			}
			
			// 记录历史
			history := &models.AlertHistory{
				AlertFingerprint: alert.Fingerprint,
				Action:          "updated",
				Details:         models.JSONB{"status": alert.Status},
			}
			s.deps.Repositories.AlertHistory.Create(history)
			
			// Record processing metric
			metrics.RecordAlertProcessed("updated", existingAlert.Status)
			
			// Apply routing rules for updated alerts
			s.processAlertRouting(ctx, existingAlert)
			
			// Broadcast alert update via WebSocket
			if s.deps.WebSocketHub != nil {
				s.deps.WebSocketHub.BroadcastAlertUpdate(existingAlert, "updated")
			}
		} else {
			// 创建新告警
			if err := s.deps.Repositories.Alert.Create(alert); err != nil {
				s.deps.Logger.WithError(err).Error("Failed to create alert")
				continue
			}
			
			// 记录历史
			history := &models.AlertHistory{
				AlertFingerprint: alert.Fingerprint,
				Action:          "created",
				Details:         models.JSONB{"status": alert.Status, "severity": alert.Severity},
			}
			s.deps.Repositories.AlertHistory.Create(history)
			
			// Record processing metric
			metrics.RecordAlertProcessed("created", alert.Status)
			
			// Store deduplication metadata if available
			if dedupResult != nil {
				s.storeDeduplicationMetadata(alert, dedupResult)
			}
			
			// Apply routing rules for new alerts
			s.processAlertRouting(ctx, alert)
			
			// Broadcast new alert via WebSocket
			if s.deps.WebSocketHub != nil {
				s.deps.WebSocketHub.BroadcastAlertUpdate(alert, "created")
			}
		}
	}
	
	return nil
}

func (s *alertService) GetAlert(ctx context.Context, fingerprint string) (*models.Alert, error) {
	return s.deps.Repositories.Alert.GetByFingerprint(fingerprint)
}

func (s *alertService) ListAlerts(ctx context.Context, filters models.AlertFilters) ([]models.Alert, int64, error) {
	return s.deps.Repositories.Alert.List(filters)
}

func (s *alertService) SilenceAlert(ctx context.Context, fingerprint string, duration string, comment string) error {
	alert, err := s.deps.Repositories.Alert.GetByFingerprint(fingerprint)
	if err != nil {
		return err
	}
	
	alert.Status = string(models.AlertStatusSilenced)
	alert.UpdatedAt = time.Now()
	
	if err := s.deps.Repositories.Alert.Update(alert); err != nil {
		return err
	}
	
	// 记录历史
	history := &models.AlertHistory{
		AlertFingerprint: fingerprint,
		Action:          "silenced",
		Details:         models.JSONB{"duration": duration, "comment": comment},
	}
	return s.deps.Repositories.AlertHistory.Create(history)
}

func (s *alertService) AcknowledgeAlert(ctx context.Context, fingerprint string, comment string) error {
	alert, err := s.deps.Repositories.Alert.GetByFingerprint(fingerprint)
	if err != nil {
		return err
	}
	
	alert.Status = string(models.AlertStatusAcknowledged)
	alert.UpdatedAt = time.Now()
	
	if err := s.deps.Repositories.Alert.Update(alert); err != nil {
		return err
	}
	
	// 记录历史
	history := &models.AlertHistory{
		AlertFingerprint: fingerprint,
		Action:          "acknowledged",
		Details:         models.JSONB{"comment": comment},
	}
	return s.deps.Repositories.AlertHistory.Create(history)
}

func (s *alertService) ResolveAlert(ctx context.Context, fingerprint string, comment string) error {
	alert, err := s.deps.Repositories.Alert.GetByFingerprint(fingerprint)
	if err != nil {
		return err
	}
	
	alert.Status = string(models.AlertStatusResolved)
	now := time.Now()
	alert.EndsAt = &now
	alert.UpdatedAt = now
	
	if err := s.deps.Repositories.Alert.Update(alert); err != nil {
		return err
	}
	
	// 记录历史
	history := &models.AlertHistory{
		AlertFingerprint: fingerprint,
		Action:          "resolved",
		Details:         models.JSONB{"comment": comment},
	}
	
	// Broadcast alert resolution via WebSocket
	if s.deps.WebSocketHub != nil {
		s.deps.WebSocketHub.BroadcastAlertUpdate(alert, "resolved")
	}
	
	return s.deps.Repositories.AlertHistory.Create(history)
}

func (s *alertService) convertPrometheusAlert(promAlert models.PrometheusAlert) *models.Alert {
	// 生成指纹
	fingerprint := s.generateFingerprint(promAlert.Labels)
	
	// 转换labels和annotations
	labels := models.JSONB{}
	for k, v := range promAlert.Labels {
		labels[k] = v
	}
	
	annotations := models.JSONB{}
	for k, v := range promAlert.Annotations {
		annotations[k] = v
	}
	
	// 确定状态
	status := string(models.AlertStatusFiring)
	var endsAt *time.Time
	if !promAlert.EndsAt.IsZero() && promAlert.EndsAt.Before(time.Now()) {
		status = string(models.AlertStatusResolved)
		endsAt = &promAlert.EndsAt
	}
	
	// 确定严重程度
	severity := string(models.AlertSeverityWarning)
	if sev, ok := promAlert.Labels["severity"]; ok {
		severity = sev
	}
	
	return &models.Alert{
		Fingerprint: fingerprint,
		Labels:      labels,
		Annotations: annotations,
		Status:      status,
		Severity:    severity,
		StartsAt:    promAlert.StartsAt,
		EndsAt:      endsAt,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

func (s *alertService) generateFingerprint(labels map[string]string) string {
	// 排除特定的标签
	excludeLabels := map[string]bool{
		"__name__": true,
		"job":      true,
	}
	
	var pairs []string
	for k, v := range labels {
		if !excludeLabels[k] {
			pairs = append(pairs, fmt.Sprintf("%s=%s", k, v))
		}
	}
	
	sort.Strings(pairs)
	combined := strings.Join(pairs, ",")
	
	hash := md5.Sum([]byte(combined))
	return fmt.Sprintf("%x", hash)[:16]
}

// processAlertRouting applies routing rules to an alert
func (s *alertService) processAlertRouting(ctx context.Context, alert *models.Alert) {
	if s.deps.RuleEngine == nil {
		s.deps.Logger.Debug("Rule engine not available, skipping routing")
		return
	}
	
	// Check if alert is silenced before routing
	isSilenced, silenceID := s.isAlertSilenced(ctx, alert)
	if isSilenced {
		s.deps.Logger.WithFields(logrus.Fields{
			"alert_fingerprint": alert.Fingerprint,
			"silence_id": silenceID,
		}).Info("Alert is silenced, skipping notification")
		return
	}
	
	// Check if alert is inhibited before routing
	isInhibited, inhibitionID := s.isAlertInhibited(ctx, alert)
	if isInhibited {
		s.deps.Logger.WithFields(logrus.Fields{
			"alert_fingerprint": alert.Fingerprint,
			"inhibition_id": inhibitionID,
		}).Info("Alert is inhibited, skipping notification")
		return
	}
	
	// Find matching rules
	matchedRules, err := s.deps.RuleEngine.MatchAlert(ctx, alert)
	if err != nil {
		s.deps.Logger.WithError(err).WithField("alert_fingerprint", alert.Fingerprint).Error("Failed to match routing rules")
		return
	}
	
	if len(matchedRules) == 0 {
		s.deps.Logger.WithField("alert_fingerprint", alert.Fingerprint).Debug("No routing rules matched alert")
		return
	}
	
	// Process matched rules (notification logic will be implemented later)
	for _, rule := range matchedRules {
		s.deps.Logger.WithFields(logrus.Fields{
			"alert_fingerprint": alert.Fingerprint,
			"rule_id": rule.ID,
			"rule_name": rule.Name,
		}).Info("Alert matched routing rule")
		
		// Send notifications based on rule receivers
		go s.sendRuleNotifications(context.Background(), alert, rule)
	}
}

// sendRuleNotifications sends notifications based on rule receivers
func (s *alertService) sendRuleNotifications(ctx context.Context, alert *models.Alert, rule models.RoutingRule) {
	if s.deps.NotificationManager == nil {
		s.deps.Logger.Debug("Notification manager not available")
		return
	}

	// Parse receivers from rule - JSONB is already a map[string]interface{}
	if rule.Receivers == nil {
		s.deps.Logger.WithField("rule_id", rule.ID).Error("Invalid receivers format in routing rule")
		return
	}

	// Get channels array from receivers
	channelsInterface, exists := rule.Receivers["channels"]
	if !exists {
		s.deps.Logger.WithField("rule_id", rule.ID).Debug("No channels defined in rule receivers")
		return
	}

	// Parse channel IDs
	var channelIDs []uint
	switch channels := channelsInterface.(type) {
	case []interface{}:
		for _, ch := range channels {
			switch id := ch.(type) {
			case float64:
				channelIDs = append(channelIDs, uint(id))
			case int:
				channelIDs = append(channelIDs, uint(id))
			case uint:
				channelIDs = append(channelIDs, id)
			}
		}
	}

	if len(channelIDs) == 0 {
		s.deps.Logger.WithField("rule_id", rule.ID).Debug("No valid channel IDs found in rule receivers")
		return
	}

	// Send notifications to each channel
	for _, channelID := range channelIDs {
		// Get channel configuration
		channel, err := s.deps.Repositories.NotificationChannel.GetByID(channelID)
		if err != nil {
			s.deps.Logger.WithError(err).WithField("channel_id", channelID).Error("Failed to get notification channel")
			continue
		}

		if !channel.Enabled {
			s.deps.Logger.WithField("channel_id", channelID).Debug("Channel is disabled, skipping notification")
			continue
		}

		// Convert channel type to NotificationChannelType
		channelType := models.NotificationChannelType(channel.Type)

		// Send notification through the notification manager
		start := time.Now()
		err = s.deps.NotificationManager.SendAlertNotification(ctx, alert, channel.Config, channelType)
		duration := time.Since(start).Seconds()
		
		if err != nil {
			s.deps.Logger.WithError(err).WithFields(logrus.Fields{
				"alert_fingerprint": alert.Fingerprint,
				"channel_id":        channelID,
				"channel_type":      channel.Type,
			}).Error("Failed to send notification")
			
			// Record notification failure metric
			metrics.RecordNotificationSent(channel.Type, "failed", duration)
		} else {
			s.deps.Logger.WithFields(logrus.Fields{
				"alert_fingerprint": alert.Fingerprint,
				"channel_id":        channelID,
				"channel_type":      channel.Type,
			}).Info("Notification sent successfully")
			
			// Record notification success metric
			metrics.RecordNotificationSent(channel.Type, "success", duration)
		}
	}
	
	// Notification implementation enabled
}

// BatchSilenceAlerts silences multiple alerts at once
func (s *alertService) BatchSilenceAlerts(ctx context.Context, fingerprints []string, duration string, comment string) error {
	if len(fingerprints) == 0 {
		return fmt.Errorf("no fingerprints provided")
	}

	successCount := 0
	var lastError error

	for _, fingerprint := range fingerprints {
		err := s.SilenceAlert(ctx, fingerprint, duration, comment)
		if err != nil {
			s.deps.Logger.WithError(err).WithField("fingerprint", fingerprint).Error("Failed to silence alert in batch operation")
			lastError = err
		} else {
			successCount++
		}
	}

	if successCount == 0 {
		return fmt.Errorf("failed to silence any alerts: %v", lastError)
	}

	if lastError != nil {
		s.deps.Logger.Warnf("Batch silence completed with errors: %d/%d succeeded", successCount, len(fingerprints))
	}

	return nil
}

// BatchAcknowledgeAlerts acknowledges multiple alerts at once
func (s *alertService) BatchAcknowledgeAlerts(ctx context.Context, fingerprints []string, comment string) error {
	if len(fingerprints) == 0 {
		return fmt.Errorf("no fingerprints provided")
	}

	successCount := 0
	var lastError error

	for _, fingerprint := range fingerprints {
		err := s.AcknowledgeAlert(ctx, fingerprint, comment)
		if err != nil {
			s.deps.Logger.WithError(err).WithField("fingerprint", fingerprint).Error("Failed to acknowledge alert in batch operation")
			lastError = err
		} else {
			successCount++
		}
	}

	if successCount == 0 {
		return fmt.Errorf("failed to acknowledge any alerts: %v", lastError)
	}

	if lastError != nil {
		s.deps.Logger.Warnf("Batch acknowledge completed with errors: %d/%d succeeded", successCount, len(fingerprints))
	}

	return nil
}

// BatchResolveAlerts resolves multiple alerts at once
func (s *alertService) BatchResolveAlerts(ctx context.Context, fingerprints []string, comment string) error {
	if len(fingerprints) == 0 {
		return fmt.Errorf("no fingerprints provided")
	}

	successCount := 0
	var lastError error

	for _, fingerprint := range fingerprints {
		err := s.ResolveAlert(ctx, fingerprint, comment)
		if err != nil {
			s.deps.Logger.WithError(err).WithField("fingerprint", fingerprint).Error("Failed to resolve alert in batch operation")
			lastError = err
		} else {
			successCount++
		}
	}

	if successCount == 0 {
		return fmt.Errorf("failed to resolve any alerts: %v", lastError)
	}

	if lastError != nil {
		s.deps.Logger.Warnf("Batch resolve completed with errors: %d/%d succeeded", successCount, len(fingerprints))
	}

	return nil
}

// GetAlertHistory returns the history for a specific alert by fingerprint
func (s *alertService) GetAlertHistory(ctx context.Context, fingerprint string) ([]models.AlertHistory, error) {
	if fingerprint == "" {
		return nil, fmt.Errorf("fingerprint is required")
	}

	return s.deps.Repositories.AlertHistory.GetByFingerprint(fingerprint)
}

// ListAlertHistory returns paginated alert history with optional filters
func (s *alertService) ListAlertHistory(ctx context.Context, filters models.AlertHistoryFilters) ([]models.AlertHistory, int64, error) {
	return s.deps.Repositories.AlertHistory.List(filters)
}

// handleDuplicateAlert processes duplicate alerts based on deduplication result
func (s *alertService) handleDuplicateAlert(ctx context.Context, alert *models.Alert, dedupResult *engine.DeduplicationResult) {
	if dedupResult.ExistingAlert == nil {
		return
	}

	existingAlert := dedupResult.ExistingAlert
	action := dedupResult.Action

	s.deps.Logger.WithFields(logrus.Fields{
		"alert_fingerprint":    alert.Fingerprint,
		"existing_fingerprint": existingAlert.Fingerprint,
		"action":              action,
	}).Info("Processing duplicate alert")

	switch action {
	case "update_severity":
		if s.shouldUpdateSeverity(alert, existingAlert) {
			existingAlert.Severity = alert.Severity
			existingAlert.UpdatedAt = time.Now()
			s.deps.Repositories.Alert.Update(existingAlert)
			
			// Record severity update
			s.recordAlertHistory(existingAlert.Fingerprint, "severity_updated", 
				models.JSONB{"old_severity": existingAlert.Severity, "new_severity": alert.Severity})
		}

	case "update_status":
		if existingAlert.Status != alert.Status {
			existingAlert.Status = alert.Status
			existingAlert.UpdatedAt = time.Now()
			s.deps.Repositories.Alert.Update(existingAlert)
			
			// Record status update
			s.recordAlertHistory(existingAlert.Fingerprint, "status_updated",
				models.JSONB{"old_status": existingAlert.Status, "new_status": alert.Status})
		}

	case "refresh":
		existingAlert.UpdatedAt = time.Now()
		existingAlert.Annotations = alert.Annotations // Update annotations
		s.deps.Repositories.Alert.Update(existingAlert)
		
		// Record refresh
		s.recordAlertHistory(existingAlert.Fingerprint, "refreshed", models.JSONB{})

	case "ignore":
	default:
		// Just record that we saw a duplicate
		s.recordAlertHistory(existingAlert.Fingerprint, "duplicate_ignored", 
			models.JSONB{"deduplication_key": dedupResult.DeduplicationKey})
	}

	// Update metrics
	metrics.RecordAlertProcessed("deduplicated", existingAlert.Status)
	
	// Broadcast update if action was taken
	if action != "ignore" && s.deps.WebSocketHub != nil {
		s.deps.WebSocketHub.BroadcastAlertUpdate(existingAlert, "deduplicated")
	}
}

// isAlertSilenced checks if an alert matches any active silence rules
func (s *alertService) isAlertSilenced(ctx context.Context, alert *models.Alert) (bool, uint) {
	// Get all active silences
	silences, err := s.deps.Repositories.Silence.List()
	if err != nil {
		s.deps.Logger.WithError(err).Error("Failed to get silences")
		return false, 0
	}
	
	now := time.Now()
	
	for _, silence := range silences {
		// Check if silence is currently active
		if now.Before(silence.StartsAt) || now.After(silence.EndsAt) {
			continue
		}
		
		// Check if alert matches silence matchers
		if s.alertMatchesSilence(alert, silence) {
			return true, silence.ID
		}
	}
	
	return false, 0
}

// alertMatchesSilence checks if an alert matches a silence rule
func (s *alertService) alertMatchesSilence(alert *models.Alert, silence models.Silence) bool {
	// Parse matchers from silence
	matchersData, ok := silence.Matchers["matchers"]
	if !ok {
		return false
	}
	
	matchers, ok := matchersData.([]interface{})
	if !ok {
		return false
	}
	
	// All matchers must match for the silence to apply
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
		
		// Get the label value from the alert
		alertLabelInterface, exists := alert.Labels[name]
		if !exists {
			return false // If the label doesn't exist, the matcher doesn't match
		}
		
		// Convert interface{} to string
		alertLabelValue, ok := alertLabelInterface.(string)
		if !ok {
			// Try to convert to string using fmt.Sprint
			alertLabelValue = fmt.Sprint(alertLabelInterface)
		}
		
		// Check if the value matches
		if isRegex {
			// TODO: Implement regex matching
			// For now, we'll do a simple contains check
			if !strings.Contains(alertLabelValue, value) {
				return false
			}
		} else {
			// Exact match
			if alertLabelValue != value {
				return false
			}
		}
	}
	
	// All matchers matched
	return true
}

// isAlertInhibited checks if an alert matches any active inhibition rules
func (s *alertService) isAlertInhibited(ctx context.Context, alert *models.Alert) (bool, uint) {
	// Get all active inhibition rules
	inhibitions, err := s.deps.Repositories.Inhibition.List()
	if err != nil {
		s.deps.Logger.WithError(err).Error("Failed to get inhibitions")
		return false, 0
	}
	
	for _, inhibition := range inhibitions {
		if !inhibition.Enabled {
			continue
		}
		
		// Check if this alert matches the target matchers
		if !s.alertMatchesInhibitionTarget(alert, inhibition) {
			continue
		}
		
		// Now check if there are any active source alerts that match
		sourceAlerts, err := s.findInhibitionSourceAlerts(ctx, inhibition, alert)
		if err != nil {
			s.deps.Logger.WithError(err).Error("Failed to find source alerts for inhibition")
			continue
		}
		
		if len(sourceAlerts) > 0 {
			// Found source alerts that inhibit this alert
			return true, inhibition.ID
		}
	}
	
	return false, 0
}

// alertMatchesInhibitionTarget checks if an alert matches inhibition target matchers
func (s *alertService) alertMatchesInhibitionTarget(alert *models.Alert, inhibition models.InhibitionRule) bool {
	// Parse target matchers
	targetMatchers, ok := inhibition.TargetMatchers["matchers"]
	if !ok {
		return false
	}
	
	matchers, ok := targetMatchers.([]interface{})
	if !ok {
		return false
	}
	
	// All matchers must match
	for _, matcherInterface := range matchers {
		matcher, ok := matcherInterface.(map[string]interface{})
		if !ok {
			continue
		}
		
		name, _ := matcher["name"].(string)
		value, _ := matcher["value"].(string)
		isRegex, _ := matcher["is_regex"].(bool)
		
		alertLabelInterface, exists := alert.Labels[name]
		if !exists {
			return false
		}
		
		alertLabelValue, ok := alertLabelInterface.(string)
		if !ok {
			alertLabelValue = fmt.Sprint(alertLabelInterface)
		}
		
		if isRegex {
			if !strings.Contains(alertLabelValue, value) {
				return false
			}
		} else {
			if alertLabelValue != value {
				return false
			}
		}
	}
	
	return true
}

// findInhibitionSourceAlerts finds active source alerts for an inhibition rule
func (s *alertService) findInhibitionSourceAlerts(ctx context.Context, inhibition models.InhibitionRule, targetAlert *models.Alert) ([]models.Alert, error) {
	// Get all firing alerts
	filters := models.AlertFilters{
		Status: "firing",
	}
	alerts, _, err := s.deps.Repositories.Alert.List(filters)
	if err != nil {
		return nil, err
	}
	
	var sourceAlerts []models.Alert
	
	// Parse source matchers
	sourceMatchers, ok := inhibition.SourceMatchers["matchers"]
	if !ok {
		return sourceAlerts, nil
	}
	
	matchers, ok := sourceMatchers.([]interface{})
	if !ok {
		return sourceAlerts, nil
	}
	
	// Check each alert to see if it matches source matchers
	for _, alert := range alerts {
		if alert.ID == targetAlert.ID {
			continue // Skip the target alert itself
		}
		
		// Check if alert matches all source matchers
		matchesAll := true
		for _, matcherInterface := range matchers {
			matcher, ok := matcherInterface.(map[string]interface{})
			if !ok {
				matchesAll = false
				break
			}
			
			name, _ := matcher["name"].(string)
			value, _ := matcher["value"].(string)
			isRegex, _ := matcher["is_regex"].(bool)
			
			alertLabelInterface, exists := alert.Labels[name]
			if !exists {
				matchesAll = false
				break
			}
			
			alertLabelValue, ok := alertLabelInterface.(string)
			if !ok {
				alertLabelValue = fmt.Sprint(alertLabelInterface)
			}
			
			if isRegex {
				if !strings.Contains(alertLabelValue, value) {
					matchesAll = false
					break
				}
			} else {
				if alertLabelValue != value {
					matchesAll = false
					break
				}
			}
		}
		
		if !matchesAll {
			continue
		}
		
		// Check if equal_labels match (if specified)
		if inhibition.EqualLabels != nil && len(inhibition.EqualLabels) > 0 {
			// EqualLabels is a JSONB which is map[string]interface{}
			// It should contain an array of label names
			equalLabelsInterface, hasEqual := inhibition.EqualLabels["equal"]
			if !hasEqual {
				// Try direct array format
				equalLabelsInterface = inhibition.EqualLabels["labels"]
			}
			
			equalLabels, ok := equalLabelsInterface.([]interface{})
			if ok && len(equalLabels) > 0 {
				allEqual := true
				for _, labelInterface := range equalLabels {
					label, ok := labelInterface.(string)
					if !ok {
						continue
					}
					
					sourceValue, sourceExists := alert.Labels[label]
					targetValue, targetExists := targetAlert.Labels[label]
					
					if !sourceExists || !targetExists {
						allEqual = false
						break
					}
					
					// Convert to string for comparison
					sourceStr := fmt.Sprint(sourceValue)
					targetStr := fmt.Sprint(targetValue)
					
					if sourceStr != targetStr {
						allEqual = false
						break
					}
				}
				
				if !allEqual {
					continue
				}
			}
		}
		
		// This alert matches all criteria
		sourceAlerts = append(sourceAlerts, alert)
	}
	
	return sourceAlerts, nil
}

// storeDeduplicationMetadata stores deduplication information in alert history
func (s *alertService) storeDeduplicationMetadata(alert *models.Alert, dedupResult *engine.DeduplicationResult) {
	details := models.JSONB{
		"deduplication_key": dedupResult.DeduplicationKey,
		"correlation_key":   dedupResult.CorrelationKey,
		"related_count":     len(dedupResult.RelatedAlerts),
	}

	if len(dedupResult.RelatedAlerts) > 0 {
		relatedFingerprints := make([]string, 0, len(dedupResult.RelatedAlerts))
		for _, related := range dedupResult.RelatedAlerts {
			relatedFingerprints = append(relatedFingerprints, related.Fingerprint)
		}
		details["related_fingerprints"] = relatedFingerprints
	}

	s.recordAlertHistory(alert.Fingerprint, "deduplication_processed", details)
}

// shouldUpdateSeverity determines if severity should be updated
func (s *alertService) shouldUpdateSeverity(newAlert, existingAlert *models.Alert) bool {
	severityOrder := map[string]int{
		"info":     1,
		"warning":  2,
		"critical": 3,
	}
	
	newLevel := severityOrder[newAlert.Severity]
	existingLevel := severityOrder[existingAlert.Severity]
	
	return newLevel > existingLevel
}

// recordAlertHistory creates an alert history entry
func (s *alertService) recordAlertHistory(fingerprint, action string, details models.JSONB) {
	history := &models.AlertHistory{
		AlertFingerprint: fingerprint,
		Action:          action,
		Details:         details,
	}
	
	if err := s.deps.Repositories.AlertHistory.Create(history); err != nil {
		s.deps.Logger.WithError(err).WithField("fingerprint", fingerprint).Error("Failed to record alert history")
	}
}

// GetAlertRelations returns alert relationships and deduplication information
func (s *alertService) GetAlertRelations(ctx context.Context, fingerprint string) (*models.AlertRelations, error) {
	alert, err := s.deps.Repositories.Alert.GetByFingerprint(fingerprint)
	if err != nil {
		return nil, err
	}

	relations := &models.AlertRelations{
		Alert: alert,
	}

	// Get deduplication information if deduplication engine is available
	if s.deps.DeduplicationEngine != nil {
		dedupResult, err := s.deps.DeduplicationEngine.ProcessAlert(ctx, alert)
		if err != nil {
			s.deps.Logger.WithError(err).WithField("fingerprint", fingerprint).Warn("Failed to get deduplication information")
		} else {
			relations.DeduplicationKey = dedupResult.DeduplicationKey
			relations.CorrelationKey = dedupResult.CorrelationKey
			relations.DeduplicationAction = dedupResult.Action
			relations.RelatedAlerts = dedupResult.RelatedAlerts
			
			if dedupResult.IsDuplicate && dedupResult.ExistingAlert != nil {
				relations.DuplicateOf = dedupResult.ExistingAlert
			}
		}
	}

	// Find duplicates of this alert
	duplicates, err := s.findDuplicatesOf(ctx, fingerprint)
	if err != nil {
		s.deps.Logger.WithError(err).WithField("fingerprint", fingerprint).Warn("Failed to find duplicates")
	} else {
		relations.Duplicates = duplicates
	}

	return relations, nil
}

// findDuplicatesOf finds alerts that are duplicates of the given alert
func (s *alertService) findDuplicatesOf(ctx context.Context, fingerprint string) ([]*models.Alert, error) {
	// For now, use alert history to find duplicates
	// This could be optimized with better indexing in the future
	historyFilters := models.AlertHistoryFilters{
		Action: "duplicate_ignored",
		Size:   100,
	}
	
	histories, _, err := s.deps.Repositories.AlertHistory.List(historyFilters)
	if err != nil {
		return nil, err
	}

	var duplicateFingerprints []string
	for _, history := range histories {
		if details, ok := history.Details["target_fingerprint"].(string); ok && details == fingerprint {
			duplicateFingerprints = append(duplicateFingerprints, history.AlertFingerprint)
		}
	}

	var duplicates []*models.Alert
	for _, fp := range duplicateFingerprints {
		if duplicate, err := s.deps.Repositories.Alert.GetByFingerprint(fp); err == nil {
			duplicates = append(duplicates, duplicate)
		}
	}

	return duplicates, nil
}

// UpdateDeduplicationConfig updates the deduplication engine configuration
func (s *alertService) UpdateDeduplicationConfig(ctx context.Context, config models.DeduplicationConfig) error {
	if s.deps.DeduplicationEngine == nil {
		return fmt.Errorf("deduplication engine not available")
	}

	// Convert models.DeduplicationConfig to engine.DeduplicationConfig
	engineConfig := engine.DeduplicationConfig{
		DeduplicationWindow:     config.DeduplicationWindow,
		IgnoreLabels:           config.IgnoreLabels,
		CorrelationLabels:      config.CorrelationLabels,
		CorrelationWindow:      config.CorrelationWindow,
		MaxRelatedAlerts:       config.MaxRelatedAlerts,
		EnableTimeBasedDedup:   config.EnableTimeBasedDedup,
		EnableContentBasedDedup: config.EnableContentBasedDedup,
		EnableCorrelation:      config.EnableCorrelation,
	}

	s.deps.DeduplicationEngine.UpdateDeduplicationConfig(engineConfig)
	
	s.deps.Logger.WithFields(logrus.Fields{
		"deduplication_window": config.DeduplicationWindow,
		"correlation_window":   config.CorrelationWindow,
		"max_related_alerts":   config.MaxRelatedAlerts,
	}).Info("Deduplication configuration updated")

	return nil
}