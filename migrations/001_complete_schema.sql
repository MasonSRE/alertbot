-- =============================================
-- AlertBot Complete Schema Migration
-- =============================================
-- This file provides a complete database schema as SQL backup
-- Generated automatically from Go models and optimizations

-- Drop existing tables if they exist (for clean install)
DROP TABLE IF EXISTS inhibition_status CASCADE;
DROP TABLE IF EXISTS inhibition_rules CASCADE;
DROP TABLE IF EXISTS alert_group_rules CASCADE;
DROP TABLE IF EXISTS alert_groups CASCADE;
DROP TABLE IF EXISTS alert_history CASCADE;
DROP TABLE IF EXISTS silences CASCADE;
DROP TABLE IF EXISTS notification_channels CASCADE;
DROP TABLE IF EXISTS routing_rules CASCADE;
DROP TABLE IF EXISTS notification_configs CASCADE;
DROP TABLE IF EXISTS prometheus_configs CASCADE;
DROP TABLE IF EXISTS system_configs CASCADE;
DROP TABLE IF EXISTS alerts CASCADE;

-- =============================================
-- CORE TABLES
-- =============================================

-- Alerts table (main entity)
CREATE TABLE alerts (
    id BIGSERIAL PRIMARY KEY,
    fingerprint VARCHAR(64) UNIQUE NOT NULL,
    labels JSONB NOT NULL DEFAULT '{}',
    annotations JSONB NOT NULL DEFAULT '{}',
    status VARCHAR(20) NOT NULL DEFAULT 'firing',
    severity VARCHAR(20) NOT NULL DEFAULT 'warning',
    starts_at TIMESTAMP WITH TIME ZONE NOT NULL,
    ends_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Routing rules table
CREATE TABLE routing_rules (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    conditions JSONB NOT NULL DEFAULT '{}',
    receivers JSONB NOT NULL DEFAULT '[]',
    priority INTEGER NOT NULL DEFAULT 0,
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Notification channels table
CREATE TABLE notification_channels (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL,
    config JSONB NOT NULL DEFAULT '{}',
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Silences table
CREATE TABLE silences (
    id BIGSERIAL PRIMARY KEY,
    matchers JSONB NOT NULL DEFAULT '[]',
    starts_at TIMESTAMP WITH TIME ZONE NOT NULL,
    ends_at TIMESTAMP WITH TIME ZONE NOT NULL,
    creator VARCHAR(255) NOT NULL,
    comment TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Alert history table
CREATE TABLE alert_history (
    id BIGSERIAL PRIMARY KEY,
    alert_fingerprint VARCHAR(64) NOT NULL,
    action VARCHAR(50) NOT NULL,
    details JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Alert groups table
CREATE TABLE alert_groups (
    id BIGSERIAL PRIMARY KEY,
    group_key VARCHAR(255) UNIQUE NOT NULL,
    group_by JSONB NOT NULL DEFAULT '{}',
    common_labels JSONB NOT NULL DEFAULT '{}',
    alert_count INTEGER NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'firing',
    severity VARCHAR(20) NOT NULL DEFAULT 'warning',
    first_alert_at TIMESTAMP WITH TIME ZONE NOT NULL,
    last_alert_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Alert group rules table
CREATE TABLE alert_group_rules (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    group_by JSONB NOT NULL DEFAULT '[]',
    group_wait INTEGER NOT NULL DEFAULT 10,
    group_interval INTEGER NOT NULL DEFAULT 300,
    repeat_interval INTEGER NOT NULL DEFAULT 3600,
    matchers JSONB DEFAULT '[]',
    priority INTEGER NOT NULL DEFAULT 0,
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Inhibition rules table
CREATE TABLE inhibition_rules (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    source_matchers JSONB NOT NULL DEFAULT '[]',
    target_matchers JSONB NOT NULL DEFAULT '[]',
    equal_labels JSONB DEFAULT '[]',
    duration INTEGER NOT NULL DEFAULT 0,
    priority INTEGER NOT NULL DEFAULT 0,
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Inhibition status table
CREATE TABLE inhibition_status (
    id BIGSERIAL PRIMARY KEY,
    source_fingerprint VARCHAR(64) NOT NULL,
    target_fingerprint VARCHAR(64) NOT NULL,
    rule_id BIGINT NOT NULL REFERENCES inhibition_rules(id) ON DELETE CASCADE,
    inhibited_at TIMESTAMP WITH TIME ZONE NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- =============================================
-- SETTINGS TABLES
-- =============================================

-- System configuration table
CREATE TABLE system_configs (
    id BIGSERIAL PRIMARY KEY,
    system_name VARCHAR(255) NOT NULL DEFAULT 'AlertBot',
    admin_email VARCHAR(255) NOT NULL,
    retention_days INTEGER NOT NULL DEFAULT 30,
    enable_notifications BOOLEAN NOT NULL DEFAULT true,
    enable_webhooks BOOLEAN NOT NULL DEFAULT true,
    webhook_timeout INTEGER NOT NULL DEFAULT 30,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Prometheus configuration table
CREATE TABLE prometheus_configs (
    id BIGSERIAL PRIMARY KEY,
    enabled BOOLEAN NOT NULL DEFAULT true,
    url VARCHAR(255) NOT NULL DEFAULT 'http://localhost:9090',
    timeout INTEGER NOT NULL DEFAULT 30,
    query_timeout INTEGER NOT NULL DEFAULT 30,
    scrape_interval VARCHAR(20) NOT NULL DEFAULT '15s',
    evaluation_interval VARCHAR(20) NOT NULL DEFAULT '15s',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Notification configuration table
CREATE TABLE notification_configs (
    id BIGSERIAL PRIMARY KEY,
    max_retries INTEGER NOT NULL DEFAULT 3,
    retry_interval INTEGER NOT NULL DEFAULT 30,
    rate_limit INTEGER NOT NULL DEFAULT 100,
    batch_size INTEGER NOT NULL DEFAULT 10,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- =============================================
-- CREATE INDEXES FOR PERFORMANCE
-- =============================================

-- Alert table indexes
CREATE INDEX idx_alerts_status_created ON alerts(status, created_at DESC);
CREATE INDEX idx_alerts_severity_created ON alerts(severity, created_at DESC);
CREATE INDEX idx_alerts_status_severity_created ON alerts(status, severity, created_at DESC);
CREATE INDEX idx_alerts_fingerprint_status ON alerts(fingerprint, status);
CREATE INDEX idx_alerts_starts_at ON alerts(starts_at DESC);
CREATE INDEX idx_alerts_ends_at ON alerts(ends_at DESC) WHERE ends_at IS NOT NULL;
CREATE INDEX idx_alerts_updated_at ON alerts(updated_at DESC);

-- JSONB indexes for labels and annotations
CREATE INDEX idx_alerts_labels_gin ON alerts USING GIN(labels);
CREATE INDEX idx_alerts_annotations_gin ON alerts USING GIN(annotations);

-- Specific label searches
CREATE INDEX idx_alerts_alertname ON alerts((labels->>'alertname')) WHERE labels ? 'alertname';
CREATE INDEX idx_alerts_instance ON alerts((labels->>'instance')) WHERE labels ? 'instance';
CREATE INDEX idx_alerts_job ON alerts((labels->>'job')) WHERE labels ? 'job';
CREATE INDEX idx_alerts_service ON alerts((labels->>'service')) WHERE labels ? 'service';

-- Dashboard optimized indexes
CREATE INDEX idx_alerts_dashboard_main ON alerts(status, severity, created_at DESC) WHERE status IN ('firing', 'acknowledged');
CREATE INDEX idx_alerts_recent_critical ON alerts(created_at DESC) WHERE severity = 'critical' AND status = 'firing';

-- Other table indexes
CREATE INDEX idx_routing_rules_enabled_priority ON routing_rules(enabled, priority DESC) WHERE enabled = true;
CREATE INDEX idx_routing_rules_conditions_gin ON routing_rules USING GIN(conditions);

CREATE INDEX idx_notification_channels_type_enabled ON notification_channels(type, enabled) WHERE enabled = true;
CREATE INDEX idx_notification_channels_type ON notification_channels(type);

CREATE INDEX idx_silences_active ON silences(starts_at, ends_at) WHERE ends_at > NOW();
CREATE INDEX idx_silences_matchers_gin ON silences USING GIN(matchers);

CREATE INDEX idx_alert_history_fingerprint_created ON alert_history(alert_fingerprint, created_at DESC);
CREATE INDEX idx_alert_history_action_created ON alert_history(action, created_at DESC);

CREATE INDEX idx_alert_groups_status_updated ON alert_groups(status, updated_at DESC);
CREATE INDEX idx_alert_groups_severity ON alert_groups(severity, updated_at DESC);
CREATE INDEX idx_alert_groups_key ON alert_groups(group_key);

-- =============================================
-- CREATE OPTIMIZED VIEWS
-- =============================================

-- Active alerts view
CREATE OR REPLACE VIEW active_alerts AS 
SELECT 
   id, fingerprint, labels, annotations, status, severity, 
   starts_at, ends_at, updated_at, created_at,
   labels->>'alertname' as alert_name,
   labels->>'instance' as instance,
   labels->>'job' as job,
   labels->>'service' as service
FROM alerts 
WHERE status IN ('firing', 'acknowledged') 
ORDER BY 
   CASE severity 
     WHEN 'critical' THEN 1 
     WHEN 'warning' THEN 2 
     ELSE 3 
   END, 
   created_at DESC;

-- Alert summary for dashboard
CREATE OR REPLACE VIEW alert_summary AS
SELECT 
   labels->>'alertname' as alert_name,
   labels->>'instance' as instance,
   labels->>'job' as job,
   severity,
   status,
   COUNT(*) as count,
   COUNT(CASE WHEN status = 'firing' THEN 1 END) as firing_count,
   COUNT(CASE WHEN status = 'acknowledged' THEN 1 END) as ack_count,
   MIN(created_at) as first_seen,
   MAX(created_at) as last_seen,
   MAX(updated_at) as last_updated
FROM alerts 
WHERE created_at > NOW() - INTERVAL '30 days'
GROUP BY labels->>'alertname', labels->>'instance', labels->>'job', severity, status;

-- =============================================
-- INSERT DEFAULT DATA
-- =============================================

-- Insert default system configuration
INSERT INTO system_configs (system_name, admin_email, retention_days) 
VALUES ('AlertBot', 'admin@company.com', 30)
ON CONFLICT DO NOTHING;

-- Insert default Prometheus configuration
INSERT INTO prometheus_configs (enabled, url, timeout, query_timeout) 
VALUES (true, 'http://localhost:9090', 30, 30)
ON CONFLICT DO NOTHING;

-- Insert default notification configuration
INSERT INTO notification_configs (max_retries, retry_interval, rate_limit, batch_size) 
VALUES (3, 30, 100, 10)
ON CONFLICT DO NOTHING;

-- Insert default routing rule
INSERT INTO routing_rules (name, description, conditions, receivers, priority, enabled)
VALUES (
    'Default Rule',
    'Default catch-all routing rule for all alerts',
    '{"severity": ["critical", "warning", "info"]}',
    '[{"channel_id": 1, "template": "default"}]',
    1,
    true
) ON CONFLICT DO NOTHING;

-- =============================================
-- COMPLETION LOG
-- =============================================

-- Log schema creation completion
INSERT INTO schema_migrations (version, dirty) 
VALUES ('001_complete_schema', FALSE) 
ON CONFLICT (version) DO UPDATE SET applied_at = NOW();

-- Display completion message
SELECT 'AlertBot complete schema migration completed successfully!' as message;