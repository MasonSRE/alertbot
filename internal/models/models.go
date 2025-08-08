package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

type JSONB map[string]interface{}

func (j JSONB) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = make(JSONB)
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("cannot scan %T into JSONB", value)
	}

	return json.Unmarshal(bytes, j)
}

type Alert struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Fingerprint string    `json:"fingerprint" gorm:"uniqueIndex;size:64;not null"`
	Labels      JSONB     `json:"labels" gorm:"type:jsonb;not null"`
	Annotations JSONB     `json:"annotations" gorm:"type:jsonb;default:'{}'"`
	Status      string    `json:"status" gorm:"size:20;default:firing;index"`
	Severity    string    `json:"severity" gorm:"size:20;default:warning;index"`
	StartsAt    time.Time `json:"starts_at" gorm:"not null"`
	EndsAt      *time.Time `json:"ends_at"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"autoUpdateTime"`
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime;index"`
}

type RoutingRule struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Name        string    `json:"name" gorm:"size:255;not null"`
	Description string    `json:"description" gorm:"type:text"`
	Conditions  JSONB     `json:"conditions" gorm:"type:jsonb;not null"`
	Receivers   JSONB     `json:"receivers" gorm:"type:jsonb;not null"`
	Priority    int       `json:"priority" gorm:"default:0;index"`
	Enabled     bool      `json:"enabled" gorm:"default:true"`
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

type NotificationChannel struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Name      string    `json:"name" gorm:"size:255;not null"`
	Type      string    `json:"type" gorm:"size:50;not null"`
	Config    JSONB     `json:"config" gorm:"type:jsonb;not null"`
	Enabled   bool      `json:"enabled" gorm:"default:true"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

type Silence struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Matchers  JSONB     `json:"matchers" gorm:"type:jsonb;not null"`
	StartsAt  time.Time `json:"starts_at" gorm:"not null"`
	EndsAt    time.Time `json:"ends_at" gorm:"not null"`
	Creator   string    `json:"creator" gorm:"size:255;not null"`
	Comment   string    `json:"comment" gorm:"type:text"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
}

type AlertHistory struct {
	ID               uint      `json:"id" gorm:"primaryKey"`
	AlertFingerprint string    `json:"alert_fingerprint" gorm:"size:64;not null;index"`
	Action           string    `json:"action" gorm:"size:50;not null"`
	Details          JSONB     `json:"details" gorm:"type:jsonb;default:'{}'"`
	CreatedAt        time.Time `json:"created_at" gorm:"autoCreateTime"`
}

type AlertFilters struct {
	Status    string `json:"status" form:"status"`
	Severity  string `json:"severity" form:"severity"`
	AlertName string `json:"alertname" form:"alertname"`
	Instance  string `json:"instance" form:"instance"`
	Page      int    `json:"page" form:"page"`
	Size      int    `json:"size" form:"size"`
	Sort      string `json:"sort" form:"sort"`
	Order     string `json:"order" form:"order"`
}

type AlertHistoryFilters struct {
	AlertFingerprint string `json:"alert_fingerprint" form:"alert_fingerprint"`
	Action           string `json:"action" form:"action"`
	Page             int    `json:"page" form:"page"`
	Size             int    `json:"size" form:"size"`
	Sort             string `json:"sort" form:"sort"`
	Order            string `json:"order" form:"order"`
}

type PrometheusAlert struct {
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	StartsAt     time.Time         `json:"startsAt"`
	EndsAt       time.Time         `json:"endsAt"`
	GeneratorURL string            `json:"generatorURL"`
}

type AlertStatus string

const (
	AlertStatusFiring    AlertStatus = "firing"
	AlertStatusResolved  AlertStatus = "resolved"
	AlertStatusSilenced  AlertStatus = "silenced"
	AlertStatusAcknowledged AlertStatus = "acknowledged"
)

type AlertSeverity string

const (
	AlertSeverityCritical AlertSeverity = "critical"
	AlertSeverityWarning  AlertSeverity = "warning"
	AlertSeverityInfo     AlertSeverity = "info"
)

