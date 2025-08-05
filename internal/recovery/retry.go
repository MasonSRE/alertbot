package recovery

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/sirupsen/logrus"
)

// RetryConfig holds configuration for retry logic
type RetryConfig struct {
	MaxAttempts     int
	InitialDelay    time.Duration
	MaxDelay        time.Duration
	BackoffFactor   float64
	Jitter          bool
	RetryCondition  func(error) bool
	Logger          *logrus.Logger
}

// DefaultRetryConfig returns a default retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:   3,
		InitialDelay:  time.Second,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 2.0,
		Jitter:        true,
		RetryCondition: func(err error) bool {
			// Retry on any error by default
			return err != nil
		},
		Logger: logrus.New(),
	}
}

// Retry executes a function with retry logic
func Retry(ctx context.Context, config RetryConfig, fn func(context.Context) error) error {
	var lastErr error
	
	for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
		// Execute the function
		err := fn(ctx)
		if err == nil {
			// Success
			if attempt > 1 {
				config.Logger.WithFields(logrus.Fields{
					"attempt": attempt,
					"success": true,
				}).Info("Retry succeeded")
			}
			return nil
		}

		lastErr = err

		// Check if we should retry
		if !config.RetryCondition(err) {
			config.Logger.WithFields(logrus.Fields{
				"attempt": attempt,
				"error":   err.Error(),
				"reason":  "retry condition not met",
			}).Debug("Not retrying due to retry condition")
			return err
		}

		// Don't sleep after the last attempt
		if attempt == config.MaxAttempts {
			break
		}

		// Calculate delay
		delay := calculateDelay(config, attempt)

		config.Logger.WithFields(logrus.Fields{
			"attempt":    attempt,
			"error":      err.Error(),
			"next_delay": delay,
		}).Warn("Retrying after error")

		// Wait before next attempt
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	config.Logger.WithFields(logrus.Fields{
		"max_attempts": config.MaxAttempts,
		"final_error":  lastErr.Error(),
	}).Error("All retry attempts failed")

	return fmt.Errorf("all %d retry attempts failed, last error: %w", config.MaxAttempts, lastErr)
}

// calculateDelay calculates the delay for the next retry attempt
func calculateDelay(config RetryConfig, attempt int) time.Duration {
	// Exponential backoff
	delay := time.Duration(float64(config.InitialDelay) * math.Pow(config.BackoffFactor, float64(attempt-1)))
	
	// Cap at max delay
	if delay > config.MaxDelay {
		delay = config.MaxDelay
	}

	// Add jitter to prevent thundering herd
	if config.Jitter {
		jitterRange := float64(delay) * 0.1 // 10% jitter
		jitter := time.Duration(rand.Float64() * jitterRange)
		if rand.Intn(2) == 0 {
			delay += jitter
		} else {
			delay -= jitter
		}
	}

	return delay
}

// RetryWithCircuitBreaker combines retry logic with circuit breaker
func RetryWithCircuitBreaker(
	ctx context.Context,
	retryConfig RetryConfig,
	circuitBreaker *CircuitBreaker,
	fn func(context.Context) error,
) error {
	return Retry(ctx, retryConfig, func(ctx context.Context) error {
		return circuitBreaker.Execute(ctx, fn)
	})
}

// IsRetryable checks if an error is retryable based on common patterns
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	
	// Common retryable errors
	retryablePatterns := []string{
		"connection refused",
		"timeout",
		"temporary failure",
		"service unavailable",
		"rate limit",
		"too many requests",
		"circuit breaker",
		"network error",
	}

	for _, pattern := range retryablePatterns {
		if contains(errStr, pattern) {
			return true
		}
	}

	return false
}

// IsTemporaryError checks if an error is temporary
func IsTemporaryError(err error) bool {
	type temporary interface {
		Temporary() bool
	}

	if t, ok := err.(temporary); ok {
		return t.Temporary()
	}

	return IsRetryable(err)
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