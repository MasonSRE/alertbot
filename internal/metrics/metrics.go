package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Prometheus metrics for AlertBot
var (
	// HTTP request metrics
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "alertbot_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status_code"},
	)

	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "alertbot_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	// Alert processing metrics
	AlertsReceived = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "alertbot_alerts_received_total",
			Help: "Total number of alerts received",
		},
		[]string{"status", "severity"},
	)

	AlertsProcessed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "alertbot_alerts_processed_total",
			Help: "Total number of alerts processed",
		},
		[]string{"action", "status"},
	)

	AlertProcessingDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "alertbot_alert_processing_duration_seconds",
			Help:    "Time spent processing alerts in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 2.5, 5.0, 10.0},
		},
		[]string{"operation"},
	)

	ActiveAlerts = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "alertbot_active_alerts",
			Help: "Number of currently active alerts",
		},
		[]string{"severity", "status"},
	)

	// Rule engine metrics
	RulesEvaluated = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "alertbot_rules_evaluated_total",
			Help: "Total number of rules evaluated",
		},
		[]string{"rule_name", "matched"},
	)

	RuleEvaluationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "alertbot_rule_evaluation_duration_seconds",
			Help:    "Time spent evaluating rules in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0},
		},
		[]string{"rule_name"},
	)

	ActiveRules = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "alertbot_active_rules",
			Help: "Number of active routing rules",
		},
	)

	// Notification metrics
	NotificationsSent = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "alertbot_notifications_sent_total",
			Help: "Total number of notifications sent",
		},
		[]string{"channel_type", "status"},
	)

	NotificationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "alertbot_notification_duration_seconds",
			Help:    "Time spent sending notifications in seconds",
			Buckets: []float64{0.1, 0.5, 1.0, 2.5, 5.0, 10.0, 30.0},
		},
		[]string{"channel_type"},
	)

	NotificationErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "alertbot_notification_errors_total",
			Help: "Total number of notification errors",
		},
		[]string{"channel_type", "error_type"},
	)

	// WebSocket metrics
	WebSocketConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "alertbot_websocket_connections",
			Help: "Number of active WebSocket connections",
		},
	)

	WebSocketMessagesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "alertbot_websocket_messages_total",
			Help: "Total number of WebSocket messages",
		},
		[]string{"direction", "message_type"},
	)

	// Database metrics
	DatabaseConnections = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "alertbot_database_connections",
			Help: "Number of database connections",
		},
		[]string{"state"}, // open, idle, in_use
	)

	DatabaseQueryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "alertbot_database_query_duration_seconds",
			Help:    "Database query duration in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 5.0},
		},
		[]string{"operation", "table"},
	)

	DatabaseErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "alertbot_database_errors_total",
			Help: "Total number of database errors",
		},
		[]string{"operation", "error_type"},
	)

	// System metrics
	SystemUptime = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "alertbot_uptime_seconds_total",
			Help: "Total uptime of the AlertBot service in seconds",
		},
	)

	ConfigReloads = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "alertbot_config_reloads_total",
			Help: "Total number of configuration reloads",
		},
	)

	// Rate limiting metrics
	RateLimitedRequests = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "alertbot_rate_limited_requests_total",
			Help: "Total number of rate limited requests",
		},
		[]string{"client_ip"},
	)

	// Memory and performance metrics
	MemoryUsage = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "alertbot_memory_bytes",
			Help: "Memory usage in bytes",
		},
		[]string{"type"}, // heap, stack, gc
	)

	GoroutineCount = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "alertbot_goroutines",
			Help: "Number of goroutines",
		},
	)
)

// RecordHTTPRequest records HTTP request metrics
func RecordHTTPRequest(method, endpoint, statusCode string, duration float64) {
	HTTPRequestsTotal.WithLabelValues(method, endpoint, statusCode).Inc()
	HTTPRequestDuration.WithLabelValues(method, endpoint).Observe(duration)
}

// RecordAlertReceived records alert received metrics
func RecordAlertReceived(status, severity string) {
	AlertsReceived.WithLabelValues(status, severity).Inc()
}

// RecordAlertProcessed records alert processed metrics
func RecordAlertProcessed(action, status string) {
	AlertsProcessed.WithLabelValues(action, status).Inc()
}

// RecordAlertProcessingDuration records alert processing duration
func RecordAlertProcessingDuration(operation string, duration float64) {
	AlertProcessingDuration.WithLabelValues(operation).Observe(duration)
}

// UpdateActiveAlerts updates active alerts gauge
func UpdateActiveAlerts(severity, status string, count float64) {
	ActiveAlerts.WithLabelValues(severity, status).Set(count)
}

// RecordRuleEvaluation records rule evaluation metrics
func RecordRuleEvaluation(ruleName string, matched bool, duration float64) {
	matchedStr := "false"
	if matched {
		matchedStr = "true"
	}
	RulesEvaluated.WithLabelValues(ruleName, matchedStr).Inc()
	RuleEvaluationDuration.WithLabelValues(ruleName).Observe(duration)
}

// UpdateActiveRules updates active rules gauge
func UpdateActiveRules(count float64) {
	ActiveRules.Set(count)
}

// RecordNotificationSent records notification sent metrics
func RecordNotificationSent(channelType, status string, duration float64) {
	NotificationsSent.WithLabelValues(channelType, status).Inc()
	NotificationDuration.WithLabelValues(channelType).Observe(duration)
}

// RecordNotificationError records notification error metrics
func RecordNotificationError(channelType, errorType string) {
	NotificationErrors.WithLabelValues(channelType, errorType).Inc()
}

// UpdateWebSocketConnections updates WebSocket connections gauge
func UpdateWebSocketConnections(count float64) {
	WebSocketConnections.Set(count)
}

// RecordWebSocketMessage records WebSocket message metrics
func RecordWebSocketMessage(direction, messageType string) {
	WebSocketMessagesTotal.WithLabelValues(direction, messageType).Inc()
}

// UpdateDatabaseConnections updates database connections gauge
func UpdateDatabaseConnections(state string, count float64) {
	DatabaseConnections.WithLabelValues(state).Set(count)
}

// RecordDatabaseQuery records database query metrics
func RecordDatabaseQuery(operation, table string, duration float64) {
	DatabaseQueryDuration.WithLabelValues(operation, table).Observe(duration)
}

// RecordDatabaseError records database error metrics
func RecordDatabaseError(operation, errorType string) {
	DatabaseErrors.WithLabelValues(operation, errorType).Inc()
}

// IncrementSystemUptime increments system uptime
func IncrementSystemUptime() {
	SystemUptime.Inc()
}

// IncrementConfigReloads increments config reloads counter
func IncrementConfigReloads() {
	ConfigReloads.Inc()
}

// RecordRateLimitedRequest records rate limited request
func RecordRateLimitedRequest(clientIP string) {
	RateLimitedRequests.WithLabelValues(clientIP).Inc()
}

// UpdateMemoryUsage updates memory usage metrics
func UpdateMemoryUsage(memType string, bytes float64) {
	MemoryUsage.WithLabelValues(memType).Set(bytes)
}

// UpdateGoroutineCount updates goroutine count
func UpdateGoroutineCount(count float64) {
	GoroutineCount.Set(count)
}