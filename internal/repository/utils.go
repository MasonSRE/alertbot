package repository

import (
	"context"
	"time"

	"alertbot/internal/errors"
	"alertbot/internal/recovery"

	"github.com/lib/pq"
	"gorm.io/gorm"
)

// DatabaseUtils provides utility functions for database operations with error handling
type DatabaseUtils struct {
	db           *gorm.DB
	retryConfig  recovery.RetryConfig
}

// NewDatabaseUtils creates a new database utilities instance
func NewDatabaseUtils(db *gorm.DB) *DatabaseUtils {
	return &DatabaseUtils{
		db: db,
		retryConfig: recovery.RetryConfig{
			MaxAttempts:   2,
			InitialDelay:  100 * time.Millisecond,
			MaxDelay:      2 * time.Second,
			BackoffFactor: 2.0,
			Jitter:        true,
			RetryCondition: func(err error) bool {
				return recovery.IsRetryable(err) && !errors.IsValidationError(err)
			},
		},
	}
}

// ExecWithRetry executes a database operation with retry logic
func (du *DatabaseUtils) ExecWithRetry(ctx context.Context, operation func(*gorm.DB) error) error {
	return recovery.Retry(ctx, du.retryConfig, func(ctx context.Context) error {
		return operation(du.db.WithContext(ctx))
	})
}

// TransactionWithRetry executes a database transaction with retry logic
func (du *DatabaseUtils) TransactionWithRetry(ctx context.Context, fn func(*gorm.DB) error) error {
	return recovery.Retry(ctx, du.retryConfig, func(ctx context.Context) error {
		return du.db.WithContext(ctx).Transaction(fn)
	})
}

// HandleGormError converts GORM errors to application errors
func HandleGormError(err error) error {
	if err == nil {
		return nil
	}

	switch err {
	case gorm.ErrRecordNotFound:
		return errors.ErrRecordNotFound
	case gorm.ErrInvalidTransaction:
		return errors.NewInternalError("Invalid database transaction", err)
	case gorm.ErrNotImplemented:
		return errors.NewInternalError("Database operation not implemented", err)
	case gorm.ErrMissingWhereClause:
		return errors.NewValidationError("Missing WHERE clause in database operation", "query")
	case gorm.ErrUnsupportedRelation:
		return errors.NewInternalError("Unsupported database relation", err)
	case gorm.ErrPrimaryKeyRequired:
		return errors.NewValidationError("Primary key required for operation", "id")
	case gorm.ErrModelValueRequired:
		return errors.NewValidationError("Model value required for operation", "model")
	case gorm.ErrInvalidData:
		return errors.NewValidationError("Invalid data provided", "data")
	case gorm.ErrUnsupportedDriver:
		return errors.NewInternalError("Unsupported database driver", err)
	case gorm.ErrRegistered:
		return errors.NewInternalError("Database component already registered", err)
	case gorm.ErrInvalidField:
		return errors.NewValidationError("Invalid database field", "field")
	case gorm.ErrEmptySlice:
		return errors.NewValidationError("Empty slice provided to database operation", "data")
	case gorm.ErrDryRunModeUnsupported:
		return errors.NewInternalError("Dry run mode not supported", err)
	default:
		// Check if it's a SQL error
		if sqlErr, ok := err.(*pq.Error); ok {
			return errors.Wrap(err, "DATABASE_SQL_ERROR", sqlErr.Error(), 500)
		}
		
		// Check for common database error patterns
		errStr := err.Error()
		if contains(errStr, "duplicate key") || contains(errStr, "unique constraint") {
			return errors.ErrDuplicateRecord
		}
		if contains(errStr, "foreign key constraint") {
			return errors.NewValidationError("Foreign key constraint violation", "reference")
		}
		if contains(errStr, "connection refused") || contains(errStr, "timeout") {
			return errors.ErrDatabaseConnection
		}
		
		return errors.NewInternalError("Database operation failed", err)
	}
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && 
		   (s == substr || 
		    len(s) > len(substr) && 
		    (stringContains(s, substr)))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// PaginationParams represents pagination parameters
type PaginationParams struct {
	Page     int `json:"page" form:"page"`
	PageSize int `json:"page_size" form:"page_size"`
	Offset   int `json:"-"`
}

// Validate validates pagination parameters
func (p *PaginationParams) Validate() error {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.PageSize < 1 {
		p.PageSize = 20
	}
	if p.PageSize > 100 {
		p.PageSize = 100
	}
	
	p.Offset = (p.Page - 1) * p.PageSize
	return nil
}

// PaginationResult represents paginated query results
type PaginationResult struct {
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
	Data     interface{} `json:"data"`
}

// Paginate performs paginated query with error handling
func (du *DatabaseUtils) Paginate(ctx context.Context, query *gorm.DB, params *PaginationParams, result interface{}) (*PaginationResult, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}

	var total int64
	var countErr error

	// Get total count with retry
	err := recovery.Retry(ctx, du.retryConfig, func(ctx context.Context) error {
		countErr = query.WithContext(ctx).Count(&total).Error
		return HandleGormError(countErr)
	})
	
	if err != nil {
		return nil, err
	}

	// Get paginated data with retry
	err = recovery.Retry(ctx, du.retryConfig, func(ctx context.Context) error {
		findErr := query.WithContext(ctx).
			Offset(params.Offset).
			Limit(params.PageSize).
			Find(result).Error
		return HandleGormError(findErr)
	})
	
	if err != nil {
		return nil, err
	}

	return &PaginationResult{
		Total:    total,
		Page:     params.Page,
		PageSize: params.PageSize,
		Data:     result,
	}, nil
}