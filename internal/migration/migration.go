package migration

import (
	"fmt"
	"time"

	"alertbot/internal/models"

	"gorm.io/gorm"
	"github.com/sirupsen/logrus"
)

// Migrator handles database migrations
type Migrator struct {
	db     *gorm.DB
	logger *logrus.Logger
}

// NewMigrator creates a new database migrator
func NewMigrator(db *gorm.DB, logger *logrus.Logger) *Migrator {
	return &Migrator{
		db:     db,
		logger: logger,
	}
}

// Migrate runs all database migrations
func (m *Migrator) Migrate() error {
	m.logger.Info("Starting database migrations")

	// Auto-migrate all models
	err := m.db.AutoMigrate(
		&models.Alert{},
		&models.RoutingRule{},
		&models.NotificationChannel{},
		&models.Silence{},
		&models.AlertHistory{},
		&models.AlertGroup{},
		&models.AlertGroupRule{},
		&models.InhibitionRule{},
		&models.InhibitionStatus{},
		&models.SystemConfig{},
		&models.PrometheusConfig{},
		&models.NotificationConfig{},
	)
	if err != nil {
		return fmt.Errorf("failed to auto-migrate models: %w", err)
	}

	// Create indexes
	if err := m.createIndexes(); err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	// Run custom migrations
	if err := m.runCustomMigrations(); err != nil {
		return fmt.Errorf("failed to run custom migrations: %w", err)
	}

	m.logger.Info("Database migrations completed successfully")
	return nil
}