type NotificationChannelType string

const (
	ChannelTypeDingTalk   NotificationChannelType = "dingtalk"
	ChannelTypeWeChatWork NotificationChannelType = "wechat_work"
	ChannelTypeEmail      NotificationChannelType = "email"
	ChannelTypeSMS        NotificationChannelType = "sms"
	ChannelTypeTelegram   NotificationChannelType = "telegram"
	ChannelTypeSlack      NotificationChannelType = "slack"
)

// AlertGroup represents a group of alerts that share common characteristics
type AlertGroup struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	GroupKey      string    `json:"group_key" gorm:"size:255;not null;uniqueIndex"` // Hash of grouping labels
	GroupBy       JSONB     `json:"group_by" gorm:"type:jsonb;not null"`            // Labels used for grouping
	CommonLabels  JSONB     `json:"common_labels" gorm:"type:jsonb;not null"`       // Common labels across all alerts
	AlertCount    int       `json:"alert_count" gorm:"default:0"`                   // Number of alerts in group
	Status        string    `json:"status" gorm:"size:20;default:firing;index"`     // Group status (firing, resolved)
	Severity      string    `json:"severity" gorm:"size:20;default:warning;index"`  // Highest severity in group
	FirstAlertAt  time.Time `json:"first_alert_at"`                                 // Time of first alert in group
	LastAlertAt   time.Time `json:"last_alert_at"`                                  // Time of last alert in group
	CreatedAt     time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt     time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// AlertGroupRule defines how alerts should be grouped
