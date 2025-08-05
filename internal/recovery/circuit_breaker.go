package recovery

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// CircuitState represents the state of a circuit breaker
type CircuitState int

const (
	// StateClosed - normal operation, requests allowed
	StateClosed CircuitState = iota
	// StateOpen - circuit is open, requests are rejected
	StateOpen
	// StateHalfOpen - circuit is in testing mode
	StateHalfOpen
)

func (s CircuitState) String() string {
	switch s {
	case StateClosed:
		return "CLOSED"
	case StateOpen:
		return "OPEN"
	case StateHalfOpen:
		return "HALF_OPEN"
	default:
		return "UNKNOWN"
	}
}

// CircuitBreaker implements the circuit breaker pattern for fault tolerance
type CircuitBreaker struct {
	name           string
	maxFailures    int
	resetTimeout   time.Duration
	failureCount   int
	lastFailureTime time.Time
	state          CircuitState
	mutex          sync.RWMutex
	logger         *logrus.Logger

	// Callbacks
	onStateChange func(name string, from, to CircuitState)
}

// CircuitBreakerConfig holds configuration for circuit breaker
type CircuitBreakerConfig struct {
	Name         string
	MaxFailures  int
	ResetTimeout time.Duration
	Logger       *logrus.Logger
	OnStateChange func(name string, from, to CircuitState)
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(config CircuitBreakerConfig) *CircuitBreaker {
	if config.MaxFailures <= 0 {
		config.MaxFailures = 5
	}
	if config.ResetTimeout <= 0 {
		config.ResetTimeout = 60 * time.Second
	}
	if config.Logger == nil {
		config.Logger = logrus.New()
	}

	return &CircuitBreaker{
		name:          config.Name,
		maxFailures:   config.MaxFailures,
		resetTimeout:  config.ResetTimeout,
		state:         StateClosed,
		logger:        config.Logger,
		onStateChange: config.OnStateChange,
	}
}

// Execute executes a function with circuit breaker protection
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func(context.Context) error) error {
	// Check if circuit is open
	if !cb.allowRequest() {
		return errors.New("circuit breaker is open")
	}

	// Execute the function
	err := fn(ctx)

	// Record the result
	cb.recordResult(err == nil)

	return err
}

// allowRequest checks if the request should be allowed
func (cb *CircuitBreaker) allowRequest() bool {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		// Check if we should transition to half-open
		if time.Since(cb.lastFailureTime) > cb.resetTimeout {
			cb.setState(StateHalfOpen)
			return true
		}
		return false
	case StateHalfOpen:
		return true
	default:
		return false
	}
}

// recordResult records the result of a request
func (cb *CircuitBreaker) recordResult(success bool) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	if success {
		cb.onSuccess()
	} else {
		cb.onFailure()
	}
}

// onSuccess handles successful requests
func (cb *CircuitBreaker) onSuccess() {
	cb.failureCount = 0

	if cb.state == StateHalfOpen {
		cb.setState(StateClosed)
	}
}

// onFailure handles failed requests
func (cb *CircuitBreaker) onFailure() {
	cb.failureCount++
	cb.lastFailureTime = time.Now()

	if cb.state == StateHalfOpen {
		// Failed in half-open, go back to open
		cb.setState(StateOpen)
	} else if cb.failureCount >= cb.maxFailures {
		// Too many failures, open the circuit
		cb.setState(StateOpen)
	}
}

// setState changes the circuit breaker state
func (cb *CircuitBreaker) setState(newState CircuitState) {
	if cb.state == newState {
		return
	}

	oldState := cb.state
	cb.state = newState

	cb.logger.WithFields(logrus.Fields{
		"circuit_breaker": cb.name,
		"from_state":     oldState.String(),
		"to_state":       newState.String(),
		"failure_count":  cb.failureCount,
	}).Info("Circuit breaker state changed")

	// Call state change callback if provided
	if cb.onStateChange != nil {
		go cb.onStateChange(cb.name, oldState, newState)
	}
}

// GetState returns the current state of the circuit breaker
func (cb *CircuitBreaker) GetState() CircuitState {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.state
}

// GetFailureCount returns the current failure count
func (cb *CircuitBreaker) GetFailureCount() int {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.failureCount
}

// Reset manually resets the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	cb.failureCount = 0
	cb.setState(StateClosed)
}

// ForceOpen manually opens the circuit breaker
func (cb *CircuitBreaker) ForceOpen() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	cb.setState(StateOpen)
	cb.lastFailureTime = time.Now()
}

// GetStats returns circuit breaker statistics
func (cb *CircuitBreaker) GetStats() map[string]interface{} {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()

	return map[string]interface{}{
		"name":              cb.name,
		"state":             cb.state.String(),
		"failure_count":     cb.failureCount,
		"max_failures":      cb.maxFailures,
		"last_failure_time": cb.lastFailureTime,
		"reset_timeout":     cb.resetTimeout,
	}
}