// createIndexes creates additional database indexes for performance
func (m *Migrator) createIndexes() error {
	m.logger.Info("Creating database indexes")

	// High-performance indexes for optimal query performance
	indexes := []string{
		// === ALERT TABLE INDEXES === //
		
		// Primary query patterns - status and time-based searches
		"CREATE INDEX IF NOT EXISTS idx_alerts_status_created ON alerts(status, created_at DESC)",
		"CREATE INDEX IF NOT EXISTS idx_alerts_severity_created ON alerts(severity, created_at DESC)",
		"CREATE INDEX IF NOT EXISTS idx_alerts_status_severity_created ON alerts(status, severity, created_at DESC)", // Combined filter
		"CREATE INDEX IF NOT EXISTS idx_alerts_fingerprint_status ON alerts(fingerprint, status)",
		
		// Time-based queries
		"CREATE INDEX IF NOT EXISTS idx_alerts_starts_at ON alerts(starts_at DESC)",
		"CREATE INDEX IF NOT EXISTS idx_alerts_ends_at ON alerts(ends_at DESC) WHERE ends_at IS NOT NULL",
		"CREATE INDEX IF NOT EXISTS idx_alerts_updated_at ON alerts(updated_at DESC)", // For recent updates
		"CREATE INDEX IF NOT EXISTS idx_alerts_time_range ON alerts(starts_at, ends_at) WHERE ends_at IS NOT NULL", // Range queries
		
		// Fingerprint-based operations (critical for deduplication)
		"CREATE INDEX IF NOT EXISTS idx_alerts_fingerprint_unique ON alerts(fingerprint) WHERE status IN ('firing', 'acknowledged')", // Active alerts only
		"CREATE INDEX IF NOT EXISTS idx_alerts_fingerprint_time ON alerts(fingerprint, starts_at DESC)", // Alert timeline
		
		// JSONB indexes for labels and annotations (with specific paths)
		"CREATE INDEX IF NOT EXISTS idx_alerts_labels_gin ON alerts USING GIN(labels)",
		"CREATE INDEX IF NOT EXISTS idx_alerts_annotations_gin ON alerts USING GIN(annotations)",
		
		// Specific label searches (commonly used in dashboards)
		"CREATE INDEX IF NOT EXISTS idx_alerts_alertname ON alerts((labels->>'alertname')) WHERE labels ? 'alertname'",
		"CREATE INDEX IF NOT EXISTS idx_alerts_instance ON alerts((labels->>'instance')) WHERE labels ? 'instance'",
		"CREATE INDEX IF NOT EXISTS idx_alerts_job ON alerts((labels->>'job')) WHERE labels ? 'job'",
		"CREATE INDEX IF NOT EXISTS idx_alerts_service ON alerts((labels->>'service')) WHERE labels ? 'service'",
		"CREATE INDEX IF NOT EXISTS idx_alerts_cluster ON alerts((labels->>'cluster')) WHERE labels ? 'cluster'",
		
		// Composite label indexes for common queries
		"CREATE INDEX IF NOT EXISTS idx_alerts_alertname_status ON alerts((labels->>'alertname'), status) WHERE labels ? 'alertname'",
		"CREATE INDEX IF NOT EXISTS idx_alerts_instance_severity ON alerts((labels->>'instance'), severity) WHERE labels ? 'instance'",
		
		// Dashboard query optimizations
		"CREATE INDEX IF NOT EXISTS idx_alerts_dashboard_main ON alerts(status, severity, created_at DESC) WHERE status IN ('firing', 'acknowledged')", // Main dashboard
		"CREATE INDEX IF NOT EXISTS idx_alerts_recent_critical ON alerts(created_at DESC) WHERE severity = 'critical' AND status = 'firing'", // Critical alerts
		
		// === ROUTING RULES INDEXES === //
		"CREATE INDEX IF NOT EXISTS idx_routing_rules_enabled_priority ON routing_rules(enabled, priority DESC) WHERE enabled = true",
		"CREATE INDEX IF NOT EXISTS idx_routing_rules_conditions_gin ON routing_rules USING GIN(conditions)",
		"CREATE INDEX IF NOT EXISTS idx_routing_rules_updated ON routing_rules(updated_at DESC)", // Rule management
		
		// === NOTIFICATION CHANNELS INDEXES === //
		"CREATE INDEX IF NOT EXISTS idx_notification_channels_type_enabled ON notification_channels(type, enabled) WHERE enabled = true",
		"CREATE INDEX IF NOT EXISTS idx_notification_channels_type ON notification_channels(type)", // Channel listing
		"CREATE INDEX IF NOT EXISTS idx_notification_channels_name ON notification_channels(name)", // Search by name
		
		// === SILENCES INDEXES === //
		"CREATE INDEX IF NOT EXISTS idx_silences_active ON silences(starts_at, ends_at) WHERE ends_at > NOW()",
		"CREATE INDEX IF NOT EXISTS idx_silences_matchers_gin ON silences USING GIN(matchers)",
		"CREATE INDEX IF NOT EXISTS idx_silences_creator ON silences(creator, created_at DESC)", // By creator
		"CREATE INDEX IF NOT EXISTS idx_silences_time_range ON silences(starts_at, ends_at)", // Time range queries
		
		// === ALERT HISTORY INDEXES === //
		"CREATE INDEX IF NOT EXISTS idx_alert_history_fingerprint_created ON alert_history(alert_fingerprint, created_at DESC)",
		"CREATE INDEX IF NOT EXISTS idx_alert_history_action_created ON alert_history(action, created_at DESC)",
		"CREATE INDEX IF NOT EXISTS idx_alert_history_recent ON alert_history(created_at DESC) WHERE created_at > NOW() - INTERVAL '7 days'", // Recent history
		
		// === ALERT GROUP INDEXES === //
		"CREATE INDEX IF NOT EXISTS idx_alert_groups_status_updated ON alert_groups(status, updated_at DESC)",
		"CREATE INDEX IF NOT EXISTS idx_alert_groups_severity ON alert_groups(severity, updated_at DESC)",
		"CREATE INDEX IF NOT EXISTS idx_alert_groups_key ON alert_groups(group_key)", // Unique group identification
		"CREATE INDEX IF NOT EXISTS idx_alert_groups_common_labels_gin ON alert_groups USING GIN(common_labels)",
		
		// === ALERT GROUP RULES INDEXES === //
		"CREATE INDEX IF NOT EXISTS idx_alert_group_rules_enabled_priority ON alert_group_rules(enabled, priority DESC) WHERE enabled = true",
		"CREATE INDEX IF NOT EXISTS idx_alert_group_rules_matchers_gin ON alert_group_rules USING GIN(matchers)",
		
		// === INHIBITION RULES INDEXES === //
		"CREATE INDEX IF NOT EXISTS idx_inhibition_rules_enabled ON inhibition_rules(enabled) WHERE enabled = true",
		"CREATE INDEX IF NOT EXISTS idx_inhibition_rules_source_gin ON inhibition_rules USING GIN(source_matchers)",
		"CREATE INDEX IF NOT EXISTS idx_inhibition_rules_target_gin ON inhibition_rules USING GIN(target_matchers)",
		"CREATE INDEX IF NOT EXISTS idx_inhibition_rules_priority ON inhibition_rules(priority DESC, enabled) WHERE enabled = true",
		
		// === INHIBITION STATUS INDEXES === //
		"CREATE INDEX IF NOT EXISTS idx_inhibition_status_source ON inhibition_status(source_fingerprint, created_at DESC)",
		"CREATE INDEX IF NOT EXISTS idx_inhibition_status_target ON inhibition_status(target_fingerprint, created_at DESC)",
		"CREATE INDEX IF NOT EXISTS idx_inhibition_status_rule ON inhibition_status(rule_id, created_at DESC)",
		"CREATE INDEX IF NOT EXISTS idx_inhibition_status_expires ON inhibition_status(expires_at) WHERE expires_at IS NOT NULL",
		
		// === SETTINGS TABLES INDEXES === //
		"CREATE INDEX IF NOT EXISTS idx_system_config_updated ON system_configs(updated_at DESC)",
		"CREATE INDEX IF NOT EXISTS idx_prometheus_config_updated ON prometheus_configs(updated_at DESC)",
		"CREATE INDEX IF NOT EXISTS idx_notification_config_updated ON notification_configs(updated_at DESC)",
	}

	for _, indexSQL := range indexes {
		if err := m.db.Exec(indexSQL).Error; err != nil {
			m.logger.WithError(err).WithField("sql", indexSQL).Error("Failed to create index")
			return err
		}
	}

	m.logger.Info("Database indexes created successfully")
	return nil
}

