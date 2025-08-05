package errors

import (
	"errors"
	"fmt"
	"net/http"
)

// Custom error types for AlertBot
var (
	// Authentication errors
	ErrUnauthorized     = errors.New("unauthorized access")
	ErrInvalidToken     = errors.New("invalid or expired token")
	ErrInsufficientPermissions = errors.New("insufficient permissions")

	// Validation errors
	ErrInvalidInput     = errors.New("invalid input data")
	ErrMissingRequired  = errors.New("missing required field")
	ErrInvalidFormat    = errors.New("invalid data format")

	// Database errors
	ErrRecordNotFound   = errors.New("record not found")
	ErrDuplicateRecord  = errors.New("duplicate record")
	ErrDatabaseConnection = errors.New("database connection failed")

	// Business logic errors
	ErrAlertNotFound    = errors.New("alert not found")
	ErrRuleNotFound     = errors.New("routing rule not found")
	ErrChannelNotFound  = errors.New("notification channel not found")
	ErrSilenceNotFound  = errors.New("silence not found")

	// External service errors
	ErrNotificationFailed = errors.New("notification delivery failed")
	ErrWebhookFailed    = errors.New("webhook delivery failed")
	ErrExternalService  = errors.New("external service error")

	// System errors
	ErrConfigInvalid    = errors.New("invalid configuration")
	ErrServiceUnavailable = errors.New("service temporarily unavailable")
	ErrRateLimited      = errors.New("rate limit exceeded")
)

// AppError represents an application error with additional context
type AppError struct {
	Type       string                 `json:"type"`
	Code       string                 `json:"code"`
	Message    string                 `json:"message"`
	Details    map[string]interface{} `json:"details,omitempty"`
	Cause      error                  `json:"-"`
	HTTPStatus int                    `json:"-"`
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

// Unwrap returns the underlying error
func (e *AppError) Unwrap() error {
	return e.Cause
}

// New creates a new AppError
func New(code, message string, httpStatus int) *AppError {
	return &AppError{
		Type:       "application_error",
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
		Details:    make(map[string]interface{}),
	}
}

// Wrap wraps an existing error with additional context
func Wrap(err error, code, message string, httpStatus int) *AppError {
	return &AppError{
		Type:       "application_error",
		Code:       code,
		Message:    message,
		Cause:      err,
		HTTPStatus: httpStatus,
		Details:    make(map[string]interface{}),
	}
}

// WithDetails adds details to an error
func (e *AppError) WithDetails(key string, value interface{}) *AppError {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	e.Details[key] = value
	return e
}

// WithField adds a single field to error details
func (e *AppError) WithField(key string, value interface{}) *AppError {
	return e.WithDetails(key, value)
}

// Predefined error constructors
func NewValidationError(message string, field string) *AppError {
	return New("VALIDATION_ERROR", message, http.StatusBadRequest).
		WithField("field", field)
}

func NewNotFoundError(resource string, id interface{}) *AppError {
	return New("NOT_FOUND", fmt.Sprintf("%s not found", resource), http.StatusNotFound).
		WithField("resource", resource).
		WithField("id", id)
}

func NewUnauthorizedError(message string) *AppError {
	if message == "" {
		message = "Unauthorized access"
	}
	return New("UNAUTHORIZED", message, http.StatusUnauthorized)
}

func NewForbiddenError(message string) *AppError {
	if message == "" {
		message = "Access forbidden"
	}
	return New("FORBIDDEN", message, http.StatusForbidden)
}

func NewConflictError(message string) *AppError {
	return New("CONFLICT", message, http.StatusConflict)
}

func NewInternalError(message string, cause error) *AppError {
	if message == "" {
		message = "Internal server error"
	}
	return Wrap(cause, "INTERNAL_ERROR", message, http.StatusInternalServerError)
}

func NewServiceUnavailableError(message string) *AppError {
	if message == "" {
		message = "Service temporarily unavailable"
	}
	return New("SERVICE_UNAVAILABLE", message, http.StatusServiceUnavailable)
}

func NewRateLimitError() *AppError {
	return New("RATE_LIMITED", "Too many requests, please try again later", http.StatusTooManyRequests)
}

func NewBadRequestError(message string) *AppError {
	return New("BAD_REQUEST", message, http.StatusBadRequest)
}

// Error classification helpers
func IsValidationError(err error) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code == "VALIDATION_ERROR"
	}
	return false
}

func IsNotFoundError(err error) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code == "NOT_FOUND"
	}
	return errors.Is(err, ErrRecordNotFound) ||
		   errors.Is(err, ErrAlertNotFound) ||
		   errors.Is(err, ErrRuleNotFound) ||
		   errors.Is(err, ErrChannelNotFound) ||
		   errors.Is(err, ErrSilenceNotFound)
}

func IsAuthenticationError(err error) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code == "UNAUTHORIZED" || appErr.Code == "FORBIDDEN"
	}
	return errors.Is(err, ErrUnauthorized) ||
		   errors.Is(err, ErrInvalidToken) ||
		   errors.Is(err, ErrInsufficientPermissions)
}

func IsInternalError(err error) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code == "INTERNAL_ERROR"
	}
	return false
}

// GetHTTPStatus returns the HTTP status code for an error
func GetHTTPStatus(err error) int {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.HTTPStatus
	}

	// Default mappings for common errors
	switch {
	case IsNotFoundError(err):
		return http.StatusNotFound
	case IsAuthenticationError(err):
		return http.StatusUnauthorized
	case IsValidationError(err):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

// ToResponse converts an error to a JSON response structure
func ToResponse(err error) map[string]interface{} {
	var appErr *AppError
	if errors.As(err, &appErr) {
		response := map[string]interface{}{
			"success": false,
			"error": map[string]interface{}{
				"code":    appErr.Code,
				"message": appErr.Message,
			},
		}
		
		if len(appErr.Details) > 0 {
			response["error"].(map[string]interface{})["details"] = appErr.Details
		}
		
		return response
	}

	// Generic error response
	return map[string]interface{}{
		"success": false,
		"error": map[string]interface{}{
			"code":    "UNKNOWN_ERROR",
			"message": err.Error(),
		},
	}
}