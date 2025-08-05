package service

import (
	"context"
	"crypto/md5"
	"fmt"
	"sort"
	"strings"
	"time"

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

	// Parse receivers from rule
	// Skip receivers processing for now
	if rule.Receivers == nil {
		s.deps.Logger.WithField("rule_id", rule.ID).Error("Invalid receivers format in routing rule")
		return
	}

	// Skip notification processing for now
	_ = rule.Receivers
	s.deps.Logger.WithField("rule_id", rule.ID).Debug("Notification processing not implemented yet")
	return
	
	// Notification implementation temporarily disabled
}