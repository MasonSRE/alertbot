package repository

import (
	"context"
	"fmt"
	"time"

	"alertbot/internal/config"
	"alertbot/internal/errors"
	"alertbot/internal/recovery"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func NewDatabase(cfg config.Database) (*gorm.DB, error) {
	// Create retry configuration for database operations
	retryConfig := recovery.RetryConfig{
		MaxAttempts:   3,
		InitialDelay:  time.Second,
		MaxDelay:      10 * time.Second,
		BackoffFactor: 2.0,
		Jitter:        true,
		RetryCondition: func(err error) bool {
			return recovery.IsRetryable(err)
		},
	}

	var db *gorm.DB
	var connectionErr error

	// Retry database connection
	err := recovery.Retry(context.Background(), retryConfig, func(ctx context.Context) error {
		dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=%s TimeZone=%s",
			cfg.Host, cfg.User, cfg.Password, cfg.DBName, cfg.Port, cfg.SSLMode, cfg.TimeZone)

		gormConfig := &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent), // Use Silent in production
			PrepareStmt: true, // Enable prepared statements for better performance
			DisableForeignKeyConstraintWhenMigrating: true,
		}

		var dbErr error
		db, dbErr = gorm.Open(postgres.Open(dsn), gormConfig)
		if dbErr != nil {
			connectionErr = dbErr
			return errors.Wrap(dbErr, "DATABASE_CONNECTION_FAILED", "Failed to connect to database", 500)
		}

		sqlDB, dbErr := db.DB()
		if dbErr != nil {
			connectionErr = dbErr
			return errors.Wrap(dbErr, "DATABASE_INIT_FAILED", "Failed to get underlying sql.DB", 500)
		}

		// Connection pool configuration for high performance
		sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
		sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
		sqlDB.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetime) * time.Second)
		sqlDB.SetConnMaxIdleTime(time.Duration(cfg.ConnMaxIdleTime) * time.Second)

		// Test connection
		testCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		if pingErr := sqlDB.PingContext(testCtx); pingErr != nil {
			connectionErr = pingErr
			return errors.Wrap(pingErr, "DATABASE_PING_FAILED", "Failed to ping database", 500)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to establish database connection after retries: %w", connectionErr)
	}

	return db, nil
}