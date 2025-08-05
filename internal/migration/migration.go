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

	// Alert indexes for high-performance queries
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_alerts_status_created ON alerts(status, created_at DESC)",
		"CREATE INDEX IF NOT EXISTS idx_alerts_severity_created ON alerts(severity, created_at DESC)",
		"CREATE INDEX IF NOT EXISTS idx_alerts_fingerprint_status ON alerts(fingerprint, status)",
		"CREATE INDEX IF NOT EXISTS idx_alerts_starts_at ON alerts(starts_at)",
		"CREATE INDEX IF NOT EXISTS idx_alerts_ends_at ON alerts(ends_at) WHERE ends_at IS NOT NULL",
		
		// JSONB indexes for labels and annotations
		"CREATE INDEX IF NOT EXISTS idx_alerts_labels_gin ON alerts USING GIN(labels)",
		"CREATE INDEX IF NOT EXISTS idx_alerts_annotations_gin ON alerts USING GIN(annotations)",
		
		// Specific label searches (commonly used)
		"CREATE INDEX IF NOT EXISTS idx_alerts_alertname ON alerts((labels->>'alertname'))",
		"CREATE INDEX IF NOT EXISTS idx_alerts_instance ON alerts((labels->>'instance'))",
		"CREATE INDEX IF NOT EXISTS idx_alerts_job ON alerts((labels->>'job'))",
		
		// Routing rules indexes
		"CREATE INDEX IF NOT EXISTS idx_routing_rules_enabled_priority ON routing_rules(enabled, priority DESC) WHERE enabled = true",
		"CREATE INDEX IF NOT EXISTS idx_routing_rules_conditions_gin ON routing_rules USING GIN(conditions)",
		
		// Notification channels indexes
		"CREATE INDEX IF NOT EXISTS idx_notification_channels_type_enabled ON notification_channels(type, enabled) WHERE enabled = true",
		
		// Silences indexes
		"CREATE INDEX IF NOT EXISTS idx_silences_active ON silences(starts_at, ends_at) WHERE ends_at > NOW()",
		"CREATE INDEX IF NOT EXISTS idx_silences_matchers_gin ON silences USING GIN(matchers)",
		
		// Alert history indexes
		"CREATE INDEX IF NOT EXISTS idx_alert_history_fingerprint_created ON alert_history(alert_fingerprint, created_at DESC)",
		"CREATE INDEX IF NOT EXISTS idx_alert_history_action_created ON alert_history(action, created_at DESC)",
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
	// Create materialized views or additional indexes for common queries
	views := []string{
		`CREATE OR REPLACE VIEW active_alerts AS 
		 SELECT * FROM alerts 
		 WHERE status IN ('firing', 'acknowledged') 
		 ORDER BY severity DESC, created_at DESC`,
		
		`CREATE OR REPLACE VIEW alert_summary AS
		 SELECT 
		   labels->>'alertname' as alert_name,
		   labels->>'instance' as instance,
		   severity,
		   status,
		   COUNT(*) as count,
		   MIN(created_at) as first_seen,
		   MAX(created_at) as last_seen
		 FROM alerts 
		 GROUP BY labels->>'alertname', labels->>'instance', severity, status`,
	}

	for _, viewSQL := range views {
		if err := m.db.Exec(viewSQL).Error; err != nil {
			m.logger.WithError(err).WithField("sql", viewSQL).Error("Failed to create view")
			return err
		}
	}

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