// runCustomMigrations runs custom migration scripts
func (m *Migrator) runCustomMigrations() error {
	m.logger.Info("Running custom migrations")

	// Create migration tracking table
	if err := m.createMigrationTable(); err != nil {
		return err
	}

	// List of custom migrations to run
	migrations := []Migration{
		{
			ID:          "001_add_default_notification_templates",
			Description: "Add default notification templates",
			Up:          m.migration001Up,
		},
		{
			ID:          "002_create_default_routing_rules",
			Description: "Create default routing rules",
			Up:          m.migration002Up,
		},
		{
			ID:          "003_optimize_alert_queries",
			Description: "Add optimized views for alert queries",
			Up:          m.migration003Up,
		},
		{
			ID:          "004_add_performance_optimizations",
			Description: "Add advanced performance optimizations and partitioning",
			Up:          m.migration004Up,
		},
	}

	// Run each migration
	for _, migration := range migrations {
		if err := m.runMigration(migration); err != nil {
			return fmt.Errorf("failed to run migration %s: %w", migration.ID, err)
		}
	}

	m.logger.Info("Custom migrations completed")
	return nil
}

// Migration represents a database migration
type Migration struct {
	ID          string
	Description string
	Up          func() error
}

// MigrationRecord tracks applied migrations
type MigrationRecord struct {
	ID          string    `gorm:"primaryKey"`
	Description string
	AppliedAt   time.Time `gorm:"autoCreateTime"`
}

// createMigrationTable creates the migration tracking table
func (m *Migrator) createMigrationTable() error {
	return m.db.AutoMigrate(&MigrationRecord{})
}

// runMigration runs a single migration if it hasn't been applied yet
func (m *Migrator) runMigration(migration Migration) error {
	// Check if migration has already been applied
	var record MigrationRecord
	err := m.db.Where("id = ?", migration.ID).First(&record).Error
	if err == nil {
		m.logger.WithField("migration_id", migration.ID).Debug("Migration already applied, skipping")
		return nil
	}

	if err != gorm.ErrRecordNotFound {
		return fmt.Errorf("failed to check migration status: %w", err)
	}

	// Run the migration
	m.logger.WithFields(logrus.Fields{
		"migration_id":  migration.ID,
		"description":   migration.Description,
	}).Info("Running migration")

	if err := migration.Up(); err != nil {
		return err
	}

	// Record the migration as applied
	record = MigrationRecord{
		ID:          migration.ID,
		Description: migration.Description,
	}
	
	if err := m.db.Create(&record).Error; err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	m.logger.WithField("migration_id", migration.ID).Info("Migration completed successfully")
	return nil
}

// Migration implementations