type AlertGroupRule struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Name        string    `json:"name" gorm:"size:255;not null"`
	Description string    `json:"description" gorm:"type:text"`
	GroupBy     JSONB     `json:"group_by" gorm:"type:jsonb;not null"`     // Labels to group by
	GroupWait   int       `json:"group_wait" gorm:"default:10"`            // Seconds to wait before sending initial notification
	GroupInterval int     `json:"group_interval" gorm:"default:300"`       // Seconds between group updates
	RepeatInterval int    `json:"repeat_interval" gorm:"default:3600"`     // Seconds before repeating notifications
	Matchers    JSONB     `json:"matchers" gorm:"type:jsonb"`               // Optional matchers to filter alerts
	Priority    int       `json:"priority" gorm:"default:0;index"`         // Rule priority (higher = more important)
	Enabled     bool      `json:"enabled" gorm:"default:true"`
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// InhibitionRule defines conditions to suppress alerts based on other alerts
type InhibitionRule struct {
	ID              uint      `json:"id" gorm:"primaryKey"`
	Name            string    `json:"name" gorm:"size:255;not null"`
	Description     string    `json:"description" gorm:"type:text"`
	SourceMatchers  JSONB     `json:"source_matchers" gorm:"type:jsonb;not null"`  // Matchers for source alerts (inhibitors)
	TargetMatchers  JSONB     `json:"target_matchers" gorm:"type:jsonb;not null"`  // Matchers for target alerts (to be inhibited)
	EqualLabels     JSONB     `json:"equal_labels" gorm:"type:jsonb"`              // Labels that must be equal between source and target
	Duration        int       `json:"duration" gorm:"default:0"`                   // How long source alert must be active (seconds)
	Priority        int       `json:"priority" gorm:"default:0;index"`             // Rule priority
	Enabled         bool      `json:"enabled" gorm:"default:true"`
	CreatedAt       time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt       time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// InhibitionStatus tracks which alerts are currently inhibited
type InhibitionStatus struct {
	ID                uint      `json:"id" gorm:"primaryKey"`
	SourceFingerprint string    `json:"source_fingerprint" gorm:"size:64;not null;index"` // Alert causing inhibition
	TargetFingerprint string    `json:"target_fingerprint" gorm:"size:64;not null;index"` // Alert being inhibited
	RuleID            uint      `json:"rule_id" gorm:"not null;index"`                     // Inhibition rule applied
	InhibitedAt       time.Time `json:"inhibited_at" gorm:"not null"`                     // When inhibition started
	ExpiresAt         *time.Time `json:"expires_at"`                                       // When inhibition expires (if any)
	CreatedAt         time.Time `json:"created_at" gorm:"autoCreateTime"`
}

type Stats struct {
	TotalAlerts    int `json:"total_alerts"`
	FiringAlerts   int `json:"firing_alerts"`
	ResolvedAlerts int `json:"resolved_alerts"`
	Groups         []struct {
		Key        string  `json:"key"`
		Count      int     `json:"count"`
		Percentage float64 `json:"percentage"`
	} `json:"groups"`
	Timeline []struct {
		Timestamp string `json:"timestamp"`
		Count     int    `json:"count"`
	} `json:"timeline"`
}

// SystemConfig stores system-wide configuration settings
type SystemConfig struct {
	ID                  uint      `json:"id" gorm:"primaryKey"`
	SystemName          string    `json:"system_name" gorm:"size:255;not null"`
	AdminEmail          string    `json:"admin_email" gorm:"size:255;not null"`
	RetentionDays       int       `json:"retention_days" gorm:"not null;default:30"`
	EnableNotifications bool      `json:"enable_notifications" gorm:"default:true"`
	EnableWebhooks      bool      `json:"enable_webhooks" gorm:"default:true"`
	WebhookTimeout      int       `json:"webhook_timeout" gorm:"default:30"`
	CreatedAt           time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt           time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// PrometheusConfig stores Prometheus integration configuration
type PrometheusConfig struct {
	ID                 uint      `json:"id" gorm:"primaryKey"`
	Enabled            bool      `json:"enabled" gorm:"default:true"`
	URL                string    `json:"url" gorm:"size:255;not null"`
	Timeout            int       `json:"timeout" gorm:"default:30"`
	QueryTimeout       int       `json:"query_timeout" gorm:"default:30"`
	ScrapeInterval     string    `json:"scrape_interval" gorm:"size:20;default:'15s'"`
	EvaluationInterval string    `json:"evaluation_interval" gorm:"size:20;default:'15s'"`
	CreatedAt          time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt          time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// NotificationConfig stores global notification configuration
type NotificationConfig struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	MaxRetries    int       `json:"max_retries" gorm:"default:3"`
	RetryInterval int       `json:"retry_interval" gorm:"default:30"`
	RateLimit     int       `json:"rate_limit" gorm:"default:100"`
	BatchSize     int       `json:"batch_size" gorm:"default:10"`
	CreatedAt     time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt     time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// AlertRelations contains information about alert relationships and deduplication
type AlertRelations struct {
	Alert               *Alert   `json:"alert"`
	DuplicateOf         *Alert   `json:"duplicate_of,omitempty"`
	Duplicates          []*Alert `json:"duplicates,omitempty"`
	RelatedAlerts       []*Alert `json:"related_alerts,omitempty"`
	DeduplicationKey    string   `json:"deduplication_key"`
	CorrelationKey      string   `json:"correlation_key"`
	DeduplicationAction string   `json:"deduplication_action"`
}

// DeduplicationConfig contains configuration for alert deduplication
type DeduplicationConfig struct {
	// Time window for deduplication (alerts within this window are considered duplicates)
	DeduplicationWindow time.Duration `json:"deduplication_window"`
	
	// Labels to ignore when generating fingerprints
	IgnoreLabels []string `json:"ignore_labels"`
	
	// Labels that must match for correlation
	CorrelationLabels []string `json:"correlation_labels"`
	
	// Time window for correlation (alerts within this window can be correlated)
	CorrelationWindow time.Duration `json:"correlation_window"`
	
	// Maximum number of related alerts to track
	MaxRelatedAlerts int `json:"max_related_alerts"`
	
	// Enable time-based deduplication
	EnableTimeBasedDedup bool `json:"enable_time_based_dedup"`
	
	// Enable content-based deduplication
	EnableContentBasedDedup bool `json:"enable_content_based_dedup"`
	
	// Enable alert correlation
	EnableCorrelation bool `json:"enable_correlation"`
}