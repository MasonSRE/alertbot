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
)

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