func (m *Migrator) migration001Up() error {
	// This migration would add default notification templates
	// For now, it's a placeholder
	m.logger.Info("Adding default notification templates")
	return nil
}

func (m *Migrator) migration002Up() error {
	// Create a default "catch-all" routing rule if none exist
	var count int64
	if err := m.db.Model(&models.RoutingRule{}).Count(&count).Error; err != nil {
		return err
	}

	if count == 0 {
		defaultRule := &models.RoutingRule{
			Name:        "Default Rule",
			Description: "Default catch-all routing rule",
			Conditions: models.JSONB{
				"severity": []string{"critical", "warning", "info"},
			},
			Receivers: models.JSONB([]interface{}{
				map[string]interface{}{
					"channel_id": 1,
					"template":   "default",
				},
			}),
			Priority: 1,
			Enabled:  true,
		}

		if err := m.db.Create(defaultRule).Error; err != nil {
			return fmt.Errorf("failed to create default routing rule: %w", err)
		}

		m.logger.Info("Created default routing rule")
	}

	return nil
}

func (m *Migrator) migration003Up() error {
	// Create optimized views and additional performance enhancements
	views := []string{
		// Active alerts view - most frequently accessed
		`CREATE OR REPLACE VIEW active_alerts AS 
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
		   created_at DESC`,
		
		// Alert summary for dashboard
		`CREATE OR REPLACE VIEW alert_summary AS
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
		 GROUP BY labels->>'alertname', labels->>'instance', labels->>'job', severity, status`,
		
		// Recent critical alerts view
		`CREATE OR REPLACE VIEW critical_alerts AS
		 SELECT 
		   id, fingerprint, labels, annotations, status, 
		   starts_at, created_at, updated_at,
		   labels->>'alertname' as alert_name,
		   labels->>'instance' as instance,
		   annotations->>'description' as description,
		   annotations->>'summary' as summary
		 FROM alerts 
		 WHERE severity = 'critical' 
		   AND status = 'firing'
		   AND created_at > NOW() - INTERVAL '7 days'
		 ORDER BY created_at DESC`,
		
		// Alert statistics view for metrics
		`CREATE OR REPLACE VIEW alert_stats AS
		 SELECT 
		   DATE_TRUNC('hour', created_at) as time_bucket,
		   status,
		   severity,
		   labels->>'alertname' as alert_name,
		   COUNT(*) as alert_count
		 FROM alerts 
		 WHERE created_at > NOW() - INTERVAL '7 days'
		 GROUP BY DATE_TRUNC('hour', created_at), status, severity, labels->>'alertname'
		 ORDER BY time_bucket DESC`,
		
		// Active silences view
		`CREATE OR REPLACE VIEW active_silences AS
		 SELECT 
		   id, matchers, starts_at, ends_at, creator, comment, created_at
		 FROM silences 
		 WHERE starts_at <= NOW() AND ends_at > NOW()
		 ORDER BY created_at DESC`,
		
		// Notification channel status view
		`CREATE OR REPLACE VIEW channel_summary AS
		 SELECT 
		   nc.id, nc.name, nc.type, nc.enabled, nc.created_at, nc.updated_at,
		   COUNT(ah.id) as usage_count,
		   MAX(ah.created_at) as last_used
		 FROM notification_channels nc
		 LEFT JOIN alert_history ah ON ah.details->>'channel_id' = nc.id::text
		   AND ah.action = 'notification_sent'
		   AND ah.created_at > NOW() - INTERVAL '30 days'
		 GROUP BY nc.id, nc.name, nc.type, nc.enabled, nc.created_at, nc.updated_at`,
	}

	// Create performance optimization functions
	functions := []string{
		// Function to get alert fingerprint from labels
		`CREATE OR REPLACE FUNCTION get_alert_fingerprint(alert_labels JSONB) 
		 RETURNS TEXT AS $$
		 BEGIN
		   RETURN encode(digest(alert_labels::text, 'sha256'), 'hex');
		 END;
		 $$ LANGUAGE plpgsql IMMUTABLE;`,
		
		// Function to check if alert matches silence
		`CREATE OR REPLACE FUNCTION alert_matches_silence(alert_labels JSONB, silence_matchers JSONB)
		 RETURNS BOOLEAN AS $$
		 DECLARE
		   matcher JSONB;
		   label_key TEXT;
		   label_value TEXT;
		   match_type TEXT;
		   match_value TEXT;
		 BEGIN
		   FOR matcher IN SELECT jsonb_array_elements(silence_matchers)
		   LOOP
		     label_key := matcher->>'name';
		     match_type := matcher->>'type';
		     match_value := matcher->>'value';
		     label_value := alert_labels->>label_key;
		     
		     CASE match_type
		       WHEN '=' THEN
		         IF label_value != match_value THEN
		           RETURN FALSE;
		         END IF;
		       WHEN '!=' THEN
		         IF label_value = match_value THEN
		           RETURN FALSE;
		         END IF;
		       WHEN '=~' THEN
		         IF NOT (label_value ~ match_value) THEN
		           RETURN FALSE;
		         END IF;
		       WHEN '!~' THEN
		         IF (label_value ~ match_value) THEN
		           RETURN FALSE;
		         END IF;
		     END CASE;
		   END LOOP;
		   
		   RETURN TRUE;
		 END;
		 $$ LANGUAGE plpgsql IMMUTABLE;`,
	}

	// Create triggers for automatic maintenance
	triggers := []string{
		// Trigger to automatically update alert groups when alerts change
		`CREATE OR REPLACE FUNCTION update_alert_groups_trigger()
		 RETURNS TRIGGER AS $$
		 BEGIN
		   -- Update alert group statistics when alerts are modified
		   -- This is a placeholder for future implementation
		   RETURN COALESCE(NEW, OLD);
		 END;
		 $$ LANGUAGE plpgsql;`,
		
		// Create the trigger
		`DROP TRIGGER IF EXISTS alert_groups_update_trigger ON alerts;`,
		`CREATE TRIGGER alert_groups_update_trigger
		   AFTER INSERT OR UPDATE OR DELETE ON alerts
		   FOR EACH ROW EXECUTE FUNCTION update_alert_groups_trigger();`,
	}

	// Execute all views
	for _, viewSQL := range views {
		if err := m.db.Exec(viewSQL).Error; err != nil {
			m.logger.WithError(err).WithField("sql", viewSQL).Error("Failed to create view")
			return err
		}
	}

	// Execute all functions
	for _, funcSQL := range functions {
		if err := m.db.Exec(funcSQL).Error; err != nil {
			m.logger.WithError(err).WithField("sql", funcSQL).Warn("Failed to create function")
			// Don't fail migration if functions fail (PostgreSQL-specific)
		}
	}

	// Execute all triggers
	for _, triggerSQL := range triggers {
		if err := m.db.Exec(triggerSQL).Error; err != nil {
			m.logger.WithError(err).WithField("sql", triggerSQL).Warn("Failed to create trigger")
			// Don't fail migration if triggers fail
		}
	}

	return nil
}

func (m *Migrator) migration004Up() error {
	// Advanced performance optimizations
	m.logger.Info("Adding advanced performance optimizations")

	// Database-level optimizations
	optimizations := []string{
		// Enable query optimization settings
		"SET default_statistics_target = 1000", // Better statistics for query planner
		
		// Create extension for better text search if not exists
		"CREATE EXTENSION IF NOT EXISTS pg_trgm", // Trigram matching for fuzzy search
		
		// Vacuum and analyze settings optimization
		"ALTER TABLE alerts SET (autovacuum_analyze_scale_factor = 0.02)", // More frequent analyze
		"ALTER TABLE alert_history SET (autovacuum_analyze_scale_factor = 0.05)",
		
		// Enable parallel query processing
		"ALTER TABLE alerts SET (parallel_workers = 4)",
		"ALTER TABLE alert_history SET (parallel_workers = 2)",
	}

	// Additional specialized indexes for complex queries
	specializedIndexes := []string{
		// Partial indexes for better performance on specific conditions
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_alerts_firing_recent ON alerts(created_at DESC) WHERE status = 'firing' AND created_at > NOW() - INTERVAL '24 hours'",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_alerts_resolved_recent ON alerts(ends_at DESC) WHERE status = 'resolved' AND ends_at > NOW() - INTERVAL '7 days'",
		
		// Expression indexes for computed values
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_alerts_duration ON alerts((COALESCE(ends_at, NOW()) - starts_at)) WHERE ends_at IS NOT NULL",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_alerts_age ON alerts((NOW() - created_at))",
		
		// Fuzzy search indexes for alert names and descriptions
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_alerts_alertname_trgm ON alerts USING GIN ((labels->>'alertname') gin_trgm_ops) WHERE labels ? 'alertname'",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_alerts_description_trgm ON alerts USING GIN ((annotations->>'description') gin_trgm_ops) WHERE annotations ? 'description'",
		
		// Multi-column indexes for complex dashboard queries
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_alerts_dashboard_complex ON alerts(status, severity, (labels->>'alertname'), created_at DESC) WHERE status IN ('firing', 'acknowledged')",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_alerts_instance_status ON alerts((labels->>'instance'), status, created_at DESC) WHERE labels ? 'instance'",
		
		// Time-series specific indexes
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_alerts_timeseries ON alerts(starts_at, (labels->>'__name__')) WHERE labels ? '__name__'",
		
		// Covering indexes to avoid table lookups
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_alerts_list_covering ON alerts(status, created_at DESC) INCLUDE (id, fingerprint, severity, starts_at)",
		"CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_alerts_api_covering ON alerts(fingerprint) INCLUDE (status, severity, labels, annotations, starts_at, ends_at)",
	}

	// Table partitioning setup (for future large datasets)
	partitioningSQL := []string{
		// Create partitioned table for alert_history (grows fastest)
		`CREATE TABLE IF NOT EXISTS alert_history_partitioned (
			LIKE alert_history INCLUDING ALL
		) PARTITION BY RANGE (created_at)`,
		
		// Create monthly partitions for the last 6 months and next 6 months
		`CREATE TABLE IF NOT EXISTS alert_history_y2024m01 PARTITION OF alert_history_partitioned
		 FOR VALUES FROM ('2024-01-01') TO ('2024-02-01')`,
		`CREATE TABLE IF NOT EXISTS alert_history_y2024m02 PARTITION OF alert_history_partitioned
		 FOR VALUES FROM ('2024-02-01') TO ('2024-03-01')`,
		`CREATE TABLE IF NOT EXISTS alert_history_y2024m03 PARTITION OF alert_history_partitioned
		 FOR VALUES FROM ('2024-03-01') TO ('2024-04-01')`,
		`CREATE TABLE IF NOT EXISTS alert_history_y2024m04 PARTITION OF alert_history_partitioned
		 FOR VALUES FROM ('2024-04-01') TO ('2024-05-01')`,
		`CREATE TABLE IF NOT EXISTS alert_history_y2024m05 PARTITION OF alert_history_partitioned
		 FOR VALUES FROM ('2024-05-01') TO ('2024-06-01')`,
		`CREATE TABLE IF NOT EXISTS alert_history_y2024m06 PARTITION OF alert_history_partitioned
		 FOR VALUES FROM ('2024-06-01') TO ('2024-07-01')`,
		`CREATE TABLE IF NOT EXISTS alert_history_y2024m07 PARTITION OF alert_history_partitioned
		 FOR VALUES FROM ('2024-07-01') TO ('2024-08-01')`,
		`CREATE TABLE IF NOT EXISTS alert_history_y2024m08 PARTITION OF alert_history_partitioned
		 FOR VALUES FROM ('2024-08-01') TO ('2024-09-01')`,
		`CREATE TABLE IF NOT EXISTS alert_history_y2024m09 PARTITION OF alert_history_partitioned
		 FOR VALUES FROM ('2024-09-01') TO ('2024-10-01')`,
		`CREATE TABLE IF NOT EXISTS alert_history_y2024m10 PARTITION OF alert_history_partitioned
		 FOR VALUES FROM ('2024-10-01') TO ('2024-11-01')`,
		`CREATE TABLE IF NOT EXISTS alert_history_y2024m11 PARTITION OF alert_history_partitioned
		 FOR VALUES FROM ('2024-11-01') TO ('2024-12-01')`,
		`CREATE TABLE IF NOT EXISTS alert_history_y2024m12 PARTITION OF alert_history_partitioned
		 FOR VALUES FROM ('2024-12-01') TO ('2025-01-01')`,
		`CREATE TABLE IF NOT EXISTS alert_history_y2025m01 PARTITION OF alert_history_partitioned
		 FOR VALUES FROM ('2025-01-01') TO ('2025-02-01')`,
	}

	// Performance monitoring views
	monitoringViews := []string{
		// Query performance monitoring
		`CREATE OR REPLACE VIEW query_performance AS
		 SELECT 
		   schemaname,
		   tablename,
		   attname,
		   n_distinct,
		   correlation,
		   most_common_vals,
		   most_common_freqs
		 FROM pg_stats 
		 WHERE schemaname = 'public' 
		   AND tablename IN ('alerts', 'alert_history', 'routing_rules', 'notification_channels')`,
		
		// Index usage statistics
		`CREATE OR REPLACE VIEW index_usage_stats AS
		 SELECT 
		   schemaname,
		   tablename,
		   indexname,
		   idx_tup_read,
		   idx_tup_fetch,
		   idx_blks_read,
		   idx_blks_hit,
		   ROUND(100.0 * idx_blks_hit / NULLIF(idx_blks_hit + idx_blks_read, 0), 2) AS hit_ratio
		 FROM pg_stat_user_indexes pgsui
		 JOIN pg_statio_user_indexes pgsiui ON pgsui.indexrelid = pgsiui.indexrelid
		 WHERE schemaname = 'public'
		 ORDER BY idx_tup_read DESC`,
		
		// Table size and bloat monitoring
		`CREATE OR REPLACE VIEW table_stats AS
		 SELECT 
		   schemaname,
		   tablename,
		   n_tup_ins,
		   n_tup_upd,
		   n_tup_del,
		   n_live_tup,
		   n_dead_tup,
		   ROUND(100.0 * n_dead_tup / NULLIF(n_live_tup + n_dead_tup, 0), 2) AS dead_ratio,
		   last_vacuum,
		   last_autovacuum,
		   last_analyze,
		   last_autoanalyze
		 FROM pg_stat_user_tables
		 WHERE schemaname = 'public'
		 ORDER BY n_live_tup DESC`,
	}

	// Execute optimizations (with error handling)
	for _, sql := range optimizations {
		if err := m.db.Exec(sql).Error; err != nil {
			m.logger.WithError(err).WithField("sql", sql).Warn("Failed to apply optimization")
			// Don't fail migration for optimization errors
		}
	}

	// Execute specialized indexes
	for _, indexSQL := range specializedIndexes {
		if err := m.db.Exec(indexSQL).Error; err != nil {
			m.logger.WithError(err).WithField("sql", indexSQL).Warn("Failed to create specialized index")
			// Don't fail migration for index creation errors
		}
	}

	// Execute partitioning (optional, depends on database version and permissions)
	for _, partSQL := range partitioningSQL {
		if err := m.db.Exec(partSQL).Error; err != nil {
			m.logger.WithError(err).WithField("sql", partSQL).Debug("Failed to create partition")
			// Partitioning is optional and may not be supported in all environments
		}
	}

	// Execute monitoring views
	for _, viewSQL := range monitoringViews {
		if err := m.db.Exec(viewSQL).Error; err != nil {
			m.logger.WithError(err).WithField("sql", viewSQL).Warn("Failed to create monitoring view")
			// Don't fail migration for monitoring view errors
		}
	}

	// Update table statistics for better query planning
	statisticsSQL := []string{
		"ANALYZE alerts",
		"ANALYZE alert_history", 
		"ANALYZE routing_rules",
		"ANALYZE notification_channels",
		"ANALYZE silences",
		"ANALYZE alert_groups",
	}

	for _, sql := range statisticsSQL {
		if err := m.db.Exec(sql).Error; err != nil {
			m.logger.WithError(err).WithField("sql", sql).Warn("Failed to update statistics")
		}
	}

	m.logger.Info("Advanced performance optimizations completed")
	return nil
}

// DropAll drops all tables (use with caution!)
func (m *Migrator) DropAll() error {
	m.logger.Warn("Dropping all database tables")
	
	tables := []interface{}{
		&models.AlertHistory{},
		&models.Silence{},
		&models.NotificationChannel{},
		&models.RoutingRule{},
		&models.Alert{},
		&models.SystemConfig{},
		&models.PrometheusConfig{},
		&models.NotificationConfig{},
		&MigrationRecord{},
	}

	for _, table := range tables {
		if err := m.db.Migrator().DropTable(table); err != nil {
			m.logger.WithError(err).Error("Failed to drop table")
			return err
		}
	}

	return nil
}